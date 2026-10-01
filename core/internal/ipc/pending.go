package ipc

import "sync"

// PendingOpen holds "ui" actions captured before the GUI subscribed to the
// "ui" topic, so a cold start still honors every request that caused it.
type PendingOpen struct {
	mu       sync.Mutex
	payloads []map[string]any
}

func (p *PendingOpen) Deliver(bus *EventBus, payload map[string]any) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.payloads) == 0 && bus.HasSubscriber("ui") {
		bus.Publish("ui", payload)
		return
	}
	p.payloads = append(p.payloads, payload)
}

func (p *PendingOpen) Take() []map[string]any {
	p.mu.Lock()
	defer p.mu.Unlock()
	payloads := p.payloads
	p.payloads = nil
	return payloads
}

func (p *PendingOpen) Flush(bus *EventBus) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, payload := range p.payloads {
		bus.Publish("ui", payload)
	}
	p.payloads = nil
}
