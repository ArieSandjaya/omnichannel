package dto

type ShippingItem struct {
	Name        string `json:"name"        binding:"required"`
	Description string `json:"description"`
	Value       int    `json:"value"       binding:"required"`
	Weight      int    `json:"weight"      binding:"required"` // grams
	Quantity    int    `json:"quantity"    binding:"required"`
}

type GetRatesRequest struct {
	OriginPostalCode   string         `json:"origin_postal_code"      binding:"required"`
	OriginLatitude     string         `json:"origin_latitude"`
	OriginLongitude    string         `json:"origin_longitude"`
	DestPostalCode     string         `json:"destination_postal_code" binding:"required"`
	DestLatitude       string         `json:"destination_latitude"`
	DestLongitude      string         `json:"destination_longitude"`
	Items              []ShippingItem `json:"items"                   binding:"required,min=1"`
	Couriers           string         `json:"couriers"` // e.g. "jne,jnt,sicepat"
}

type CourierRate struct {
	CourierCode    string `json:"courier_code"`
	CourierName    string `json:"courier_name"`
	CourierService string `json:"courier_service"`
	Price          int    `json:"price"`
	EstimatedDays  string `json:"estimated_days"`
	Description    string `json:"description,omitempty"`
}

type GetRatesResponse struct {
	Success  bool          `json:"success"`
	Message  string        `json:"message"`
	Couriers []CourierRate `json:"couriers"`
}

type CreateShipmentRequest struct {
	CourierCode    string `json:"courier_code"    binding:"required"`
	CourierService string `json:"courier_service" binding:"required"`
}

type ShipmentResponse struct {
	BiteshipOrderID string `json:"biteship_order_id"`
	CourierCode     string `json:"courier_code"`
	CourierService  string `json:"courier_service"`
	TrackingNumber  string `json:"tracking_number,omitempty"`
	WaybillID       string `json:"waybill_id,omitempty"`
	Status          string `json:"status"`
	Price           int    `json:"price"`
}

type TrackEvent struct {
	Description string `json:"description"`
	Status      string `json:"status"`
	UpdatedAt   string `json:"updated_at"`
	Location    string `json:"location,omitempty"`
}

type TrackingResponse struct {
	WaybillID   string       `json:"waybill_id"`
	CourierCode string       `json:"courier_code"`
	Status      string       `json:"status"`
	History     []TrackEvent `json:"history"`
}
