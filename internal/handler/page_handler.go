package handler

import (
	"bytes"
	"net/http"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
	"github.com/ariesandjaya/omnichannel/internal/domain"
)

// DashboardStats is the data passed to the Dashboard page template.
type DashboardStats struct {
	RevenueToday  string
	OrdersToday   int
	TotalProducts int
	LowStockCount int
}

// ChannelStatus represents the sync status of a marketplace channel.
type ChannelStatus struct {
	Name      string
	Icon      string
	Connected bool
	LastSync  string
}

// renderPage writes a templ.Component to the Gin response writer.
func renderPage(c *gin.Context, status int, component templ.Component) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(status)
	if err := component.Render(c.Request.Context(), c.Writer); err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
	}
}

// renderPartial renders a templ component as an HTML fragment.
func renderPartial(c *gin.Context, component templ.Component) {
	var buf bytes.Buffer
	if err := component.Render(c.Request.Context(), &buf); err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, buf.String())
}

// getTenantAndUser extracts tenant and user from context (set by middleware).
func getTenantAndUser(c *gin.Context) (*domain.Tenant, *domain.User) {
	tenant := MockTenants[0]
	user := MockUsers[0]
	if v, ok := c.Get("tenant"); ok {
		if t, ok := v.(*domain.Tenant); ok {
			tenant = t
		}
	}
	if v, ok := c.Get("user"); ok {
		if u, ok := v.(*domain.User); ok {
			user = u
		}
	}
	return tenant, user
}

// mockChannelStatuses returns demo channel connection statuses.
func mockChannelStatuses() []ChannelStatus {
	return []ChannelStatus{
		{Name: "Shopee", Connected: true, LastSync: "5 menit lalu"},
		{Name: "Tokopedia", Connected: true, LastSync: "12 menit lalu"},
		{Name: "TikTok Shop", Connected: false, LastSync: "Belum terhubung"},
	}
}
