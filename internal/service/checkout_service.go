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

const taxRate = 0.11 // PPN 11%

type CheckoutService struct {
	db        *pgxpool.Pool
	products  repository.ProductRepository
	orders    repository.OrderRepository
	inventory repository.InventoryRepository
}

func NewCheckoutService(
	db *pgxpool.Pool,
	products repository.ProductRepository,
	orders repository.OrderRepository,
	inventory repository.InventoryRepository,
) *CheckoutService {
	return &CheckoutService{db: db, products: products, orders: orders, inventory: inventory}
}

// Process handles a web/e-commerce checkout atomically.
//
// Anti-overselling strategy:
//  1. Items are sorted by ProductID before locking to guarantee a consistent
//     lock-acquisition order across all concurrent transactions, preventing deadlocks.
//  2. Each product row is locked with SELECT FOR UPDATE inside the transaction.
//     A concurrent POS or web checkout requesting the same product will block
//     here until this transaction commits or rolls back.
//  3. Stock is validated AFTER locking, so the quantity read is authoritative.
func (s *CheckoutService) Process(ctx context.Context, req dto.CheckoutRequest, tenantID, userID uuid.UUID) (*dto.CheckoutResponse, error) {
	// Step 1: sort by ProductID to prevent deadlock between concurrent transactions
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

	// Step 2 & 3: lock each product row, then validate stock
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

	// Step 4: deduct stock and build line items
	orderID := uuid.New()
	var lineItems []domain.LineItem
	var subtotal float64
	channel := "website"
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
			ProductID: li.product.ID,
			SKU:       li.product.SKU,
			Name:      li.product.Name,
			Quantity:  li.quantity,
			UnitPrice: li.product.Price,
			Discount:  li.discount,
			Subtotal:  itemSubtotal,
		})
	}

	// Step 5: create order
	lineItemsJSON, _ := json.Marshal(lineItems)
	customerJSON, _ := json.Marshal(req.Customer)
	orderNumber, _ := s.orders.GenerateOrderNumber(ctx, tenantID, "website")
	taxAmount := subtotal * taxRate
	totalAmount := subtotal + taxAmount
	paymentMethod := req.PaymentMethod

	var notes *string
	if req.Notes != "" {
		notes = &req.Notes
	}

	order := &domain.Order{
		ID:             orderID,
		TenantID:       tenantID,
		OrderNumber:    orderNumber,
		Channel:        domain.ChannelWebsite,
		Status:         domain.OrderStatusPending,
		CustomerInfo:   customerJSON,
		LineItems:      lineItemsJSON,
		Subtotal:       subtotal,
		DiscountAmount: 0,
		TaxAmount:      taxAmount,
		TotalAmount:    totalAmount,
		PaymentMethod:  &paymentMethod,
		PaymentStatus:  domain.PaymentStatusUnpaid,
		Notes:          notes,
		CreatedBy:      &userID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.orders.Create(ctx, tx, order); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return &dto.CheckoutResponse{
		OrderID:       orderID,
		OrderNumber:   orderNumber,
		Subtotal:      subtotal,
		TaxAmount:     taxAmount,
		TotalAmount:   totalAmount,
		Status:        string(domain.OrderStatusPending),
		PaymentStatus: string(domain.PaymentStatusUnpaid),
	}, nil
}
