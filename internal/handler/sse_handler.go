package handler

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ariesandjaya/omnichannel/internal/broker"
)

type SSEHandler struct {
	Broker *broker.SSEBroker
}

func NewSSEHandler(b *broker.SSEBroker) *SSEHandler {
	return &SSEHandler{Broker: b}
}

// StockStream handles GET /sse/stock.
// The browser connects once; the server pushes stock update events.
// HTMX SSE extension auto-processes the HTML payload and does OOB swaps.
func (h *SSEHandler) StockStream(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = "00000000-0000-0000-0000-000000000001"
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	ch := h.Broker.Subscribe(tenantID)
	defer h.Broker.Unsubscribe(tenantID, ch)

	clientGone := c.Request.Context().Done()

	// Send initial keep-alive comment
	fmt.Fprintf(c.Writer, ": connected\n\n")
	c.Writer.Flush()

	for {
		select {
		case <-clientGone:
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			// Render stock badge HTML with OOB swap attribute
			fragment := renderStockBadgeHTML(event.ProductID, event.Quantity)

			// SSE data event — HTMX SSE extension processes the HTML payload
			fmt.Fprintf(c.Writer, "data: %s\n\n", fragment)
			c.Writer.Flush()
		}
	}
}

// renderStockBadgeHTML produces the OOB-swap HTML fragment for a stock badge.
func renderStockBadgeHTML(productID string, qty int) string {
	badgeClass := "bg-green-100 text-green-700"
	if qty <= 0 {
		badgeClass = "bg-red-100 text-red-700"
	} else if qty <= 5 {
		badgeClass = "bg-yellow-100 text-yellow-700"
	}

	var buf bytes.Buffer
	fmt.Fprintf(&buf,
		`<span id="stock-%s" hx-swap-oob="true" class="text-xs px-2 py-0.5 rounded-full %s">Stok: %s</span>`,
		productID, badgeClass, strconv.Itoa(qty),
	)
	return buf.String()
}
