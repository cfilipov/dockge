package docker

import (
	"fmt"
	"sync"
)

// MockState holds in-memory container state for mock Docker/Compose.
// It is shared between MockClient and MockCompose so both see the same
// running/exited/inactive status for each stack.
type MockState struct {
	mu     sync.RWMutex
	stacks map[string]string // stackName → status ("running", "exited", "inactive")
}

// NewMockState returns an empty MockState (useful for tests).
func NewMockState() *MockState {
	return &MockState{stacks: make(map[string]string)}
}

// NewMockStateFrom returns a MockState initialized from the given map.
func NewMockStateFrom(defaults map[string]string) *MockState {
	m := make(map[string]string, len(defaults))
	for k, v := range defaults {
		m[k] = v
	}
	return &MockState{stacks: m}
}

// DefaultDevState returns state for all 200+ test stacks.
// Featured stacks (00–09) get explicit statuses; filler stacks (010–199)
// are assigned ~60% running, ~20% exited, ~20% inactive based on index.
func DefaultDevState() *MockState {
	m := make(map[string]string, 210)

	// Featured stacks
	m["00-single-service"] = "running"
	m["01-web-app"] = "running"
	m["02-blog"] = "running"
	m["03-monitoring"] = "exited"
	m["04-database"] = "running"
	m["05-multi-service"] = "running"
	m["06-mixed-state"] = "running"
	m["07-full-features"] = "running"
	m["08-env-config"] = "inactive"
	m["09-mega-stack"] = "running"
	m["test-stack"] = "running"

	// Filler stacks: 60% running, 20% exited, 20% inactive
	for i := 10; i < 200; i++ {
		name := fmt.Sprintf("stack-%03d", i)
		switch i % 5 {
		case 0, 1, 2:
			m[name] = "running"
		case 3:
			m[name] = "exited"
		case 4:
			m[name] = "inactive"
		}
	}

	return NewMockStateFrom(m)
}

// Get returns the status for a stack, or "inactive" if not present.
func (s *MockState) Get(stack string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if v, ok := s.stacks[stack]; ok {
		return v
	}
	return "inactive"
}

// Set upserts the status for a stack.
func (s *MockState) Set(stack, status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stacks[stack] = status
}

// Remove deletes a stack's state entry.
func (s *MockState) Remove(stack string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.stacks, stack)
}

// All returns a snapshot copy of all stack states.
func (s *MockState) All() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m := make(map[string]string, len(s.stacks))
	for k, v := range s.stacks {
		m[k] = v
	}
	return m
}
