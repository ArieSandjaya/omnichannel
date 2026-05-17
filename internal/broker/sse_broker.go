package broker

import (
	"encoding/json"
	"sync"
)

// Event is the payload broadcast to SSE subscribers.
type Event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

type tenantEvent struct {
	tenantID string
	event    Event
}

// SSEBroker manages per-tenant SSE subscribers using an in-memory pub/sub model.
// For multi-instance deployments, replace the in-memory channel with Redis PubSub.
type SSEBroker struct {
	mu          sync.RWMutex
	subscribers map[string][]chan Event // key: tenantID
	publish     chan tenantEvent
	quit        chan struct{}
}

func NewSSEBroker() *SSEBroker {
	return &SSEBroker{
		subscribers: make(map[string][]chan Event),
		publish:     make(chan tenantEvent, 256),
		quit:        make(chan struct{}),
	}
}

// Run dispatches published events to subscribers. Call via go broker.Run().
func (b *SSEBroker) Run() {
	for {
		select {
		case te := <-b.publish:
			b.mu.RLock()
			subs := make([]chan Event, len(b.subscribers[te.tenantID]))
			copy(subs, b.subscribers[te.tenantID])
			b.mu.RUnlock()

			for _, ch := range subs {
				select {
				case ch <- te.event:
				default: // skip slow subscriber rather than blocking
				}
			}
		case <-b.quit:
			return
		}
	}
}

func (b *SSEBroker) Stop() {
	close(b.quit)
}

// Subscribe registers a new SSE channel for the given tenant.
func (b *SSEBroker) Subscribe(tenantID string) chan Event {
	ch := make(chan Event, 32)
	b.mu.Lock()
	b.subscribers[tenantID] = append(b.subscribers[tenantID], ch)
	b.mu.Unlock()
	return ch
}

// Unsubscribe removes the channel and closes it.
func (b *SSEBroker) Unsubscribe(tenantID string, ch chan Event) {
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

// Publish sends an event to all subscribers of a tenant (non-blocking).
func (b *SSEBroker) Publish(tenantID string, event Event) {
	select {
	case b.publish <- tenantEvent{tenantID: tenantID, event: event}:
	default: // drop if publish channel is full
	}
}

// MarshalEvent serializes an Event to the SSE wire format.
func MarshalEvent(e Event) []byte {
	data, _ := json.Marshal(e.Data)
	return []byte("event: " + e.Type + "\ndata: " + string(data) + "\n\n")
}
