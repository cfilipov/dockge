package stack

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestNamedMutexBasic(t *testing.T) {
	nm := NewNamedMutex()
	nm.Lock("a")
	nm.Unlock("a")
	// Should not deadlock
}

func TestNamedMutexDifferentKeysParallel(t *testing.T) {
	nm := NewNamedMutex()
	var wg sync.WaitGroup

	// Two different keys should not block each other
	wg.Add(2)
	ready := make(chan struct{})

	go func() {
		defer wg.Done()
		nm.Lock("a")
		close(ready) // signal that "a" is locked
		<-ready      // unblock immediately
		nm.Unlock("a")
	}()

	go func() {
		defer wg.Done()
		<-ready // wait for "a" to be locked
		nm.Lock("b")
		nm.Unlock("b")
	}()

	wg.Wait()
}

func TestNamedMutexSameKeySerializes(t *testing.T) {
	nm := NewNamedMutex()
	var counter atomic.Int32
	var maxConcurrent atomic.Int32
	var wg sync.WaitGroup

	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			nm.Lock("stack")
			cur := counter.Add(1)
			// Track max concurrency — should always be 1
			for {
				old := maxConcurrent.Load()
				if cur <= old || maxConcurrent.CompareAndSwap(old, cur) {
					break
				}
			}
			counter.Add(-1)
			nm.Unlock("stack")
		}()
	}

	wg.Wait()

	if max := maxConcurrent.Load(); max != 1 {
		t.Errorf("expected max concurrency 1, got %d", max)
	}
}

func TestNamedMutexCleanup(t *testing.T) {
	nm := NewNamedMutex()

	nm.Lock("temp")
	nm.Unlock("temp")

	nm.mu.Lock()
	_, exists := nm.locks["temp"]
	nm.mu.Unlock()

	if exists {
		t.Error("expected lock entry to be cleaned up after last unlock")
	}
}
