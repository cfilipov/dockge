package docker

import "sync"

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

// DefaultDevState returns hardcoded state matching the test stacks in /opt/stacks.
func DefaultDevState() *MockState {
	return NewMockStateFrom(map[string]string{
		"web-app":     "running",
		"monitoring":  "running",
		"test-alpine": "exited",
		"blog":        "running",
	})
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
