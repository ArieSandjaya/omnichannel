package broker

import "sync"

type StockEvent struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
	TenantID  string `json:"tenant_id"`
}

// SSEBroker manages per-tenant SSE client subscriptions.
// At scale (multiple instances), replace with Redis PubSub.
type SSEBroker struct {
	mu          sync.RWMutex
	subscribers map[string][]chan StockEvent // keyed by tenant_id
}

func NewSSEBroker() *SSEBroker {
	return &SSEBroker{
		subscribers: make(map[string][]chan StockEvent),
	}
}

func (b *SSEBroker) Subscribe(tenantID string) chan StockEvent {
	ch := make(chan StockEvent, 16)
	b.mu.Lock()
	b.subscribers[tenantID] = append(b.subscribers[tenantID], ch)
	b.mu.Unlock()
	return ch
}

func (b *SSEBroker) Unsubscribe(tenantID string, ch chan StockEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	subs := b.subscribers[tenantID]
	for i, s := range subs {
		if s == ch {
			b.subscribers[tenantID] = append(subs[:i], subs[i+1:]...)
			close(ch)
			return
		}
	}
}

func (b *SSEBroker) Publish(event StockEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.subscribers[event.TenantID] {
		select {
		case ch <- event:
		default:
			// drop event if subscriber channel is full (slow client)
		}
	}
}

func (b *SSEBroker) TenantClientCount(tenantID string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers[tenantID])
}
