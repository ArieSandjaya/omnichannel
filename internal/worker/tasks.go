package worker

// Asynq task type constants.
const (
	TypeStockSync      = "stock:sync"        // sync product stock to marketplaces
	TypeWebhookXenditQRIS = "webhook:xendit:qris"  // process Xendit QRIS webhook payload
	TypeWebhookXenditVA   = "webhook:xendit:va"    // process Xendit VA webhook payload
	TypeWebhookBiteship   = "webhook:biteship"     // process Biteship webhook payload
)
