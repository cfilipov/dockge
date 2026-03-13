import { describe, it, expect } from "vitest";
import { generateStats } from "../src/stats.js";
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

describe("generateStats", () => {
    it("returns all required fields", () => {
        const container = makeContainer();
        const clock = new FixedClock(new Date("2025-01-15T01:00:00Z"));
        const stats = generateStats(container, 0, clock);

        expect(stats.read).toBeTruthy();
        expect(stats.preread).toBeTruthy();
        expect(stats.pids_stats).toBeDefined();
        expect(stats.pids_stats.current).toBeGreaterThan(0);
        expect(stats.blkio_stats).toBeDefined();
        expect(stats.cpu_stats).toBeDefined();
        expect(stats.cpu_stats.cpu_usage).toBeDefined();
        expect(stats.cpu_stats.online_cpus).toBe(4);
        expect(stats.precpu_stats).toBeDefined();
        expect(stats.memory_stats).toBeDefined();
        expect(stats.memory_stats.usage).toBeGreaterThan(0);
        expect(stats.memory_stats.limit).toBeGreaterThan(0);
        expect(stats.networks).toBeDefined();
        expect(stats.networks.eth0).toBeDefined();
        expect(stats.name).toBe(container.Name);
        expect(stats.id).toBe(container.Id);
    });

    it("is deterministic for same counter", () => {
        const container = makeContainer();
        const clock1 = new FixedClock(new Date("2025-01-15T01:00:00Z"));
        const clock2 = new FixedClock(new Date("2025-01-15T01:00:00Z"));
        const s1 = generateStats(container, 5, clock1);
        const s2 = generateStats(container, 5, clock2);
        expect(s1).toEqual(s2);
    });

    it("network bytes grow with counter", () => {
        const container = makeContainer();
        const clock = new FixedClock(new Date("2025-01-15T01:00:00Z"));
        const s0 = generateStats(container, 0, clock);
        const s10 = generateStats(container, 10, clock);
        expect(s10.networks.eth0.rx_bytes).toBeGreaterThan(s0.networks.eth0.rx_bytes);
        expect(s10.networks.eth0.tx_bytes).toBeGreaterThan(s0.networks.eth0.tx_bytes);
    });

    it("CPU total_usage grows with counter", () => {
        const container = makeContainer();
        const clock = new FixedClock(new Date("2025-01-15T01:00:00Z"));
        const s1 = generateStats(container, 1, clock);
        const s10 = generateStats(container, 10, clock);
        expect(s10.cpu_stats.cpu_usage.total_usage).toBeGreaterThan(s1.cpu_stats.cpu_usage.total_usage);
    });

    it("has sine-wave variation in CPU", () => {
        const container = makeContainer();
        const clock = new FixedClock(new Date("2025-01-15T01:00:00Z"));
        // Collect CPU over a cycle
        const cpuValues: number[] = [];
        for (let i = 0; i < 64; i++) {
            const stats = generateStats(container, i, clock);
            // compute effective % from total/system ratio
            cpuValues.push(stats.cpu_stats.cpu_usage.total_usage);
        }
        // Values should not all be identical (sine variation)
        const unique = new Set(cpuValues);
        expect(unique.size).toBeGreaterThan(1);
    });

    it("uses container memory limit when set", () => {
        const container = makeContainer(`
services:
  web:
    image: nginx:latest
    mem_limit: 536870912
`);
        const clock = new FixedClock(new Date("2025-01-15T01:00:00Z"));
        const stats = generateStats(container, 0, clock);
        expect(stats.memory_stats.limit).toBe(536870912);
    });

    it("defaults to 1GB memory limit", () => {
        const container = makeContainer();
        const clock = new FixedClock(new Date("2025-01-15T01:00:00Z"));
        const stats = generateStats(container, 0, clock);
        expect(stats.memory_stats.limit).toBe(1_073_741_824);
    });

    it("memory usage is within plausible range", () => {
        const container = makeContainer();
        const clock = new FixedClock(new Date("2025-01-15T01:00:00Z"));
        const stats = generateStats(container, 0, clock);
        // 1 MB to 1 GB
        expect(stats.memory_stats.usage).toBeGreaterThan(1024 * 1024);
        expect(stats.memory_stats.usage).toBeLessThan(1024 * 1024 * 1024);
    });
});
