package debug

import (
	"context"
	"encoding/json"
	"net/http"
	"runtime"
	"sync"
	"time"
)

// memSnapshot holds a point-in-time memory reading.
type memSnapshot struct {
	HeapAlloc  uint64 `json:"heapAlloc"`
	HeapInuse  uint64 `json:"heapInuse"`
	HeapSys    uint64 `json:"heapSys"`
	StackInuse uint64 `json:"stackInuse"`
	Sys        uint64 `json:"sys"`
	NumGC      uint32 `json:"numGC"`
	TotalAlloc uint64 `json:"totalAlloc"`
	Goroutines int    `json:"goroutines"`
}

// peakSnapshot holds running maximums.
type peakSnapshot struct {
	HeapAlloc  uint64 `json:"heapAlloc"`
	HeapInuse  uint64 `json:"heapInuse"`
	Goroutines int    `json:"goroutines"`
}

// statsResponse is the JSON shape returned by HandleGet.
type statsResponse struct {
	Current        memSnapshot  `json:"current"`
	Peak           peakSnapshot `json:"peak"`
	Baseline       memSnapshot  `json:"baseline"`
	Samples        uint64       `json:"samples"`
	AvgHeapAlloc   uint64       `json:"avgHeapAlloc"`
	TotalAllocDlta uint64       `json:"totalAllocDelta"`
	GCCyclesDelta  uint32       `json:"gcCyclesDelta"`
}

// MemTracker samples runtime.ReadMemStats at a fixed interval and maintains
// baseline, peaks, and running averages. No forced GC — ReadMemStats alone
// takes ~microseconds.
type MemTracker struct {
	mu       sync.Mutex
	baseline memSnapshot
	peak     peakSnapshot
	sumHeap  uint64 // running sum of heapAlloc for average
	samples  uint64
}

func readSnapshot() memSnapshot {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return memSnapshot{
		HeapAlloc:  m.HeapAlloc,
		HeapInuse:  m.HeapInuse,
		HeapSys:    m.HeapSys,
		StackInuse: m.StackInuse,
		Sys:        m.Sys,
		NumGC:      m.NumGC,
		TotalAlloc: m.TotalAlloc,
		Goroutines: runtime.NumGoroutine(),
	}
}

// NewMemTracker starts a background goroutine that samples memory every
// interval. The goroutine stops when ctx is cancelled.
func NewMemTracker(ctx context.Context, interval time.Duration) *MemTracker {
	t := &MemTracker{}
	t.reset()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				t.sample()
			}
		}
	}()

	return t
}

func (t *MemTracker) reset() {
	snap := readSnapshot()
	t.baseline = snap
	t.peak = peakSnapshot{}
	t.sumHeap = 0
	t.samples = 0
	// Record the baseline as the first sample so peaks start from it.
	t.updatePeaks(snap)
	t.sumHeap = snap.HeapAlloc
	t.samples = 1
}

func (t *MemTracker) sample() {
	snap := readSnapshot()
	t.mu.Lock()
	defer t.mu.Unlock()
	t.updatePeaks(snap)
	t.sumHeap += snap.HeapAlloc
	t.samples++
}

// updatePeaks must be called with t.mu held (or during init before goroutine starts).
func (t *MemTracker) updatePeaks(s memSnapshot) {
	if s.HeapAlloc > t.peak.HeapAlloc {
		t.peak.HeapAlloc = s.HeapAlloc
	}
	if s.HeapInuse > t.peak.HeapInuse {
		t.peak.HeapInuse = s.HeapInuse
	}
	if s.Goroutines > t.peak.Goroutines {
		t.peak.Goroutines = s.Goroutines
	}
}

// HandleGet returns the current memory stats, peaks, baseline, and computed deltas.
// A single GC is forced here so the "current" reading reflects live heap only.
// This does NOT affect the background sampler — peaks remain undistorted.
func (t *MemTracker) HandleGet(w http.ResponseWriter, _ *http.Request) {
	runtime.GC()
	current := readSnapshot()

	t.mu.Lock()
	// Include the current reading in the running stats.
	t.updatePeaks(current)
	t.sumHeap += current.HeapAlloc
	t.samples++

	resp := statsResponse{
		Current:        current,
		Peak:           t.peak,
		Baseline:       t.baseline,
		Samples:        t.samples,
		TotalAllocDlta: current.TotalAlloc - t.baseline.TotalAlloc,
		GCCyclesDelta:  current.NumGC - t.baseline.NumGC,
	}
	if t.samples > 0 {
		resp.AvgHeapAlloc = t.sumHeap / t.samples
	}
	t.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleReset zeroes peaks and running sums, takes a fresh baseline.
func (t *MemTracker) HandleReset(w http.ResponseWriter, _ *http.Request) {
	t.mu.Lock()
	t.reset()
	t.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}
