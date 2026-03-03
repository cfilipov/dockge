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
    // Memory (percent-based). In-process 50ms sampling makes peaks reliable.
    "memory.peak.heapAlloc": { type: "percent", value: 10 },
    "memory.peak.heapInuse": { type: "percent", value: 10 },
    "memory.peak.goroutines": { type: "percent", value: 15 },
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

    // ---------- Memory (server-side tracking) ----------

    async resetMemoryBaseline(): Promise<void> {
        const resp = await fetch(`${this.baseURL}/api/debug/memstats/reset`, { method: "POST" });
        if (!resp.ok) throw new Error(`reset memstats: ${resp.status}`);
    }

    private async fetchMemoryResults(): Promise<PerfResults["memory"]> {
        const resp = await fetch(`${this.baseURL}/api/debug/memstats`);
        if (!resp.ok) throw new Error(`fetch memstats: ${resp.status}`);
        const data = await resp.json();
        return {
            baseline: {
                heapAlloc: data.baseline.heapAlloc,
                heapInuse: data.baseline.heapInuse,
                sys: data.baseline.sys,
                goroutines: data.baseline.goroutines,
                totalAlloc: data.baseline.totalAlloc,
                numGC: data.baseline.numGC,
                stackInuse: data.baseline.stackInuse,
            },
            peak: {
                heapAlloc: data.peak.heapAlloc,
                heapInuse: data.peak.heapInuse,
                goroutines: data.peak.goroutines,
            },
            final: {
                heapAlloc: data.current.heapAlloc,
                heapInuse: data.current.heapInuse,
                sys: data.current.sys,
                goroutines: data.current.goroutines,
                totalAlloc: data.current.totalAlloc,
                numGC: data.current.numGC,
                stackInuse: data.current.stackInuse,
            },
            sampleCount: data.samples,
            avgHeapAlloc: data.avgHeapAlloc,
            totalAllocDelta: data.totalAllocDelta,
            gcCyclesDelta: data.gcCyclesDelta,
        };
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

    async getResults(): Promise<PerfResults> {
        const memory = await this.fetchMemoryResults();

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
            memory,
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

        // Memory comparisons — in-process 50ms sampling makes peaks stable.
        compareField(results, "memory.peak.heapAlloc", actual.memory.peak.heapAlloc, baseline.memory.peak.heapAlloc);
        compareField(results, "memory.peak.heapInuse", actual.memory.peak.heapInuse, baseline.memory.peak.heapInuse);
        compareField(results, "memory.peak.goroutines", actual.memory.peak.goroutines, baseline.memory.peak.goroutines);
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
