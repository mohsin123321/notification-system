package stream

import (
	"context"
	"sync"

	"github.com/mohsin123321/notification-system/internal/domain/notification/pure"
)

// Watcher implements notification/port.StatusWatcher using in-process channels.
// It fans out status events to all active subscribers for a given notification ID.
type Watcher struct {
	mu   sync.RWMutex
	subs map[pure.NotificationID][]chan pure.StatusEvent
}

// New creates a new Watcher.
func New() *Watcher {
	return &Watcher{subs: make(map[pure.NotificationID][]chan pure.StatusEvent)}
}

// Notify delivers an event to all subscribers for the notification.
// Slow subscribers are skipped (non-blocking send).
func (w *Watcher) Notify(_ context.Context, event pure.StatusEvent) error {
	w.mu.RLock()
	defer w.mu.RUnlock()
	for _, ch := range w.subs[event.NotificationID] {
		select {
		case ch <- event:
		default: // drop if subscriber is slow
		}
	}
	return nil
}

// Subscribe returns a buffered channel that will receive status events for id.
func (w *Watcher) Subscribe(_ context.Context, id pure.NotificationID) (<-chan pure.StatusEvent, error) {
	ch := make(chan pure.StatusEvent, 10)
	w.mu.Lock()
	w.subs[id] = append(w.subs[id], ch)
	w.mu.Unlock()
	return ch, nil
}

// Unsubscribe removes the subscription and closes the channel.
func (w *Watcher) Unsubscribe(_ context.Context, id pure.NotificationID, ch <-chan pure.StatusEvent) {
	w.mu.Lock()
	defer w.mu.Unlock()
	subs := w.subs[id]
	for i, sub := range subs {
		if sub == ch {
			w.subs[id] = append(subs[:i], subs[i+1:]...)
			close(sub)
			break
		}
	}
}
