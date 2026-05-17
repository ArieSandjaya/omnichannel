package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ariesandjaya/omnichannel/internal/broker"
	"github.com/ariesandjaya/omnichannel/internal/middleware"
)

type SSEHandler struct {
	broker *broker.SSEBroker
}

func NewSSEHandler(b *broker.SSEBroker) *SSEHandler {
	return &SSEHandler{broker: b}
}

// Stock handles GET /sse/stock
// Streams Server-Sent Events (payment.success, stock updates) to the browser.
// HTMX SSE extension subscribes to this endpoint and swaps stock badges on the page.
func (h *SSEHandler) Stock(c *gin.Context) {
	tenantID := middleware.TenantID(c).String()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // disable nginx buffering

	// Send a heartbeat comment so the browser knows the connection is alive
	c.Writer.Write([]byte(": connected\n\n"))
	c.Writer.Flush()

	ch := h.broker.Subscribe(tenantID)
	defer h.broker.Unsubscribe(tenantID, ch)

	clientGone := c.Request.Context().Done()
	for {
		select {
		case <-clientGone:
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			_, err := c.Writer.Write(broker.MarshalEvent(event))
			if err != nil {
				return
			}
			if flusher, ok := c.Writer.(http.Flusher); ok {
				flusher.Flush()
			}
		}
	}
}
