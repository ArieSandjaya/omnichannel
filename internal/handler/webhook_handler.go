package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"

	"github.com/ariesandjaya/omnichannel/internal/worker"
)

// WebhookHandler handles inbound webhooks from Xendit and Biteship.
// It validates each request and enqueues an Asynq task for durable, at-least-once
// processing — never processing inline so the HTTP response is always fast.
type WebhookHandler struct {
	asynqClient    *asynq.Client
	xenditToken    string // XENDIT_WEBHOOK_TOKEN env var
	biteshipSecret string // BITESHIP_WEBHOOK_SECRET env var (HMAC key)
}

func NewWebhookHandler(
	a *asynq.Client,
	xenditToken string,
	biteshipSecret string,
) *WebhookHandler {
	return &WebhookHandler{
		asynqClient:    a,
		xenditToken:    xenditToken,
		biteshipSecret: biteshipSecret,
	}
}

// HandleXenditQRIS handles POST /webhooks/xendit/qris
//
// Xendit sends this when a QRIS payment is completed (status == "ACTIVE").
// We verify the callback token, acknowledge with 200 immediately, then enqueue
// the raw payload for asynchronous processing by the webhook worker.
func (h *WebhookHandler) HandleXenditQRIS(c *gin.Context) {
	if !h.verifyXenditToken(c) {
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		slog.Error("xendit QRIS webhook: failed to read body", "err", err)
		c.JSON(http.StatusOK, gin.H{"message": "ok"}) // still 200 — Xendit would retry otherwise
		return
	}

	// Respond 200 immediately — Xendit requires a response within 30 s;
	// actual processing happens in the Asynq worker.
	c.JSON(http.StatusOK, gin.H{"message": "received"})

	task := asynq.NewTask(worker.TypeWebhookXenditQRIS, body)
	if _, err := h.asynqClient.Enqueue(task,
		asynq.MaxRetry(5),
		asynq.Queue("webhooks"),
	); err != nil {
		slog.Error("xendit QRIS webhook: enqueue failed", "err", err)
	}
}

// HandleXenditVA handles POST /webhooks/xendit/virtual-account
//
// Xendit sends this when a Fixed VA payment is received (status == "PAID").
func (h *WebhookHandler) HandleXenditVA(c *gin.Context) {
	if !h.verifyXenditToken(c) {
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		slog.Error("xendit VA webhook: failed to read body", "err", err)
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "received"})

	task := asynq.NewTask(worker.TypeWebhookXenditVA, body)
	if _, err := h.asynqClient.Enqueue(task,
		asynq.MaxRetry(5),
		asynq.Queue("webhooks"),
	); err != nil {
		slog.Error("xendit VA webhook: enqueue failed", "err", err)
	}
}

// HandleBiteship handles POST /webhooks/biteship
//
// Biteship sends this on shipment status changes (picked_up, in_transit, delivered, etc.).
// The request is authenticated via HMAC-SHA256 over the raw body using the shared secret.
func (h *WebhookHandler) HandleBiteship(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		slog.Error("biteship webhook: failed to read body", "err", err)
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
		return
	}

	sig := c.GetHeader("x-biteship-signature")
	if !h.verifyBiteshipHMAC(body, sig) {
		slog.Warn("biteship webhook: invalid HMAC signature")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "received"})

	task := asynq.NewTask(worker.TypeWebhookBiteship, body)
	if _, err := h.asynqClient.Enqueue(task,
		asynq.MaxRetry(3),
		asynq.Queue("webhooks"),
	); err != nil {
		slog.Error("biteship webhook: enqueue failed", "err", err)
	}
}

// --- helpers ---

func (h *WebhookHandler) verifyXenditToken(c *gin.Context) bool {
	token := c.GetHeader("x-callback-token")
	if token != h.xenditToken {
		slog.Warn("xendit webhook: invalid callback token",
			"remote", c.ClientIP())
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid callback token"})
		return false
	}
	return true
}

func (h *WebhookHandler) verifyBiteshipHMAC(body []byte, signature string) bool {
	if h.biteshipSecret == "" || signature == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(h.biteshipSecret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}
