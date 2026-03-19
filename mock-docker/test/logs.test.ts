import { describe, it, expect } from "vitest";
import { generateStartupLogs, generateShutdownLogs, generatePeriodicLogLine, getHistoricalLogs } from "../src/logs.js";
import { generateStack } from "../src/generator.js";
import { parseCompose } from "../src/compose-parser.js";
import { parseStackMockConfig } from "../src/mock-config.js";
import { FixedClock } from "../src/clock.js";
import type { GeneratorInput } from "../src/generator.js";

function makeContainer(yaml: string = `
services:
  web:
    image: nginx:latest
    ports:
      - "8080:80"
`) {
    const input: GeneratorInput = {
        project: "test-project",
        stackDir: "/opt/stacks/test-project",
        composeFilePath: "/opt/stacks/test-project/compose.yaml",
        parsed: parseCompose(yaml),
        mockConfig: parseStackMockConfig(null),
        clock: new FixedClock(new Date("2025-01-15T00:00:00Z")),
    };
    return generateStack(input).containers[0];
}

describe("generateStartupLogs", () => {
    it("returns 5-8 lines", () => {
        const container = makeContainer();
        const clock = new FixedClock(new Date("2025-01-15T00:00:00Z"));
        const lines = generateStartupLogs(container, clock);
        expect(lines.length).toBeGreaterThanOrEqual(5);
        expect(lines.length).toBeLessThanOrEqual(8);
    });

    it("includes Listening on port with first exposed port", () => {
        const container = makeContainer();
        const clock = new FixedClock(new Date("2025-01-15T00:00:00Z"));
        const lines = generateStartupLogs(container, clock);
        const hasPort = lines.some((l) => l.includes("Listening on port 80"));
        expect(hasPort).toBe(true);
    });

    it("is deterministic", () => {
        const container = makeContainer();
        const clock1 = new FixedClock(new Date("2025-01-15T00:00:00Z"));
        const clock2 = new FixedClock(new Date("2025-01-15T00:00:00Z"));
        expect(generateStartupLogs(container, clock1)).toEqual(generateStartupLogs(container, clock2));
    });

    it("each line has timestamp format", () => {
        const container = makeContainer();
        const clock = new FixedClock(new Date("2025-01-15T00:00:00Z"));
        const lines = generateStartupLogs(container, clock);
        for (const line of lines) {
            // ISO 8601 timestamp at start
            expect(line).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}/);
        }
    });
});

describe("generateShutdownLogs", () => {
    it("returns 2-3 lines", () => {
        const container = makeContainer();
        const clock = new FixedClock(new Date("2025-01-15T00:00:00Z"));
        const lines = generateShutdownLogs(container, clock);
        expect(lines.length).toBeGreaterThanOrEqual(2);
        expect(lines.length).toBeLessThanOrEqual(3);
    });

    it("includes shutdown message", () => {
        const container = makeContainer();
        const clock = new FixedClock(new Date("2025-01-15T00:00:00Z"));
        const lines = generateShutdownLogs(container, clock);
        const hasShutdown = lines.some((l) =>
            l.includes("shutdown") || l.includes("Closing") || l.includes("Received"),
        );
        expect(hasShutdown).toBe(true);
    });
});

describe("generatePeriodicLogLine", () => {
    it("returns a formatted log line", () => {
        const container = makeContainer();
        const clock = new FixedClock(new Date("2025-01-15T01:00:00Z"));
        const line = generatePeriodicLogLine(container, 0, clock);
        // Should match: timestamp level [component] message
        expect(line).toMatch(/^\d{4}-\d{2}-\d{2}T.+ (INFO|DEBUG|WARN|ERROR) \[.+\] .+/);
    });

    it("is deterministic for same line number", () => {
        const container = makeContainer();
        const clock = new FixedClock(new Date("2025-01-15T01:00:00Z"));
        const line1 = generatePeriodicLogLine(container, 42, clock);
        const line2 = generatePeriodicLogLine(container, 42, clock);
        expect(line1).toBe(line2);
    });

    it("varies with different line numbers", () => {
        const container = makeContainer();
        const clock = new FixedClock(new Date("2025-01-15T01:00:00Z"));
        const lines = new Set<string>();
        for (let i = 0; i < 20; i++) {
            // Strip timestamp for comparison since clock is fixed
            const line = generatePeriodicLogLine(container, i, clock);
            const withoutTs = line.replace(/^\d{4}-\d{2}-\d{2}T\S+ /, "");
            lines.add(withoutTs);
        }
        // Should have at least a few unique messages
        expect(lines.size).toBeGreaterThan(3);
    });
});

describe("getHistoricalLogs", () => {
    it("returns 100 lines by default", () => {
        const container = makeContainer();
        const clock = new FixedClock(new Date("2025-01-15T01:00:00Z"));
        const lines = getHistoricalLogs(container, clock);
        expect(lines).toHaveLength(100);
    });

    it("respects tail parameter", () => {
        const container = makeContainer();
        const clock = new FixedClock(new Date("2025-01-15T01:00:00Z"));
        const lines = getHistoricalLogs(container, clock, { tail: 10 });
        expect(lines).toHaveLength(10);
    });

    it("tail=0 returns empty", () => {
        const container = makeContainer();
        const clock = new FixedClock(new Date("2025-01-15T01:00:00Z"));
        const lines = getHistoricalLogs(container, clock, { tail: 0 });
        expect(lines).toHaveLength(0);
    });

    it("is deterministic", () => {
        const container = makeContainer();
        const clock1 = new FixedClock(new Date("2025-01-15T01:00:00Z"));
        const clock2 = new FixedClock(new Date("2025-01-15T01:00:00Z"));
        expect(getHistoricalLogs(container, clock1, { tail: 5 }))
            .toEqual(getHistoricalLogs(container, clock2, { tail: 5 }));
    });

    it("each line has a timestamp", () => {
        const container = makeContainer();
        const clock = new FixedClock(new Date("2025-01-15T01:00:00Z"));
        const lines = getHistoricalLogs(container, clock, { tail: 10 });
        for (const entry of lines) {
            expect(entry.line).toMatch(/^\d{4}-\d{2}-\d{2}T/);
        }
    });
});
