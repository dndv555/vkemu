package longpoll

import (
	"sync"
	"time"
)

type event struct {
	ts   int
	data []any
}

type Hub struct {
	mu      sync.Mutex
	counter int
	events  map[int][]event
	waiters map[int]chan struct{}
	keys    map[string]int
}

func New() *Hub {
	return &Hub{
		events:  map[int][]event{},
		waiters: map[int]chan struct{}{},
		keys:    map[string]int{},
	}
}

func (h *Hub) BindKey(key string, uid int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.keys[key] = uid
}

func (h *Hub) UIDByKey(key string) (int, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	uid, ok := h.keys[key]
	return uid, ok
}

func (h *Hub) TS(uid int) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.counter
}

func (h *Hub) Push(uid int, update ...any) {
	h.mu.Lock()
	h.counter++
	h.events[uid] = append(h.events[uid], event{ts: h.counter, data: update})
	ch := h.waiterLocked(uid)
	h.mu.Unlock()
	select {
	case ch <- struct{}{}:
	default:
	}
}

func (h *Hub) Wait(uid, since int, timeout time.Duration) (int, []any) {
	deadline := time.Now().Add(timeout)
	for {
		h.mu.Lock()
		var updates []any
		remaining := make([]event, 0, len(h.events[uid]))
		last := since
		for _, e := range h.events[uid] {
			if e.ts > since {
				updates = append(updates, e.data)
				last = e.ts
				continue
			}
			if e.ts == since {
				remaining = append(remaining, e)
			}
		}
		h.events[uid] = remaining
		ch := h.waiterLocked(uid)
		h.mu.Unlock()

		if len(updates) > 0 {
			return last, updates
		}
		wait := time.Until(deadline)
		if wait <= 0 {
			return since, nil
		}
		select {
		case <-ch:
		case <-time.After(wait):
			return since, nil
		}
	}
}

func (h *Hub) waiterLocked(uid int) chan struct{} {
	ch, ok := h.waiters[uid]
	if !ok {
		ch = make(chan struct{}, 1)
		h.waiters[uid] = ch
	}
	return ch
}
