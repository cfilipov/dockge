import { readFileSync, writeFileSync, mkdirSync } from "fs";
import { join, dirname } from "path";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

// ---------- Data types ----------

export interface MemSample {
    heapAlloc: number;
    heapInuse: number;
    sys: number;
    goroutines: number;
    totalAlloc: number;
    numGC: number;
    stackInuse: number;
}

export interface TestSocketData {
    serverFrameCount: number;
    serverTotalBytes: number;
    clientFrameCount: number;
    clientTotalBytes: number;
    serverEventCounts: Record<string, number>;
    serverEventBytes: Record<string, number>;
}

export interface PerfResults {
    memory: {
        baseline: MemSample;
        peak: { heapAlloc: number; heapInuse: number; goroutines: number };
        final: MemSample;
        sampleCount: number;
        avgHeapAlloc: number;
        totalAllocDelta: number;
        gcCyclesDelta: number;
    };
    socket: {
        initialLoad: {
            [channel: string]: { count: number; bytes: number };
        };
        perTest: Record<string, TestSocketData>;
        totalServerFrames: number;
        totalServerBytes: number;
    };
}

// Tolerance definitions for comparison
export interface ToleranceSpec {
    type: "percent" | "exact";
    value: number; // percentage (e.g. 10 for ±10%) or 0 for exact
}

export const TOLERANCES: Record<string, ToleranceSpec> = {
    // Memory (percent-based). Peak goroutines omitted — 500ms polling
    // makes instantaneous goroutine peaks too noisy for comparison;
    // the value is still recorded in the baseline for human review.
    // Peak heap omitted — it's a single instantaneous maximum that swings
    // 50%+ depending on GC timing. Avg and final heap reliably catch leaks.
    "memory.final.heapAlloc": { type: "percent", value: 10 },
    "memory.avgHeapAlloc": { type: "percent", value: 10 },
    "memory.cumulativeAlloc": { type: "percent", value: 15 },
    "memory.gcCyclesDelta": { type: "percent", value: 20 },
    // Socket totals — ±2% because debounced broadcasts can shift frame
    // counts between adjacent tests, but the total is stable.
    "socket.totalServerFrames": { type: "percent", value: 2 },
    "socket.totalServerBytes": { type: "percent", value: 2 },
};

// ---------- Initial load channels ----------

const INITIAL_BROADCAST_CHANNELS = new Set([
    "info",
    "stacks",
    "containers",
    "networks",
    "images",
    "volumes",
    "updates",
]);

// ---------- PerfCollector ----------

export class PerfCollector {
    private baseURL: string;
    private pollInterval: ReturnType<typeof setInterval> | null = null;
    private samples: MemSample[] = [];
    private perTest: Record<string, TestSocketData> = {};
    private initialLoad: Record<string, { count: number; bytes: number }> = {};
    private totalServerFrames = 0;
    private totalServerBytes = 0;
    private firstTestName: string | null = null;
    private initialLoadComplete = false;
    private currentTest: string | null = null;
    private wsConnectionCounts: Record<string, number> = {};

    constructor(baseURL?: string) {
        const port = parseInt(process.env.E2E_PORT || "5051", 10);
        this.baseURL = baseURL || `http://localhost:${port}`;
    }

    // ---------- Memory polling ----------

    async startMemoryPolling(): Promise<void> {
        // Take initial sample
        await this.pollOnce();

        this.pollInterval = setInterval(() => {
            this.pollOnce().catch(() => {
                // Server unavailable during polling — skip sample
            });
        }, 500);
    }

    private async pollOnce(): Promise<void> {
        const resp = await fetch(`${this.baseURL}/api/debug/memstats`);
        if (!resp.ok) return;
        const data = await resp.json();
        this.samples.push({
            heapAlloc: data.heapAlloc,
            heapInuse: data.heapInuse,
            sys: data.sys,
            goroutines: data.goroutines,
            totalAlloc: data.totalAlloc,
            numGC: data.numGC,
            stackInuse: data.stackInuse,
        });
    }

    async stop(): Promise<void> {
        if (this.pollInterval) {
            clearInterval(this.pollInterval);
            this.pollInterval = null;
        }
        // Final sample
        await this.pollOnce().catch(() => {});
    }

    // ---------- Test lifecycle ----------

    beginTest(testName: string): void {
        this.currentTest = testName;
        if (this.firstTestName === null) {
            this.firstTestName = testName;
        }
        if (!this.perTest[testName]) {
            this.perTest[testName] = {
                serverFrameCount: 0,
                serverTotalBytes: 0,
                clientFrameCount: 0,
                clientTotalBytes: 0,
                serverEventCounts: {},
                serverEventBytes: {},
            };
        }
    }

    endTest(testName: string): void {
        // Mark initial load as complete after the first test
        if (testName === this.firstTestName) {
            this.initialLoadComplete = true;
        }
        this.currentTest = null;
    }

    // ---------- WebSocket tracking ----------

    recordNewConnection(testName: string): void {
        this.wsConnectionCounts[testName] = (this.wsConnectionCounts[testName] || 0) + 1;
    }

    recordServerFrame(testName: string, payload: string): void {
        const bytes = Buffer.byteLength(payload, "utf-8");
        this.totalServerFrames++;
        this.totalServerBytes += bytes;

        const test = this.perTest[testName];
        if (!test) return;

        test.serverFrameCount++;
        test.serverTotalBytes += bytes;

        // Parse to extract event name for channel-level tracking
        const eventName = this.extractEventName(payload);
        if (eventName) {
            test.serverEventCounts[eventName] = (test.serverEventCounts[eventName] || 0) + 1;
            test.serverEventBytes[eventName] = (test.serverEventBytes[eventName] || 0) + bytes;

            // Track initial load broadcasts (only during first test, before initial load is marked complete)
            if (!this.initialLoadComplete && testName === this.firstTestName && INITIAL_BROADCAST_CHANNELS.has(eventName)) {
                if (!this.initialLoad[eventName]) {
                    this.initialLoad[eventName] = { count: 0, bytes: 0 };
                }
                this.initialLoad[eventName].count++;
                this.initialLoad[eventName].bytes += bytes;
            }
        }
    }

    recordClientFrame(testName: string, payload: string): void {
        const bytes = Buffer.byteLength(payload, "utf-8");
        const test = this.perTest[testName];
        if (!test) return;

        test.clientFrameCount++;
        test.clientTotalBytes += bytes;
    }

    private extractEventName(payload: string): string | null {
        try {
            const msg = JSON.parse(payload);
            // Server push event: { event: "stacks", data: ... }
            if (typeof msg.event === "string") {
                return msg.event;
            }
            // ACK message: { id: 1, data: ... } — categorize as "ack"
            if (typeof msg.id === "number" && msg.data !== undefined) {
                return "ack";
            }
        } catch {
            // Not JSON — ignore
        }
        return null;
    }

    // ---------- Results ----------

    getResults(): PerfResults {
        const baseline = this.samples[0] || emptyMemSample();
        const final = this.samples[this.samples.length - 1] || emptyMemSample();

        let peakHeapAlloc = 0;
        let peakHeapInuse = 0;
        let peakGoroutines = 0;
        let sumHeapAlloc = 0;

        for (const s of this.samples) {
            if (s.heapAlloc > peakHeapAlloc) peakHeapAlloc = s.heapAlloc;
            if (s.heapInuse > peakHeapInuse) peakHeapInuse = s.heapInuse;
            if (s.goroutines > peakGoroutines) peakGoroutines = s.goroutines;
            sumHeapAlloc += s.heapAlloc;
        }

        const sampleCount = this.samples.length;
        const avgHeapAlloc = sampleCount > 0 ? Math.round(sumHeapAlloc / sampleCount) : 0;

        // Compute initial load total
        const initialLoadWithTotal: Record<string, { count: number; bytes: number }> = { ...this.initialLoad };
        let totalInitCount = 0;
        let totalInitBytes = 0;
        for (const ch of Object.values(this.initialLoad)) {
            totalInitCount += ch.count;
            totalInitBytes += ch.bytes;
        }
        initialLoadWithTotal["total"] = { count: totalInitCount, bytes: totalInitBytes };

        return {
            memory: {
                baseline,
                peak: { heapAlloc: peakHeapAlloc, heapInuse: peakHeapInuse, goroutines: peakGoroutines },
                final,
                sampleCount,
                avgHeapAlloc,
                totalAllocDelta: final.totalAlloc - baseline.totalAlloc,
                gcCyclesDelta: final.numGC - baseline.numGC,
            },
            socket: {
                initialLoad: initialLoadWithTotal,
                perTest: this.perTest,
                totalServerFrames: this.totalServerFrames,
                totalServerBytes: this.totalServerBytes,
            },
        };
    }

    // ---------- Comparison ----------

    static compare(actual: PerfResults, baseline: PerfResults): ComparisonResult[] {
        const results: ComparisonResult[] = [];

        // Memory comparisons. Peak heap and goroutines omitted — single-sample
        // maximums swing 50%+ from GC timing jitter. Avg/final are stable.
        compareField(results, "memory.final.heapAlloc", actual.memory.final.heapAlloc, baseline.memory.final.heapAlloc);
        compareField(results, "memory.avgHeapAlloc", actual.memory.avgHeapAlloc, baseline.memory.avgHeapAlloc);
        compareField(results, "memory.cumulativeAlloc", actual.memory.totalAllocDelta, baseline.memory.totalAllocDelta);
        compareField(results, "memory.gcCyclesDelta", actual.memory.gcCyclesDelta, baseline.memory.gcCyclesDelta);

        // Socket totals
        compareField(results, "socket.totalServerFrames", actual.socket.totalServerFrames, baseline.socket.totalServerFrames);
        compareField(results, "socket.totalServerBytes", actual.socket.totalServerBytes, baseline.socket.totalServerBytes);

        // Initial load — exact frame count per channel
        for (const channel of Object.keys(baseline.socket.initialLoad)) {
            const bCh = baseline.socket.initialLoad[channel];
            const aCh = actual.socket.initialLoad[channel];
            if (aCh) {
                compareField(results, `socket.initialLoad.${channel}.count`, aCh.count, bCh.count,
                    { type: "exact", value: 0 });
                compareField(results, `socket.initialLoad.${channel}.bytes`, aCh.bytes, bCh.bytes,
                    { type: "percent", value: 2 });
            } else {
                results.push({
                    field: `socket.initialLoad.${channel}`,
                    actual: 0,
                    baseline: bCh.count,
                    tolerance: "exact",
                    passed: false,
                    message: `Missing initial load channel "${channel}" (expected ${bCh.count} frames)`,
                });
            }
        }

        // Per-test socket data is recorded in the baseline for human review
        // but NOT asserted — debounced broadcasts at test boundaries cause
        // frame/byte attribution to shift between adjacent tests. The totals
        // above catch actual regressions; per-test data aids manual inspection.

        return results;
    }

    // ---------- Golden file I/O ----------

    static readonly GOLDEN_PATH = join(__dirname, "..", "__benchmarks__", "perf-baseline.json");

    static loadBaseline(): PerfResults | null {
        try {
            const raw = readFileSync(PerfCollector.GOLDEN_PATH, "utf-8");
            return JSON.parse(raw) as PerfResults;
        } catch {
            return null;
        }
    }

    static saveBaseline(results: PerfResults): void {
        const dir = dirname(PerfCollector.GOLDEN_PATH);
        mkdirSync(dir, { recursive: true });
        writeFileSync(PerfCollector.GOLDEN_PATH, JSON.stringify(results, null, 4) + "\n");
    }
}

// ---------- Helpers ----------

export interface ComparisonResult {
    field: string;
    actual: number;
    baseline: number;
    tolerance: string;
    passed: boolean;
    message: string;
}

function emptyMemSample(): MemSample {
    return { heapAlloc: 0, heapInuse: 0, sys: 0, goroutines: 0, totalAlloc: 0, numGC: 0, stackInuse: 0 };
}

function compareField(
    results: ComparisonResult[],
    field: string,
    actual: number,
    baseline: number,
    toleranceOverride?: ToleranceSpec,
): void {
    const tol = toleranceOverride || TOLERANCES[field];
    if (!tol) {
        // No tolerance defined — skip (shouldn't happen for fields we care about)
        return;
    }

    let passed: boolean;
    let toleranceStr: string;

    if (tol.type === "exact") {
        passed = actual === baseline;
        toleranceStr = "exact";
    } else {
        const margin = baseline * (tol.value / 100);
        passed = actual >= baseline - margin && actual <= baseline + margin;
        toleranceStr = `\u00b1${tol.value}%`;
    }

    const pctChange = baseline !== 0 ? ((actual - baseline) / baseline * 100).toFixed(1) : "N/A";
    const message = passed
        ? `${field}: ${actual} (baseline ${baseline}, ${pctChange}%, within ${toleranceStr})`
        : `${field}: ${actual} vs baseline ${baseline} (${pctChange}% change, exceeds ${toleranceStr} tolerance)`;

    results.push({ field, actual, baseline, tolerance: toleranceStr, passed, message });
}
