import { describe, test, expect, beforeAll } from "vitest";
import { resetMockState, withAuthClient } from "../src/helpers.js";

describe("terminal", () => {
    beforeAll(async () => {
        await resetMockState();
    });

    test("terminalJoinAndLeave — join combined, verify sessionId, leave", async () => {
        await withAuthClient(async (client) => {
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
        });
    });

    test("terminalJoinCombinedLog — binary frame with session header", async () => {
        await withAuthClient(async (client) => {
            const joinResp = await client.sendAndReceive("terminalJoin", {
                type: "combined",
                stack: "test-stack",
            });
            expect(joinResp.ok).toBe(true);
            const sessionId = joinResp.sessionId as number;

            // Wait for binary output frame
            const data = await client.waitForBinary();
            expect(data.length).toBeGreaterThanOrEqual(2);

            // First 2 bytes are session ID (big-endian uint16)
            const gotSession = (data[0] << 8) | data[1];
            expect(gotSession).toBe(sessionId);

            // Remaining bytes are terminal output
            expect(data.length).toBeGreaterThan(2);
        });
    });
});
