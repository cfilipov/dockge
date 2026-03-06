import { writeFileSync, mkdirSync } from "fs";
import { join, dirname } from "path";
import { fileURLToPath } from "url";
import { test, expect } from "../fixtures/auth.fixture";
import { PerfCollector, PerfResults, ComparisonResult, measureBuildSizes } from "../helpers/perf-collector";

const __dirname = dirname(fileURLToPath(import.meta.url));
const UPDATE_MODE = !!process.env.UPDATE_BENCHMARKS;
const REPORT_PATH = join(__dirname, "..", "..", ".e2e-output", "test-results", "benchmark-report.txt");

test.describe("Performance Benchmarks", () => {
    test("memory and socket metrics within baseline tolerances", async ({ perfCollector }) => {
        const results = await perfCollector.getResults();

        // The benchmark aggregates WebSocket and memory data collected by prior
        // tests in the same worker.  When run in isolation (e.g. a grep that
        // matches only this file), no other tests populate the collector, so
        // every metric reads zero.  Skip cleanly instead of failing.
        test.skip(
            results.socket.totalServerFrames === 0,
            "Benchmark requires the full E2E suite — run via `task test-e2e`",
        );
        const projectRoot = join(__dirname, "..", "..");
        try {
            results.build = measureBuildSizes(projectRoot);
        } catch {
            // Build artifacts missing — skip build size tracking
        }
        const baseline = PerfCollector.loadBaseline();
        const report = new ReportBuilder();

        // First run or update mode: write golden file and pass
        if (!baseline || UPDATE_MODE) {
            PerfCollector.saveBaseline(results);
            const action = baseline ? "Updated" : "Created initial";
            report.line(`${action} performance baseline at ${PerfCollector.GOLDEN_PATH}`);
            buildSummary(report, results);
            report.flush();
            return;
        }

        // Compare against baseline
        const comparisons = PerfCollector.compare(results, baseline);
        const failures = comparisons.filter((c) => !c.passed);

        report.line("\n=== Performance Benchmark Results ===\n");
        for (const c of comparisons) {
            report.line(`  ${c.passed ? "PASS" : "FAIL"} ${c.message}`);
        }

        if (failures.length > 0) {
            report.line(
                `\nTo update baselines: UPDATE_BENCHMARKS=1 task test-e2e\n` +
                `Or: task update-benchmarks\n`
            );
        }

        buildSummary(report, results);
        report.flush();

        expect(
            failures.length,
            formatFailures(failures),
        ).toBe(0);
    });
});

/** Collects report lines, logs to stdout, and writes to file. */
class ReportBuilder {
    private lines: string[] = [];

    line(msg: string): void {
        console.log(msg); // eslint-disable-line no-console
        this.lines.push(msg);
    }

    flush(): void {
        const dir = dirname(REPORT_PATH);
        mkdirSync(dir, { recursive: true });
        writeFileSync(REPORT_PATH, this.lines.join("\n") + "\n");
    }
}

function buildSummary(report: ReportBuilder, results: PerfResults): void {
    report.line("\n=== Performance Summary ===");
    report.line("");
    report.line("  Memory");
    report.line(`    Samples:              ${results.memory.sampleCount}`);
    report.line(`    Peak heap:            ${formatBytes(results.memory.peak.heapAlloc)}`);
    report.line(`    Avg heap:             ${formatBytes(results.memory.avgHeapAlloc)}`);
    report.line(`    Final heap:           ${formatBytes(results.memory.final.heapAlloc)}`);
    report.line(`    Peak goroutines:      ${results.memory.peak.goroutines}`);
    report.line(`    Cumulative alloc:     ${formatBytes(results.memory.totalAllocDelta)} (total bytes allocated during run)`);
    report.line(`    GC cycles:            ${results.memory.gcCyclesDelta}`);

    report.line("");
    report.line("  WebSocket");
    report.line(`    Total server frames:  ${results.socket.totalServerFrames}`);
    report.line(`    Total server bytes:   ${formatBytes(results.socket.totalServerBytes)}`);
    report.line(`    Tests tracked:        ${Object.keys(results.socket.perTest).length}`);

    report.line("");
    report.line("  Initial Load (first connection)");
    const channels = Object.entries(results.socket.initialLoad)
        .filter(([ch]) => ch !== "total")
        .sort(([, a], [, b]) => b.bytes - a.bytes);
    for (const [channel, data] of channels) {
        const name = (channel + ":").padEnd(16);
        report.line(`    ${name}${formatBytes(data.bytes).padStart(10)}  (${data.count} frame)`);
    }
    const total = results.socket.initialLoad["total"];
    if (total) {
        report.line(`    ${"─".repeat(28)}`);
        report.line(`    ${"total:".padEnd(16)}${formatBytes(total.bytes).padStart(10)}  (${total.count} frames)`);
    }

    if (results.build) {
        report.line("");
        report.line("  Build Sizes");
        report.line(`    Binary (embedded):    ${formatBytes(results.build.binarySize)}`);
        report.line(`    Bundle (raw):         ${formatBytes(results.build.bundleSizeRaw)} (${results.build.bundleFileCount} files)`);
        report.line(`    Bundle (gzip):        ${formatBytes(results.build.bundleSizeGzip)}`);
    }
}

function formatBytes(bytes: number): string {
    if (bytes < 1024) return `${bytes}B`;
    if (bytes < 1000 * 1024) return `${(bytes / 1024).toFixed(1)}KiB`;
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
