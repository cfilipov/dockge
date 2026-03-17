package stack

import "sync"

// NamedMutex provides per-key mutual exclusion. Each unique key (stack name)
// gets its own mutex so operations on different stacks can proceed in parallel
// while operations on the same stack are serialized.
type NamedMutex struct {
	mu    sync.Mutex
	locks map[string]*lockEntry
}

type lockEntry struct {
	mu      sync.Mutex
	waiters int // reference count of goroutines waiting or holding the lock
}

// NewNamedMutex creates a new NamedMutex.
func NewNamedMutex() *NamedMutex {
	return &NamedMutex{locks: make(map[string]*lockEntry)}
}

// Lock acquires the mutex for the given name. It blocks until the lock
// is available.
func (nm *NamedMutex) Lock(name string) {
	nm.mu.Lock()
	e, ok := nm.locks[name]
	if !ok {
		e = &lockEntry{}
		nm.locks[name] = e
	}
	e.waiters++
	nm.mu.Unlock()

	e.mu.Lock()
}

// Unlock releases the mutex for the given name. If no other goroutines
// are waiting, the entry is cleaned up to avoid unbounded map growth.
func (nm *NamedMutex) Unlock(name string) {
	nm.mu.Lock()
	e := nm.locks[name]
	e.waiters--
	if e.waiters == 0 {
		delete(nm.locks, name)
	}
	nm.mu.Unlock()

	e.mu.Unlock()
}
