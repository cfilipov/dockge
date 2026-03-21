import { describe, test, expect, beforeAll } from "vitest";
import { resetMockState, connectClient } from "../src/helpers.js";

describe("concurrent", () => {
    beforeAll(async () => {
        await resetMockState();
    });

    test("concurrent stack operations — 5 parallel stopStack all succeed and containers stop", async () => {

        // Observer joins compose terminal BEFORE stops so it sees all [Done] markers.
        // It also receives resourceEvent pushes proving containers actually stopped.
        const obs = await connectClient();
        try {
            await obs.login();

            // Drain initial afterLogin broadcasts so they don't pollute event collection
            await obs.waitForEvent("containers");

            await obs.sendAndReceive("terminalJoin", {
                type: "compose",
                stack: "test-stack",
            });

            // Fire 5 parallel stopStack — serialized via per-stack mutex
            const concurrency = 5;
            const results = await Promise.all(
                Array.from({ length: concurrency }, async () => {
                    const client = await connectClient();
                    try {
                        await client.login();
                        const resp = await client.sendAndReceive("stopStack", "test-stack");
                        return resp.ok as boolean;
                    } finally {
                        client.close();
                    }
                }),
            );

            for (const ok of results) {
                expect(ok).toBe(true);
            }

            // Collect resourceEvents to verify containers actually stopped via Docker events.
            // Only the first stop produces real events (the other 4 are no-ops on already-stopped
            // containers). test-stack has web + redis, so we expect stop events for both.
            const stoppedContainers = new Set<string>();
            const collectResourceEvents = async () => {
                while (stoppedContainers.size < 2) {
                    const evt = await obs.waitForEvent("resourceEvent", 10_000);
                    if (evt.action === "stop" && typeof evt.name === "string") {
                        stoppedContainers.add(evt.name);
                    }
                }
            };

            // Wait for all 5 operations to actually complete via [Done] markers,
            // AND collect resourceEvents concurrently.
            const collectTerminalDone = async () => {
                let doneCount = 0;
                let output = "";
                for (let i = 0; i < 200 && doneCount < concurrency; i++) {
                    const data = await obs.waitForBinary(10_000);
                    output += data.subarray(2).toString("utf-8");
                    doneCount = (output.match(/\[Done\]|\[Error\]/g) || []).length;
                }
                return doneCount;
            };

            const [doneCount] = await Promise.all([
                collectTerminalDone(),
                collectResourceEvents(),
            ]);

            expect(doneCount).toBe(concurrency);
            expect(stoppedContainers).toContain("test-stack-web-1");
            expect(stoppedContainers).toContain("test-stack-redis-1");
        } finally {
            obs.close();
        }
    });

    test("concurrent different stacks — cross-stack operations run in parallel", async () => {
        // Previous test stopped test-stack; restart it
        const setup = await connectClient();
        try {
            await setup.login();
            await setup.sendAndReceive("startStack", "test-stack");
            await setup.waitForEvent("resourceEvent");
        } finally {
            setup.close();
        }

        const obs = await connectClient();
        try {
            await obs.login();

            // Drain initial afterLogin broadcasts
            await obs.waitForEvent("containers");

            // Join compose terminals BEFORE sending stops
            for (const stack of ["test-stack", "other-stack"]) {
                await obs.sendAndReceive("terminalJoin", {
                    type: "compose",
                    stack,
                });
            }

            // Stop both stacks concurrently from separate clients
            const [r1, r2] = await Promise.all([
                (async () => {
                    const c = await connectClient();
                    try {
                        await c.login();
                        return await c.sendAndReceive("stopStack", "test-stack");
                    } finally {
                        c.close();
                    }
                })(),
                (async () => {
                    const c = await connectClient();
                    try {
                        await c.login();
                        return await c.sendAndReceive("stopStack", "other-stack");
                    } finally {
                        c.close();
                    }
                })(),
            ]);

            expect(r1.ok).toBe(true);
            expect(r2.ok).toBe(true);

            // Collect resourceEvents to verify containers from both stacks stopped
            const stoppedContainers = new Set<string>();
            const collectResourceEvents = async () => {
                // test-stack has web + redis, other-stack has app = 3 containers
                while (stoppedContainers.size < 3) {
                    const evt = await obs.waitForEvent("resourceEvent", 10_000);
                    if (evt.action === "stop" && typeof evt.name === "string") {
                        stoppedContainers.add(evt.name);
                    }
                }
            };

            // Wait for both operations to complete (1 [Done] per stack = 2 total)
            const collectTerminalDone = async () => {
                let doneCount = 0;
                let output = "";
                for (let i = 0; i < 100 && doneCount < 2; i++) {
                    const data = await obs.waitForBinary(10_000);
                    output += data.subarray(2).toString("utf-8");
                    doneCount = (output.match(/\[Done\]|\[Error\]/g) || []).length;
                }
                return doneCount;
            };

            const [doneCount] = await Promise.all([
                collectTerminalDone(),
                collectResourceEvents(),
            ]);

            expect(doneCount).toBe(2);
            expect(stoppedContainers).toContain("test-stack-web-1");
            expect(stoppedContainers).toContain("test-stack-redis-1");
            expect(stoppedContainers).toContain("other-stack-app-1");
        } finally {
            obs.close();
        }
    });
});
