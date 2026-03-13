import { describe, it, expect } from "vitest";
import { generateTop } from "../src/top.js";
import { generateStack } from "../src/generator.js";
import { parseCompose } from "../src/compose-parser.js";
import { parseStackMockConfig } from "../src/mock-config.js";
import { FixedClock } from "../src/clock.js";
import type { GeneratorInput } from "../src/generator.js";

function makeContainer(yaml: string = `
services:
  web:
    image: nginx:latest
    command: nginx -g "daemon off;"
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

describe("generateTop", () => {
    it("returns correct column titles", () => {
        const container = makeContainer();
        const result = generateTop(container);
        expect(result.Titles).toEqual(["UID", "PID", "PPID", "C", "STIME", "TTY", "TIME", "CMD", "VSZ", "RSS", "%MEM"]);
    });

    it("PID 1 uses container command", () => {
        const container = makeContainer();
        const result = generateTop(container);
        expect(result.Processes.length).toBeGreaterThan(0);
        const pid1 = result.Processes[0];
        expect(pid1[1]).toBe("1"); // PID
        expect(pid1[2]).toBe("0"); // PPID
        // CMD should include the container's path
        expect(pid1[7]).toContain("nginx");
    });

    it("PID 1 user from container config", () => {
        const container = makeContainer(`
services:
  web:
    image: nginx:latest
    user: www-data
`);
        const result = generateTop(container);
        expect(result.Processes[0][0]).toBe("www-data");
    });

    it("defaults to root user", () => {
        const container = makeContainer();
        const result = generateTop(container);
        expect(result.Processes[0][0]).toBe("root");
    });

    it("has 2-8 worker processes plus PID 1", () => {
        const container = makeContainer();
        const result = generateTop(container);
        const total = result.Processes.length;
        expect(total).toBeGreaterThanOrEqual(3); // 1 + 2 workers
        expect(total).toBeLessThanOrEqual(9);    // 1 + 8 workers
    });

    it("worker processes have PPID 1", () => {
        const container = makeContainer();
        const result = generateTop(container);
        for (let i = 1; i < result.Processes.length; i++) {
            expect(result.Processes[i][2]).toBe("1");
        }
    });

    it("is deterministic", () => {
        const container = makeContainer();
        const r1 = generateTop(container);
        const r2 = generateTop(container);
        expect(r1).toEqual(r2);
    });

    it("each process row has correct number of columns", () => {
        const container = makeContainer();
        const result = generateTop(container);
        for (const proc of result.Processes) {
            expect(proc).toHaveLength(result.Titles.length);
        }
    });
});
