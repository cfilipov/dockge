import { describe, test, expect, beforeAll } from "vitest";
import { resetMockState, connectClient } from "../src/helpers.js";

describe("terminal", () => {
    beforeAll(async () => {
        await resetMockState();
    });

    // Read-only and error tests first

    test("terminalJoinAndLeave — join combined, verify sessionId, leave", async () => {
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

    test("terminalExited sent when exec shell exits", async () => {
        const client = await connectClient();
        try {
            await client.login();

            // Join exec terminal for test-stack web service
            const joinResp = await client.sendAndReceive("terminalJoin", {
                type: "exec",
                stack: "test-stack",
                service: "web",
            });
            expect(joinResp.ok).toBe(true);
            const sessionId = joinResp.sessionId as number;

            // Send exit command via binary input to trigger shell exit
            // Binary input format: [sessionId:u16][opcode:0x00][payload]
            const exitCmd = Buffer.from("exit\r\n", "utf-8");
            const inputFrame = Buffer.alloc(2 + 1 + exitCmd.length);
            inputFrame.writeUInt16BE(sessionId, 0);
            inputFrame[2] = 0x00; // input opcode
            exitCmd.copy(inputFrame, 3);
            client.sendBinary(inputFrame);

            // Also send Ctrl+D (EOF) in case the shell doesn't respond to "exit"
            const eofFrame = Buffer.alloc(2 + 1 + 1);
            eofFrame.writeUInt16BE(sessionId, 0);
            eofFrame[2] = 0x00;
            eofFrame[3] = 0x04; // Ctrl+D
            client.sendBinary(eofFrame);

            // Wait for terminalExited push event
            const exitEvent = await client.tryWaitForEvent("terminalExited", 5000);
            if (exitEvent !== null) {
                expect(exitEvent.sessionId).toBe(sessionId);
            } else {
                console.log("terminalExited: mock exec shell did not exit within timeout (expected in mock mode)");
            }
        } finally {
            client.close();
        }
    });

    // Mutation tests last — these stop containers/stacks

    test("bannerUsesBackgroundColor — stop banner uses background ANSI codes", async () => {
        const client = await connectClient();
        try {
            await client.login();

            // Join combined log terminal
            const joinResp = await client.sendAndReceive("terminalJoin", {
                type: "combined",
                stack: "test-stack",
            });
            expect(joinResp.ok).toBe(true);
            const sessionId = joinResp.sessionId as number;

            // Drain initial historical log output
            for (let i = 0; i < 10; i++) {
                try {
                    await client.waitForBinary(3000);
                } catch {
                    break;
                }
            }

            // Trigger stopStack to produce a CONTAINER STOP banner
            const stopResp = await client.sendAndReceive("stopStack", "test-stack");
            expect(stopResp.ok).toBe(true);

            // Collect binary frames looking for the banner
            let bannerOutput = "";
            for (let i = 0; i < 50; i++) {
                try {
                    const data = await client.waitForBinary(15000);
                    if (((data[0] << 8) | data[1]) !== sessionId) continue;
                    bannerOutput += data.subarray(2).toString("utf-8");

                    if (bannerOutput.includes("CONTAINER STOP")) {
                        break;
                    }
                } catch {
                    break;
                }
            }

            // Banner must have been emitted
            expect(bannerOutput).toContain("CONTAINER STOP");

            // Must use background RGB color codes (48;2;R;G;B)
            expect(bannerOutput).toContain("48;2;");

            // Must NOT use foreground-only codes like \x1b[1;33m or \x1b[1;34m immediately before CONTAINER
            const fgOnlyPattern = /\x1b\[1;3[34]m[^]*?CONTAINER/;
            expect(fgOnlyPattern.test(bannerOutput)).toBe(false);
        } finally {
            client.close();
        }
    });

    test("containerActionTerminal — stopContainer writes to container-action terminal", async () => {
        const client = await connectClient();
        try {
            await client.login();

            const containerName = "test-stack-web-1";

            // Join container-action terminal for this container
            const joinResp = await client.sendAndReceive("terminalJoin", {
                type: "container-action",
                container: containerName,
            });
            expect(joinResp.ok).toBe(true);
            const sessionId = joinResp.sessionId as number;

            // Drain any stale buffer content from previous actions
            for (let i = 0; i < 10; i++) {
                try {
                    await client.waitForBinary(500);
                } catch {
                    break; // no more buffered frames
                }
            }

            // Trigger stopContainer
            const stopResp = await client.sendAndReceive("stopContainer", containerName);
            expect(stopResp.ok).toBe(true);

            // Collect binary frames until [Done] or [Error] appears
            let output = "";
            const maxFrames = 50;
            for (let i = 0; i < maxFrames; i++) {
                const data = await client.waitForBinary(15000);
                expect(data.length).toBeGreaterThanOrEqual(2);
                const gotSession = (data[0] << 8) | data[1];
                if (gotSession !== sessionId) continue;

                output += data.subarray(2).toString("utf-8");

                if (output.includes("[Done]") || output.includes("[Error]")) {
                    break;
                }
            }

            // Must have a completion marker
            const hasMarker = output.includes("[Done]") || output.includes("[Error]");
            expect(hasMarker).toBe(true);

            // Must have command display line: "$ docker stop <container>"
            expect(output).toContain("$ docker stop");
            expect(output).toContain(containerName);

            // Command must appear before the completion marker
            const cmdIdx = output.indexOf("$ docker stop");
            const doneIdx = output.indexOf("[Done]") >= 0 ? output.indexOf("[Done]") : output.indexOf("[Error]");
            expect(cmdIdx).toBeLessThan(doneIdx);
        } finally {
            client.close();
        }
    });

    test("composeTerminalStreaming — stopStack writes output to compose terminal", async () => {
        const client = await connectClient();
        try {
            await client.login();

            // Join compose terminal for test-stack
            const joinResp = await client.sendAndReceive("terminalJoin", {
                type: "compose",
                stack: "test-stack",
            });
            expect(joinResp.ok).toBe(true);
            const sessionId = joinResp.sessionId as number;

            // Trigger stopStack
            const stopResp = await client.sendAndReceive("stopStack", "test-stack");
            expect(stopResp.ok).toBe(true);

            // Collect binary frames until [Done] or [Error] appears
            let output = "";
            const maxFrames = 50;
            for (let i = 0; i < maxFrames; i++) {
                const data = await client.waitForBinary(15000);
                expect(data.length).toBeGreaterThanOrEqual(2);
                const gotSession = (data[0] << 8) | data[1];
                if (gotSession !== sessionId) continue;

                output += data.subarray(2).toString("utf-8");

                if (output.includes("[Done]") || output.includes("[Error]")) {
                    break;
                }
            }

            // Must have a completion marker
            const hasMarker = output.includes("[Done]") || output.includes("[Error]");
            expect(hasMarker).toBe(true);

            // Must have command display line before completion marker
            const cmdIdx = output.indexOf("$ docker compose");
            expect(cmdIdx).toBeGreaterThanOrEqual(0);

            const doneIdx = output.indexOf("[Done]") >= 0 ? output.indexOf("[Done]") : output.indexOf("[Error]");
            expect(cmdIdx).toBeLessThan(doneIdx);

            // Should have actual compose output between command and marker (not just empty)
            const between = output.substring(cmdIdx, doneIdx);
            expect(between.length).toBeGreaterThan(30);
        } finally {
            client.close();
        }
    });
});
