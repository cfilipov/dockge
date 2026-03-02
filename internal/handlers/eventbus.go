package handlers

import (
	"sync"

	"github.com/cfilipov/dockge/internal/docker"
)

// EventBus fans out Docker events from the single broadcast watcher connection
// to multiple subscribers (individual log terminals, combined logs, etc.).
// This replaces per-terminal Docker.Events() calls that each opened a separate
// HTTP streaming connection to the Docker daemon.
type EventBus struct {
	mu     sync.RWMutex
	subs   map[uint64]chan docker.DockerEvent
	nextID uint64
}

// NewEventBus creates an EventBus ready for use.
func NewEventBus() *EventBus {
	return &EventBus{
		subs: make(map[uint64]chan docker.DockerEvent),
	}
}

// Subscribe returns a buffered channel that receives Docker events and an
// unsubscribe function. The caller must call unsub when done to avoid leaks.
func (eb *EventBus) Subscribe(bufSize int) (<-chan docker.DockerEvent, func()) {
	ch := make(chan docker.DockerEvent, bufSize)

	eb.mu.Lock()
	id := eb.nextID
	eb.nextID++
	eb.subs[id] = ch
	eb.mu.Unlock()

	unsub := func() {
		eb.mu.Lock()
		delete(eb.subs, id)
		eb.mu.Unlock()
	}

	return ch, unsub
}

// Publish sends an event to all subscribers using non-blocking sends.
// Slow consumers that can't keep up will have events dropped (their buffer
// is full). This ensures the broadcast watcher is never blocked.
func (eb *EventBus) Publish(evt docker.DockerEvent) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	for _, ch := range eb.subs {
		select {
		case ch <- evt:
		default:
			// Subscriber buffer full — drop event to avoid blocking
		}
	}
}
