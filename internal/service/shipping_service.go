package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"

	"github.com/ariesandjaya/omnichannel/internal/config"
	"github.com/ariesandjaya/omnichannel/internal/domain"
	"github.com/ariesandjaya/omnichannel/internal/dto"
)

// ShippingQuerier is the DB interface required by ShippingService.
type ShippingQuerier interface {
	GetOrderByIDRaw(ctx context.Context, id, tenantID uuid.UUID) (OrderRow, error)
	GetOrderByBiteshipOrderIDRaw(ctx context.Context, biteshipOrderID string) (OrderRow, error)
	UpdateShippingInfoRaw(ctx context.Context, id, tenantID uuid.UUID, info json.RawMessage) error
}

// ShippingService calls Biteship for rate queries, shipment creation, and tracking.
type ShippingService struct {
	queries ShippingQuerier
	client  *resty.Client
	cfg     *config.Config
}

func NewShippingService(queries ShippingQuerier, c *resty.Client, cfg *config.Config) *ShippingService {
	return &ShippingService{queries: queries, client: c, cfg: cfg}
}

// ── Biteship API response shapes ─────────────────────────────────────────────

type biteshipRatesResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Pricing []struct {
		CourierCode    string `json:"courier_code"`
		CourierName    string `json:"courier_name"`
		CourierService string `json:"courier_service_code"`
		Price          int    `json:"price"`
		MinDay         int    `json:"shipment_duration_range"`
		MaxDay         int    `json:"shipment_duration_range_max"`
		Description    string `json:"description"`
	} `json:"pricing"`
}

type biteshipOrderResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	ID       string `json:"id"`
	Couriers []struct {
		TrackingID    string `json:"tracking_id"`
		WaybillID     string `json:"waybill_id"`
		CourierCode   string `json:"courier_code"`
		CourierService string `json:"courier_service_code"`
		Price         int    `json:"price"`
		Status        string `json:"status"`
		EstimatedDate string `json:"estimated_shipment_date"`
	} `json:"couriers"`
}

type biteshipTrackingResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Courier struct {
		Company   string `json:"company"`
		WaybillID string `json:"waybill_id"`
	} `json:"courier"`
	Status  string `json:"status"`
	History []struct {
		Note      string `json:"note"`
		Status    string `json:"status"`
		UpdatedAt string `json:"updated_at"`
		Location  string `json:"service_location_name"`
	} `json:"link"`
}

// GetRates fetches shipping rates from Biteship.
func (s *ShippingService) GetRates(ctx context.Context, req *dto.GetRatesRequest) (*dto.GetRatesResponse, error) {
	body := map[string]any{
		"origin_postal_code":      req.OriginPostalCode,
		"destination_postal_code": req.DestPostalCode,
		"couriers":                req.Couriers,
		"items":                   req.Items,
	}
	if req.OriginLatitude != "" {
		body["origin_latitude"] = req.OriginLatitude
		body["origin_longitude"] = req.OriginLongitude
	}
	if req.DestLatitude != "" {
		body["destination_latitude"] = req.DestLatitude
		body["destination_longitude"] = req.DestLongitude
	}

	var bResp biteshipRatesResponse
	resp, err := s.client.R().
		SetContext(ctx).
		SetBody(body).
		SetResult(&bResp).
		Post("/rates/couriers")
	if err != nil {
		return nil, fmt.Errorf("biteship get rates: %w", err)
	}
	if resp.IsError() {
		return nil, fmt.Errorf("biteship rates HTTP %d: %s", resp.StatusCode(), resp.String())
	}

	rates := make([]dto.CourierRate, 0, len(bResp.Pricing))
	for _, p := range bResp.Pricing {
		rates = append(rates, dto.CourierRate{
			CourierCode:    p.CourierCode,
			CourierName:    p.CourierName,
			CourierService: p.CourierService,
			Price:          p.Price,
			EstimatedDays:  fmt.Sprintf("%d-%d hari", p.MinDay, p.MaxDay),
			Description:    p.Description,
		})
	}
	return &dto.GetRatesResponse{Success: bResp.Success, Message: bResp.Message, Couriers: rates}, nil
}

// CreateShipment creates a Biteship order and stores the result in orders.shipping_info.
func (s *ShippingService) CreateShipment(
	ctx context.Context,
	tenantID, orderID uuid.UUID,
	req *dto.CreateShipmentRequest,
) (*dto.ShipmentResponse, error) {
	order, err := s.queries.GetOrderByIDRaw(ctx, orderID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("order not found: %w", err)
	}

	var lineItems []domain.LineItem
	if err := json.Unmarshal(order.LineItems, &lineItems); err != nil {
		return nil, fmt.Errorf("parse line_items: %w", err)
	}

	biteshipItems := make([]map[string]any, 0, len(lineItems))
	for _, item := range lineItems {
		biteshipItems = append(biteshipItems, map[string]any{
			"name":     item.Name,
			"value":    item.Price,
			"quantity": item.Quantity,
		})
	}

	var bResp biteshipOrderResponse
	resp, err := s.client.R().
		SetContext(ctx).
		SetBody(map[string]any{
			"shipper_contact_name":  "Omnichannel Store",
			"receiver_contact_name": order.CustomerName,
			"courier_company":       req.CourierCode,
			"courier_type":          req.CourierService,
			"delivery_type":         "now",
			"items":                 biteshipItems,
		}).
		SetResult(&bResp).
		Post("/orders")
	if err != nil {
		return nil, fmt.Errorf("biteship create shipment: %w", err)
	}
	if resp.IsError() {
		return nil, fmt.Errorf("biteship create shipment HTTP %d: %s", resp.StatusCode(), resp.String())
	}

	shipInfo := domain.ShippingInfo{
		BiteshipOrderID: bResp.ID,
		CourierCode:     req.CourierCode,
		CourierService:  req.CourierService,
		Status:          "confirmed",
	}
	price := 0
	if len(bResp.Couriers) > 0 {
		c := bResp.Couriers[0]
		shipInfo.TrackingNumber = c.TrackingID
		shipInfo.WaybillID = c.WaybillID
		shipInfo.Status = c.Status
		shipInfo.EstimatedDays = c.EstimatedDate
		shipInfo.Price = int64(c.Price)
		price = c.Price
	}

	shipInfoJSON, _ := json.Marshal(shipInfo)
	if err := s.queries.UpdateShippingInfoRaw(ctx, orderID, tenantID, shipInfoJSON); err != nil {
		return nil, fmt.Errorf("persist shipping info: %w", err)
	}

	slog.Info("shipment created",
		"biteship_order_id", bResp.ID,
		"courier", req.CourierCode,
		"order_id", orderID,
	)

	return &dto.ShipmentResponse{
		BiteshipOrderID: bResp.ID,
		CourierCode:     req.CourierCode,
		CourierService:  req.CourierService,
		TrackingNumber:  shipInfo.TrackingNumber,
		WaybillID:       shipInfo.WaybillID,
		Status:          shipInfo.Status,
		Price:           price,
	}, nil
}

// GetTrackingInfo fetches shipment tracking history from Biteship.
func (s *ShippingService) GetTrackingInfo(ctx context.Context, waybillID, courierCode string) (*dto.TrackingResponse, error) {
	var bResp biteshipTrackingResponse
	resp, err := s.client.R().
		SetContext(ctx).
		SetQueryParam("courier_company", courierCode).
		SetResult(&bResp).
		Get(fmt.Sprintf("/trackings/%s", waybillID))
	if err != nil {
		return nil, fmt.Errorf("biteship tracking: %w", err)
	}
	if resp.IsError() {
		return nil, fmt.Errorf("biteship tracking HTTP %d: %s", resp.StatusCode(), resp.String())
	}

	history := make([]dto.TrackEvent, 0, len(bResp.History))
	for _, h := range bResp.History {
		history = append(history, dto.TrackEvent{
			Description: h.Note,
			Status:      h.Status,
			UpdatedAt:   h.UpdatedAt,
			Location:    h.Location,
		})
	}

	return &dto.TrackingResponse{
		WaybillID:   waybillID,
		CourierCode: bResp.Courier.Company,
		Status:      bResp.Status,
		History:     history,
	}, nil
}

// UpdateShippingStatus is called by the Biteship webhook worker.
func (s *ShippingService) UpdateShippingStatus(
	ctx context.Context,
	biteshipOrderID, status, waybillID string,
) error {
	order, err := s.queries.GetOrderByBiteshipOrderIDRaw(ctx, biteshipOrderID)
	if err != nil {
		return fmt.Errorf("order lookup by biteship_order_id %q: %w", biteshipOrderID, err)
	}

	var shipInfo domain.ShippingInfo
	if order.ShippingInfo != nil {
		_ = json.Unmarshal(order.ShippingInfo, &shipInfo)
	}
	shipInfo.Status = status
	if waybillID != "" {
		shipInfo.WaybillID = waybillID
	}

	shipInfoJSON, _ := json.Marshal(shipInfo)
	if err := s.queries.UpdateShippingInfoRaw(ctx, order.ID, order.TenantID, shipInfoJSON); err != nil {
		return fmt.Errorf("update shipping status: %w", err)
	}

	slog.Info("shipping status updated",
		"biteship_order_id", biteshipOrderID,
		"status", status,
		"order_id", order.ID,
	)
	return nil
}
