import { describe, test, expect, beforeAll } from "vitest";
import { resetMockState, withAuthClient } from "../src/helpers.js";

describe("settings", () => {
    beforeAll(async () => {
        await resetMockState();
    });

    test("getSettings — jwtSecret filtered out", async () => {
        await withAuthClient(async (client) => {
            const resp = await client.sendAndReceive("getSettings");
            expect(resp.ok).toBe(true);

            const data = resp.data as Record<string, unknown>;
            expect(data).toBeTruthy();
            expect(data).not.toHaveProperty("jwtSecret");
        });
    });

    test("setSettings — set and verify primaryHostname", async () => {
        await withAuthClient(async (client) => {
            const setResp = await client.sendAndReceive("setSettings", {
                primaryHostname: "example.com",
            }, "");
            expect(setResp.ok).toBe(true);

            // Verify round-trip
            const getResp = await client.sendAndReceive("getSettings");
            const data = getResp.data as Record<string, unknown>;
            expect(data.primaryHostname).toBe("example.com");
        });
    });

    test("globalENV round-trip (protocol-only)", async () => {
        await withAuthClient(async (client) => {
            const setResp = await client.sendAndReceive("setSettings", {
                globalENV: "MY_VAR=hello\nOTHER_VAR=world",
            }, "");
            expect(setResp.ok).toBe(true);

            const getResp = await client.sendAndReceive("getSettings");
            const data = getResp.data as Record<string, unknown>;
            expect(data.globalENV).toBe("MY_VAR=hello\nOTHER_VAR=world");
        });
    });

    test("globalENV default deletes (protocol-only)", async () => {
        await withAuthClient(async (client) => {
            // Set a real value first
            await client.sendAndReceive("setSettings", {
                globalENV: "MY_VAR=hello",
            }, "");

            // Set to default content
            const setResp = await client.sendAndReceive("setSettings", {
                globalENV: "# VARIABLE=value #comment",
            }, "");
            expect(setResp.ok).toBe(true);

            // Get should return default
            const getResp = await client.sendAndReceive("getSettings");
            const data = getResp.data as Record<string, unknown>;
            expect(data.globalENV).toBe("# VARIABLE=value #comment");
        });
    });

    test("globalENV empty deletes (protocol-only)", async () => {
        await withAuthClient(async (client) => {
            // Set a value first
            await client.sendAndReceive("setSettings", {
                globalENV: "MY_VAR=hello",
            }, "");

            // Set to empty
            const setResp = await client.sendAndReceive("setSettings", {
                globalENV: "",
            }, "");
            expect(setResp.ok).toBe(true);

            // Get should return default placeholder
            const getResp = await client.sendAndReceive("getSettings");
            const data = getResp.data as Record<string, unknown>;
            expect(data.globalENV).toBe("# VARIABLE=value #comment");
        });
    });

    test("globalENV default on missing — returns placeholder", async () => {
        await withAuthClient(async (client) => {
            const resp = await client.sendAndReceive("getSettings");
            const data = resp.data as Record<string, unknown>;
            expect(data.globalENV).toBe("# VARIABLE=value #comment");
        });
    });
});
