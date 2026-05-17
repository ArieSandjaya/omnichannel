package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/ariesandjaya/omnichannel/internal/domain"
	"github.com/ariesandjaya/omnichannel/internal/dto"
	"github.com/ariesandjaya/omnichannel/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type POSService struct {
	db        *pgxpool.Pool
	products  repository.ProductRepository
	orders    repository.OrderRepository
	inventory repository.InventoryRepository
}

func NewPOSService(
	db *pgxpool.Pool,
	products repository.ProductRepository,
	orders repository.OrderRepository,
	inventory repository.InventoryRepository,
) *POSService {
	return &POSService{db: db, products: products, orders: orders, inventory: inventory}
}

// ProcessTransaction handles a POS (cashier) sale atomically.
//
// Locking behaviour is identical to CheckoutService.Process:
//   - Items sorted by ProductID before locking (deadlock prevention)
//   - SELECT FOR UPDATE per product inside the transaction
//   - Cash payment → order immediately "completed" + payment "paid"
//   - QRIS payment → order "pending" + payment "unpaid" until webhook confirms
func (s *POSService) ProcessTransaction(ctx context.Context, req dto.POSTransactionRequest, tenantID, userID uuid.UUID) (*dto.POSTransactionResponse, error) {
	sort.Slice(req.Items, func(i, j int) bool {
		return req.Items[i].ProductID.String() < req.Items[j].ProductID.String()
	})

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	type lockedItem struct {
		product  *domain.Product
		quantity int
		discount float64
	}

	locked := make([]lockedItem, 0, len(req.Items))
	for _, item := range req.Items {
		p, err := s.products.GetByIDForUpdate(ctx, tx, tenantID, item.ProductID)
		if err != nil {
			return nil, err
		}
		if p.TrackInventory && p.StockQuantity < item.Quantity {
			return nil, fmt.Errorf(
				"stok '%s' tidak mencukupi: tersedia %d, diminta %d",
				p.Name, p.StockQuantity, item.Quantity,
			)
		}
		locked = append(locked, lockedItem{p, item.Quantity, item.Discount})
	}

	orderID := uuid.New()
	var lineItems []domain.LineItem
	var results []dto.POSItemResult
	var subtotal float64
	channel := "pos"
	refType := "order"

	for _, li := range locked {
		itemSubtotal := (li.product.Price * float64(li.quantity)) - li.discount
		subtotal += itemSubtotal

		newQty, err := s.products.DeductStock(ctx, tx, tenantID, li.product.ID, li.quantity)
		if err != nil {
			return nil, err
		}

		if err := s.inventory.CreateLog(ctx, tx, &domain.InventoryLog{
			ID:             uuid.New(),
			TenantID:       tenantID,
			ProductID:      li.product.ID,
			Type:           domain.ChangeTypeSale,
			QuantityChange: -li.quantity,
			QuantityBefore: li.product.StockQuantity,
			QuantityAfter:  newQty,
			ReferenceType:  &refType,
			ReferenceID:    &orderID,
			Channel:        &channel,
			CreatedBy:      &userID,
			CreatedAt:      time.Now(),
		}); err != nil {
			return nil, err
		}

		lineItems = append(lineItems, domain.LineItem{
			ProductID: li.product.ID, SKU: li.product.SKU, Name: li.product.Name,
			Quantity: li.quantity, UnitPrice: li.product.Price,
			Discount: li.discount, Subtotal: itemSubtotal,
		})
		results = append(results, dto.POSItemResult{
			ProductID: li.product.ID, Name: li.product.Name,
			Quantity: li.quantity, UnitPrice: li.product.Price,
			Subtotal: itemSubtotal, StockAfter: newQty,
		})
	}

	lineItemsJSON, _ := json.Marshal(lineItems)
	customerJSON, _ := json.Marshal(map[string]string{"name": req.CustomerName})
	orderNumber, _ := s.orders.GenerateOrderNumber(ctx, tenantID, "pos")
	paymentMethod := req.PaymentMethod

	// POS: cash/card → langsung selesai; QRIS → tunggu konfirmasi pembayaran
	paymentStatus := domain.PaymentStatusPaid
	orderStatus := domain.OrderStatusCompleted
	if req.PaymentMethod == "qris" {
		paymentStatus = domain.PaymentStatusUnpaid
		orderStatus = domain.OrderStatusPending
	}

	var notes *string
	if req.Notes != "" {
		notes = &req.Notes
	}

	order := &domain.Order{
		ID: orderID, TenantID: tenantID, OrderNumber: orderNumber,
		Channel: domain.ChannelPOS, Status: orderStatus,
		CustomerInfo: customerJSON, LineItems: lineItemsJSON,
		Subtotal: subtotal, TotalAmount: subtotal,
		PaymentMethod: &paymentMethod, PaymentStatus: paymentStatus,
		Notes: notes, CreatedBy: &userID,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}

	if err := s.orders.Create(ctx, tx, order); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	change := req.AmountPaid - subtotal
	if change < 0 {
		change = 0
	}
	return &dto.POSTransactionResponse{
		OrderID: orderID, OrderNumber: orderNumber,
		TotalAmount: subtotal, AmountPaid: req.AmountPaid, Change: change,
		PaymentStatus: string(paymentStatus), Items: results,
	}, nil
}
