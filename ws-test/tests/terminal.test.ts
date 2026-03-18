import { describe, test, expect } from "vitest";
import { resetMockState, connectClient } from "../src/helpers.js";

describe("terminal", () => {
    test("terminalJoinAndLeave — join combined, verify sessionId, leave", async () => {
        await resetMockState();

        const client = await connectClient();
        try {
            await client.login();

            const joinResp = await client.sendAndReceive("terminalJoin", {
                type: "combined",
                stack: "test-stack",
            });
            expect(joinResp.ok).toBe(true);
            expect(typeof joinResp.sessionId).toBe("number");

            const sessionId = joinResp.sessionId as number;

            const leaveResp = await client.sendAndReceive("terminalLeave", {
                sessionId,
            });
            expect(leaveResp.ok).toBe(true);
        } finally {
            client.close();
        }
    });

    test("terminalJoinCombinedLog — binary output contains actual log content", async () => {
        await resetMockState();

        const client = await connectClient();
        try {
            await client.login();

            const joinResp = await client.sendAndReceive("terminalJoin", {
                type: "combined",
                stack: "test-stack",
            });
            expect(joinResp.ok).toBe(true);
            const sessionId = joinResp.sessionId as number;

            // Collect binary frames and accumulate output
            let output = "";
            for (let i = 0; i < 10; i++) {
                const data = await client.waitForBinary();
                expect(data.length).toBeGreaterThanOrEqual(2);

                // First 2 bytes are session ID (big-endian uint16)
                const gotSession = (data[0] << 8) | data[1];
                expect(gotSession).toBe(sessionId);

                // Remaining bytes are terminal output
                output += data.subarray(2).toString("utf-8");

                // Check if we have enough content — look for nginx or redis log markers
                if (output.includes("nginx") || output.includes("Redis")) {
                    break;
                }
            }

            // test-stack has nginx + redis; combined log should contain startup lines from one or both
            const hasNginx = output.includes("nginx") || output.includes("ready for start up");
            const hasRedis = output.includes("Redis") || output.includes("Ready to accept connections");
            expect(hasNginx || hasRedis).toBe(true);
        } finally {
            client.close();
        }
    });

    test("terminalJoinContainerLog — binary output contains service-specific content", async () => {
        await resetMockState();

        const client = await connectClient();
        try {
            await client.login();

            const joinResp = await client.sendAndReceive("terminalJoin", {
                type: "container-log",
                stack: "test-stack",
                service: "web",
            });
            expect(joinResp.ok).toBe(true);
            const sessionId = joinResp.sessionId as number;

            // Collect binary frames
            let output = "";
            for (let i = 0; i < 10; i++) {
                const data = await client.waitForBinary();
                const gotSession = (data[0] << 8) | data[1];
                expect(gotSession).toBe(sessionId);

                output += data.subarray(2).toString("utf-8");

                if (output.includes("nginx") || output.includes("ready for start up")) {
                    break;
                }
            }

            // web service uses nginx:latest — should contain nginx log content
            expect(output.includes("nginx") || output.includes("ready for start up")).toBe(true);
        } finally {
            client.close();
        }
    });

    test("terminalJoinInvalidType — returns error", async () => {
        await resetMockState();

        const client = await connectClient();
        try {
            await client.login();

            const resp = await client.sendAndReceive("terminalJoin", {
                type: "bogus-type",
            });
            expect(resp.ok).toBe(false);
            expect(resp.msg).toBeDefined();
            expect(String(resp.msg)).toContain("unknown terminal type");
        } finally {
            client.close();
        }
    });

    test("terminalJoinMissingStack — returns error", async () => {
        await resetMockState();

        const client = await connectClient();
        try {
            await client.login();

            const resp = await client.sendAndReceive("terminalJoin", {
                type: "combined",
                // no stack field
            });
            expect(resp.ok).toBe(false);
        } finally {
            client.close();
        }
    });

    test("combinedLogHistoricalOrdering — logs from multiple services are interleaved", async () => {
        await resetMockState();

        const client = await connectClient();
        try {
            await client.login();

            const joinResp = await client.sendAndReceive("terminalJoin", {
                type: "combined",
                stack: "test-stack",
            });
            expect(joinResp.ok).toBe(true);
            const sessionId = joinResp.sessionId as number;

            // Collect binary frames until we have enough output from both services
            let output = "";
            for (let i = 0; i < 20; i++) {
                const data = await client.waitForBinary(5000);
                expect(data.length).toBeGreaterThanOrEqual(2);
                const gotSession = (data[0] << 8) | data[1];
                expect(gotSession).toBe(sessionId);
                output += data.subarray(2).toString("utf-8");

                // Stop when we have enough from both services
                if (output.includes("web") && output.includes("redis") && output.length > 500) break;
            }

            // Split into lines and identify service prefixes
            const lines = output.split("\n").filter(l => l.trim());
            const serviceOrder = lines.map(l => {
                if (l.includes("web")) return "web";
                if (l.includes("redis")) return "redis";
                return null;
            }).filter(Boolean) as string[];

            // Must have lines from both services
            expect(serviceOrder).toContain("web");
            expect(serviceOrder).toContain("redis");

            // Must be interleaved (not all-web-then-all-redis — that was the old broken behavior)
            const firstRedis = serviceOrder.indexOf("redis");
            const lastWeb = serviceOrder.lastIndexOf("web");
            expect(firstRedis).toBeLessThan(lastWeb);

            await client.sendAndReceive("terminalLeave", { sessionId });
        } finally {
            client.close();
        }
    });

    test("terminalLeaveInvalidSession — rejects nonexistent session", async () => {
        await resetMockState();

        const client = await connectClient();
        try {
            await client.login();

            const resp = await client.sendAndReceive("terminalLeave", {
                sessionId: 99999,
            });
            expect(resp.ok).toBe(false);
        } finally {
            client.close();
        }
    });
});
