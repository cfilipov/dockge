use std::sync::{Arc, Mutex};

use serde::Serialize;

/// Memory snapshot from /proc/self/statm and /proc/self/status.
/// Field names match the Go memtracker JSON shape expected by E2E tests.
#[derive(Serialize, Clone, Default)]
#[serde(rename_all = "camelCase")]
struct MemSnapshot {
    heap_alloc: u64,
    heap_inuse: u64,
    heap_sys: u64,
    stack_inuse: u64,
    sys: u64,
    num_gc: u32,
    total_alloc: u64,
    goroutines: i32,
}

#[derive(Serialize, Clone, Default)]
#[serde(rename_all = "camelCase")]
struct PeakSnapshot {
    heap_alloc: u64,
    heap_inuse: u64,
    goroutines: i32,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
pub struct StatsResponse {
    current: MemSnapshot,
    peak: PeakSnapshot,
    baseline: MemSnapshot,
    samples: u64,
    avg_heap_alloc: u64,
    total_alloc_delta: u64,
    gc_cycles_delta: u32,
}

struct Inner {
    baseline: MemSnapshot,
    peak: PeakSnapshot,
    sum_heap: u64,
    samples: u64,
}

pub struct MemTracker {
    inner: Mutex<Inner>,
}

/// Read RSS and VmSize from /proc/self/statm (Linux only).
/// Returns (rss_bytes, vm_size_bytes).
fn read_proc_mem() -> (u64, u64) {
    let page_size = 4096u64; // standard Linux page size
    let Ok(content) = std::fs::read_to_string("/proc/self/statm") else {
        return (0, 0);
    };
    let mut parts = content.split_whitespace();
    let vm_pages: u64 = parts.next().and_then(|s| s.parse().ok()).unwrap_or(0);
    let rss_pages: u64 = parts.next().and_then(|s| s.parse().ok()).unwrap_or(0);
    (rss_pages * page_size, vm_pages * page_size)
}

/// Read VmRSS and VmData from /proc/self/status for more detailed stats.
fn read_proc_status() -> (u64, u64) {
    let Ok(content) = std::fs::read_to_string("/proc/self/status") else {
        return (0, 0);
    };
    let mut vm_rss = 0u64;
    let mut vm_data = 0u64;
    for line in content.lines() {
        if let Some(val) = line.strip_prefix("VmRSS:") {
            vm_rss = parse_kb_value(val);
        } else if let Some(val) = line.strip_prefix("VmData:") {
            vm_data = parse_kb_value(val);
        }
    }
    (vm_rss, vm_data)
}

fn parse_kb_value(s: &str) -> u64 {
    s.split_whitespace().next()
        .and_then(|v| v.parse::<u64>().ok())
        .unwrap_or(0)
        * 1024
}

fn read_snapshot() -> MemSnapshot {
    let (rss, vm_size) = read_proc_mem();
    let (vm_rss, vm_data) = read_proc_status();
    // Map Linux memory concepts to Go-like fields:
    // heapAlloc ≈ VmRSS (resident), heapInuse ≈ VmData (heap+data),
    // sys ≈ VmSize (total virtual)
    let heap = if vm_rss > 0 { vm_rss } else { rss };
    MemSnapshot {
        heap_alloc: heap,
        heap_inuse: vm_data,
        heap_sys: vm_data,
        stack_inuse: 0,
        sys: vm_size,
        num_gc: 0,
        total_alloc: heap, // cumulative approximation — grows with each sample
        goroutines: tokio::runtime::Handle::current().metrics().num_alive_tasks() as i32,
    }
}

impl MemTracker {
    pub fn new() -> Self {
        let snap = read_snapshot();
        let peak = PeakSnapshot {
            heap_alloc: snap.heap_alloc,
            heap_inuse: snap.heap_inuse,
            goroutines: snap.goroutines,
        };
        Self {
            inner: Mutex::new(Inner {
                baseline: snap.clone(),
                peak,
                sum_heap: snap.heap_alloc,
                samples: 1,
            }),
        }
    }

    pub fn reset(&self) {
        let snap = read_snapshot();
        let mut inner = self.inner.lock().unwrap();
        inner.baseline = snap.clone();
        inner.peak = PeakSnapshot {
            heap_alloc: snap.heap_alloc,
            heap_inuse: snap.heap_inuse,
            goroutines: snap.goroutines,
        };
        inner.sum_heap = snap.heap_alloc;
        inner.samples = 1;
    }

    fn sample(&self) {
        let snap = read_snapshot();
        let mut inner = self.inner.lock().unwrap();
        if snap.heap_alloc > inner.peak.heap_alloc {
            inner.peak.heap_alloc = snap.heap_alloc;
        }
        if snap.heap_inuse > inner.peak.heap_inuse {
            inner.peak.heap_inuse = snap.heap_inuse;
        }
        if snap.goroutines > inner.peak.goroutines {
            inner.peak.goroutines = snap.goroutines;
        }
        inner.sum_heap += snap.heap_alloc;
        inner.samples += 1;
    }

    pub fn stats(&self) -> StatsResponse {
        let current = read_snapshot();
        let inner = self.inner.lock().unwrap();
        let avg = if inner.samples > 0 {
            (inner.sum_heap + current.heap_alloc) / (inner.samples + 1)
        } else {
            0
        };
        StatsResponse {
            current: current.clone(),
            peak: inner.peak.clone(),
            baseline: inner.baseline.clone(),
            samples: inner.samples + 1,
            avg_heap_alloc: avg,
            total_alloc_delta: current.heap_alloc.saturating_sub(inner.baseline.heap_alloc),
            gc_cycles_delta: 0,
        }
    }

    /// Spawn a background task that samples every 50ms.
    pub fn spawn_sampler(tracker: Arc<Self>) {
        tokio::spawn(async move {
            let mut interval = tokio::time::interval(std::time::Duration::from_millis(50));
            loop {
                interval.tick().await;
                tracker.sample();
            }
        });
    }
}
