import { describe, test, expect, beforeAll } from "vitest";
import { resetMockState, withAuthClient } from "../src/helpers.js";

describe("services", () => {
    beforeAll(async () => {
        await resetMockState();
    });

    test("startService", async () => {
        await withAuthClient(async (client) => {
            const resp = await client.sendAndReceive("startService", "test-stack", "web");
            expect(resp.ok).toBe(true);
        });
    });

    test("stopService", async () => {
        await withAuthClient(async (client) => {
            const resp = await client.sendAndReceive("stopService", "test-stack", "web");
            expect(resp.ok).toBe(true);
        });
    });

    test("restartService", async () => {
        await withAuthClient(async (client) => {
            const resp = await client.sendAndReceive("restartService", "test-stack", "web");
            expect(resp.ok).toBe(true);
        });
    });

    test("recreateService", async () => {
        await withAuthClient(async (client) => {
            const resp = await client.sendAndReceive("recreateService", "test-stack", "web");
            expect(resp.ok).toBe(true);
        });
    });

    test("updateService", async () => {
        await withAuthClient(async (client) => {
            const resp = await client.sendAndReceive("updateService", "test-stack", "web");
            expect(resp.ok).toBe(true);
        });
    });

    test("checkImageUpdates", async () => {
        await withAuthClient(async (client) => {
            const resp = await client.sendAndReceive("checkImageUpdates", "test-stack");
            expect(resp.ok).toBe(true);
        });
    });

    test("stopContainer", async () => {
        await withAuthClient(async (client) => {
            const resp = await client.sendAndReceive("stopContainer", "test-stack-web-1");
            expect(resp.ok).toBe(true);
        });
    });

    test("restartContainer", async () => {
        await withAuthClient(async (client) => {
            const resp = await client.sendAndReceive("restartContainer", "test-stack-web-1");
            expect(resp.ok).toBe(true);
        });
    });

    test("startContainer", async () => {
        await withAuthClient(async (client) => {
            const resp = await client.sendAndReceive("startContainer", "test-stack-web-1");
            expect(resp.ok).toBe(true);
        });
    });

    test("serviceMissingArgs — empty service/stack name fails", async () => {
        await withAuthClient(async (client) => {
            // Missing service name
            const resp1 = await client.sendAndReceive("startService", "test-stack", "");
            expect(resp1.ok).toBe(false);

            // Missing stack name
            const resp2 = await client.sendAndReceive("stopService", "", "web");
            expect(resp2.ok).toBe(false);
        });
    });
});
