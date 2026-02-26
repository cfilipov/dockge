package docker

import (
	"fmt"
	"sync"
)

// MockState holds in-memory container state for the mock Docker system.
// It is shared between the fake Docker daemon and the mock docker binary
// so both see the same running/exited/inactive status for each stack.
type MockState struct {
	mu       sync.RWMutex
	stacks   map[string]string // stackName → status ("running", "exited", "inactive")
	services map[string]string // "stackName/serviceName" → status override
	defaults map[string]string // initial state to restore on Reset()
}

// NewMockState returns an empty MockState (useful for tests).
func NewMockState() *MockState {
	return &MockState{
		stacks:   make(map[string]string),
		services: make(map[string]string),
	}
}

// NewMockStateFrom returns a MockState initialized from the given map.
func NewMockStateFrom(defaults map[string]string) *MockState {
	m := make(map[string]string, len(defaults))
	for k, v := range defaults {
		m[k] = v
	}
	d := make(map[string]string, len(defaults))
	for k, v := range defaults {
		d[k] = v
	}
	return &MockState{stacks: m, services: make(map[string]string), defaults: d}
}

// defaultDevStateMap returns the default state map for dev/test use.
// This is the hardcoded fallback; prefer DefaultDevStateFromData when MockData is available.
func defaultDevStateMap() map[string]string {
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

	return m
}

// DefaultDevState returns state for all 200+ test stacks using the hardcoded map.
// Featured stacks (00–09) get explicit statuses; filler stacks (010–199)
// are assigned ~60% running, ~20% exited, ~20% inactive based on index.
func DefaultDevState() *MockState {
	return NewMockStateFrom(defaultDevStateMap())
}

// DefaultDevStateFromData builds the initial state map from MockData.
// For stacks with mock.yaml status, uses that. For filler stacks without
// mock.yaml, uses the 60/20/20 distribution based on index.
func DefaultDevStateFromData(data *MockData) *MockState {
	base := defaultDevStateMap()

	// Override with mock.yaml statuses
	for name, status := range data.stackStatuses {
		base[name] = status
	}

	return NewMockStateFrom(base)
}

// Reset restores the mock state to its initial defaults, discarding any
// mutations made by tests (start/stop/down operations).
func (s *MockState) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.defaults != nil {
		s.stacks = make(map[string]string, len(s.defaults))
		for k, v := range s.defaults {
			s.stacks[k] = v
		}
	} else {
		s.stacks = defaultDevStateMap()
	}
	s.services = make(map[string]string)
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

// Set upserts the status for a stack and clears any per-service overrides
// (a stack-level change like "stop all" resets individual service states).
func (s *MockState) Set(stack, status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stacks[stack] = status
	// Clear per-service overrides — a stack-level action overrides them all
	for k := range s.services {
		if len(k) > len(stack) && k[:len(stack)+1] == stack+"/" {
			delete(s.services, k)
		}
	}
}

// SetService sets a per-service state override within a stack.
func (s *MockState) SetService(stack, service, status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.services[stack+"/"+service] = status
}

// GetService returns the per-service state override, or "" if none is set.
func (s *MockState) GetService(stack, service string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.services[stack+"/"+service]
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
