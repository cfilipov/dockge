package docker

import (
	"os"
	"path/filepath"
	"sync"
)

// MockState holds in-memory container state for the mock Docker system.
// It is shared between the fake Docker daemon and the mock docker binary
// so both see the same running/exited/inactive status for each stack.
//
// Valid statuses: "running", "exited", "inactive", "paused".
type MockState struct {
	mu       sync.RWMutex
	stacks   map[string]string // stackName → status ("running", "exited", "inactive", "paused")
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

// DefaultDevState returns state for all 200+ test stacks by auto-discovering
// the test-data/stacks directory and reading mock.yaml statuses.
// Panics if test-data cannot be found (only used from tests/benchmarks).
func DefaultDevState() *MockState {
	stacksDir := findStacksDir()
	data := BuildMockData(stacksDir)
	return DefaultDevStateFromData(data)
}

// DefaultDevStateFromData builds the initial state map from MockData.
// Every stack's status comes from its mock.yaml file (parsed into data.stackStatuses).
func DefaultDevStateFromData(data *MockData) *MockState {
	return NewMockStateFrom(data.stackStatuses)
}

// findStacksDir walks up from cwd to find test-data/stacks.
func findStacksDir() string {
	dir, err := os.Getwd()
	if err != nil {
		panic("mockstate: cannot get cwd: " + err.Error())
	}
	for {
		candidate := filepath.Join(dir, "test-data", "stacks")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			panic("mockstate: test-data/stacks directory not found")
		}
		dir = parent
	}
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
		s.stacks = make(map[string]string)
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
