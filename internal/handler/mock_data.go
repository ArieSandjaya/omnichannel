package handler

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ariesandjaya/omnichannel/internal/domain"
)

var (
	mockTenantID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	mockUserID   = uuid.MustParse("00000000-0000-0000-0000-000000000002")
)

// MockTenants holds demo tenant data for development/preview mode.
var MockTenants = []*domain.Tenant{
	{
		ID:                 mockTenantID,
		Name:               "Toko Batik Nusantara",
		Slug:               "batik-nusantara",
		BusinessType:       "retail",
		SubscriptionPlan:   "pro",
		SubscriptionStatus: "active",
		LogoURL:            "/static/img/logo.svg",
		Currency:           "IDR",
		Timezone:           "Asia/Jakarta",
		Settings: domain.TenantSettings{
			Theme: domain.TenantTheme{
				PrimaryColor:   "#c2410c",
				SecondaryColor: "#ea580c",
				HeroText:       "Koleksi Batik Terbaik dari Seluruh Nusantara",
				FontFamily:     "Inter, sans-serif",
			},
			Store: domain.TenantStore{
				Address: "Jl. Batik Indah No. 17, Yogyakarta",
				Phone:   "0274-555-1234",
				Email:   "hello@batiknusantara.id",
			},
		},
		CreatedAt: time.Now().AddDate(-1, 0, 0),
	},
	{
		ID:                 uuid.MustParse("00000000-0000-0000-0000-000000000003"),
		Name:               "Kedai Kopi Arabika",
		Slug:               "kopi-arabika",
		BusinessType:       "fnb",
		SubscriptionPlan:   "starter",
		SubscriptionStatus: "active",
		LogoURL:            "/static/img/logo.svg",
		Currency:           "IDR",
		Timezone:           "Asia/Jakarta",
		Settings: domain.TenantSettings{
			Theme: domain.TenantTheme{
				PrimaryColor:   "#6366f1",
				SecondaryColor: "#8b5cf6",
				HeroText:       "Kopi Premium untuk Hari yang Lebih Baik",
				FontFamily:     "Inter, sans-serif",
			},
			Store: domain.TenantStore{
				Address: "Jl. Kopi Hijau No. 5, Bandung",
				Phone:   "022-555-9876",
				Email:   "hello@kopiarabika.id",
			},
		},
		CreatedAt: time.Now().AddDate(0, -6, 0),
	},
}

// MockUsers holds demo user data.
var MockUsers = []*domain.User{
	{
		ID:        mockUserID,
		TenantID:  mockTenantID,
		Email:     "owner@batiknusantara.id",
		FullName:  "Budi Santoso",
		Role:      domain.RoleOwner,
		AvatarURL: "",
		IsActive:  true,
		CreatedAt: time.Now().AddDate(-1, 0, 0),
	},
}

// MockProducts holds demo product catalog.
var MockProducts = []domain.Product{
	{ID: uuid.New(), TenantID: mockTenantID, SKU: "BTK-001", Name: "Batik Tulis Parang Klasik", Price: 450000, StockQuantity: 12, MinStockLevel: 3, Unit: "pcs", IsActive: true, CreatedAt: time.Now()},
	{ID: uuid.New(), TenantID: mockTenantID, SKU: "BTK-002", Name: "Batik Cap Mega Mendung", Price: 185000, StockQuantity: 3, MinStockLevel: 5, Unit: "pcs", IsActive: true, CreatedAt: time.Now()},
	{ID: uuid.New(), TenantID: mockTenantID, SKU: "BTK-003", Name: "Batik Jumputan Pelangi", Price: 225000, StockQuantity: 0, MinStockLevel: 3, Unit: "pcs", IsActive: true, CreatedAt: time.Now()},
	{ID: uuid.New(), TenantID: mockTenantID, SKU: "BTK-004", Name: "Kain Batik Sido Mukti", Price: 320000, StockQuantity: 8, MinStockLevel: 3, Unit: "meter", IsActive: true, CreatedAt: time.Now()},
	{ID: uuid.New(), TenantID: mockTenantID, SKU: "BTK-005", Name: "Batik Kawung Modern", Price: 275000, StockQuantity: 15, MinStockLevel: 3, Unit: "pcs", IsActive: true, CreatedAt: time.Now()},
	{ID: uuid.New(), TenantID: mockTenantID, SKU: "BTK-006", Name: "Batik Sogan Jawa Klasik", Price: 550000, StockQuantity: 2, MinStockLevel: 3, Unit: "pcs", IsActive: true, CreatedAt: time.Now()},
	{ID: uuid.New(), TenantID: mockTenantID, SKU: "BTK-007", Name: "Batik Printing Lereng", Price: 95000, StockQuantity: 25, MinStockLevel: 5, Unit: "pcs", IsActive: true, CreatedAt: time.Now()},
	{ID: uuid.New(), TenantID: mockTenantID, SKU: "BTK-008", Name: "Selendang Batik Sutera", Price: 380000, StockQuantity: 5, MinStockLevel: 3, Unit: "pcs", IsActive: true, CreatedAt: time.Now()},
	{ID: uuid.New(), TenantID: mockTenantID, SKU: "BTK-009", Name: "Kemeja Batik Pria Modern", Price: 215000, StockQuantity: 1, MinStockLevel: 3, Unit: "pcs", IsActive: true, CreatedAt: time.Now()},
	{ID: uuid.New(), TenantID: mockTenantID, SKU: "BTK-010", Name: "Dress Batik Wanita Elegan", Price: 475000, StockQuantity: 7, MinStockLevel: 3, Unit: "pcs", IsActive: true, CreatedAt: time.Now()},
}

// MockOrders holds demo recent orders.
var MockOrders = []domain.Order{
	{ID: uuid.New(), TenantID: mockTenantID, OrderNumber: "ORD-20260517-001", Channel: domain.ChannelPOS, Status: domain.OrderStatusPaid, PaymentMethod: domain.PaymentTunai, TotalAmount: 450000, CreatedAt: time.Now().Add(-30 * time.Minute)},
	{ID: uuid.New(), TenantID: mockTenantID, OrderNumber: "ORD-20260517-002", Channel: domain.ChannelShopee, Status: domain.OrderStatusShipped, PaymentMethod: domain.PaymentTransfer, TotalAmount: 185000, CreatedAt: time.Now().Add(-2 * time.Hour)},
	{ID: uuid.New(), TenantID: mockTenantID, OrderNumber: "ORD-20260517-003", Channel: domain.ChannelTokopedia, Status: domain.OrderStatusPending, PaymentMethod: domain.PaymentQRIS, TotalAmount: 820000, CreatedAt: time.Now().Add(-3 * time.Hour)},
	{ID: uuid.New(), TenantID: mockTenantID, OrderNumber: "ORD-20260517-004", Channel: domain.ChannelPOS, Status: domain.OrderStatusCompleted, PaymentMethod: domain.PaymentQRIS, TotalAmount: 275000, CreatedAt: time.Now().Add(-4 * time.Hour)},
	{ID: uuid.New(), TenantID: mockTenantID, OrderNumber: "ORD-20260516-018", Channel: domain.ChannelTikTok, Status: domain.OrderStatusPaid, PaymentMethod: domain.PaymentTransfer, TotalAmount: 550000, CreatedAt: time.Now().Add(-26 * time.Hour)},
}

// MockDashboardStats returns computed stats from mock data.
func MockDashboardStats() DashboardStats {
	var revenueToday int64
	ordersToday := 0
	for _, o := range MockOrders {
		if time.Since(o.CreatedAt) < 24*time.Hour {
			revenueToday += o.TotalAmount
			ordersToday++
		}
	}

	lowStockCount := 0
	for _, p := range MockProducts {
		if p.IsLowStock() {
			lowStockCount++
		}
	}

	return DashboardStats{
		RevenueToday:  formatMockRupiah(revenueToday),
		OrdersToday:   ordersToday,
		TotalProducts: len(MockProducts),
		LowStockCount: lowStockCount,
	}
}

// MockTenantBySlug returns a tenant matching the slug, or the first as default.
func MockTenantBySlug(slug string) *domain.Tenant {
	for _, t := range MockTenants {
		if t.Slug == slug {
			return t
		}
	}
	return MockTenants[0]
}

// LowStockProducts returns products at or below their minimum stock level.
func LowStockProducts() []domain.Product {
	var result []domain.Product
	for _, p := range MockProducts {
		if p.IsLowStock() {
			result = append(result, p)
		}
	}
	return result
}

func formatMockRupiah(amount int64) string {
	if amount == 0 {
		return "Rp 0"
	}
	s := fmt.Sprintf("%d", amount)
	result := ""
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result += "."
		}
		result += string(c)
	}
	return "Rp " + result
}
