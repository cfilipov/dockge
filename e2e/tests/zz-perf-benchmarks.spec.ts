import { test, expect } from "../fixtures/auth.fixture";
import { PerfCollector, ComparisonResult } from "../helpers/perf-collector";

const UPDATE_MODE = !!process.env.UPDATE_BENCHMARKS;

test.describe("Performance Benchmarks", () => {
    test("memory and socket metrics within baseline tolerances", async ({ perfCollector }) => {
        const results = perfCollector.getResults();
        const baseline = PerfCollector.loadBaseline();

        // First run or update mode: write golden file and pass
        if (!baseline || UPDATE_MODE) {
            PerfCollector.saveBaseline(results);
            const action = baseline ? "Updated" : "Created initial";
            // eslint-disable-next-line no-console
            console.log(`${action} performance baseline at ${PerfCollector.GOLDEN_PATH}`);
            logSummary(results);
            return;
        }

        // Compare against baseline
        const comparisons = PerfCollector.compare(results, baseline);
        const failures = comparisons.filter((c) => !c.passed);

        // Log all comparisons for visibility
        // eslint-disable-next-line no-console
        console.log("\n=== Performance Benchmark Results ===\n");
        for (const c of comparisons) {
            // eslint-disable-next-line no-console
            console.log(`  ${c.passed ? "PASS" : "FAIL"} ${c.message}`);
        }

        if (failures.length > 0) {
            // eslint-disable-next-line no-console
            console.log(
                `\nTo update baselines: UPDATE_BENCHMARKS=1 task test-e2e\n` +
                `Or: task update-benchmarks\n`
            );
        }

        logSummary(results);

        expect(
            failures.length,
            formatFailures(failures),
        ).toBe(0);
    });
});

function logSummary(results: ReturnType<PerfCollector["getResults"]>): void {
    const log = (msg: string) => console.log(msg); // eslint-disable-line no-console

    log("\n=== Performance Summary ===");
    log("");
    log("  Memory");
    log(`    Samples:              ${results.memory.sampleCount}`);
    log(`    Peak heap:            ${formatBytes(results.memory.peak.heapAlloc)}`);
    log(`    Avg heap:             ${formatBytes(results.memory.avgHeapAlloc)}`);
    log(`    Final heap:           ${formatBytes(results.memory.final.heapAlloc)}`);
    log(`    Peak goroutines:      ${results.memory.peak.goroutines}`);
    log(`    Cumulative alloc:     ${formatBytes(results.memory.totalAllocDelta)} (total bytes allocated during run)`);
    log(`    GC cycles:            ${results.memory.gcCyclesDelta}`);

    log("");
    log("  WebSocket");
    log(`    Total server frames:  ${results.socket.totalServerFrames}`);
    log(`    Total server bytes:   ${formatBytes(results.socket.totalServerBytes)}`);
    log(`    Tests tracked:        ${Object.keys(results.socket.perTest).length}`);

    log("");
    log("  Initial Load (first connection)");
    const channels = Object.entries(results.socket.initialLoad)
        .filter(([ch]) => ch !== "total")
        .sort(([, a], [, b]) => b.bytes - a.bytes);
    for (const [channel, data] of channels) {
        const name = (channel + ":").padEnd(16);
        log(`    ${name}${formatBytes(data.bytes).padStart(10)}  (${data.count} frame)`);
    }
    const total = results.socket.initialLoad["total"];
    if (total) {
        log(`    ${"─".repeat(28)}`);
        log(`    ${"total:".padEnd(16)}${formatBytes(total.bytes).padStart(10)}  (${total.count} frames)`);
    }
}

function formatBytes(bytes: number): string {
    if (bytes < 1024) return `${bytes}B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)}KiB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)}MiB`;
}

function formatFailures(failures: ComparisonResult[]): string {
    const lines = [`${failures.length} performance metric(s) exceeded tolerance:\n`];
    for (const f of failures) {
        lines.push(`  - ${f.message}`);
    }
    lines.push("");
    lines.push("Run `task update-benchmarks` to accept these changes as the new baseline.");
    return lines.join("\n");
}
