package sse

import "sync"

// Hub is a tiny in-memory broadcast: every subscriber receives every published
// message. At a friend-group scale this is plenty and needs no external broker.
type Hub struct {
	mu   sync.RWMutex
	subs map[chan string]struct{}
}

func NewHub() *Hub {
	return &Hub{subs: make(map[chan string]struct{})}
}

// Subscribe registers a new buffered channel.
func (h *Hub) Subscribe() chan string {
	ch := make(chan string, 8)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

// Unsubscribe removes and closes a channel.
func (h *Hub) Unsubscribe(ch chan string) {
	h.mu.Lock()
	if _, ok := h.subs[ch]; ok {
		delete(h.subs, ch)
		close(ch)
	}
	h.mu.Unlock()
}

// Publish fans a message out to all subscribers, dropping it for any slow one.
func (h *Hub) Publish(msg string) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.subs {
		select {
		case ch <- msg:
		default:
		}
	}
}
