import { describe, test, expect } from "vitest";
import { withClient } from "../src/helpers.js";

describe("connect", () => {
    test("info event on connect — version, dev fields present", async () => {
        await withClient(async (client) => {
            // The server sends "info" immediately on WebSocket connect (before auth)
            const info = await client.waitForEvent("info");
            expect(info.version).toBeTruthy();
            expect(typeof info.version).toBe("string");
            expect(typeof info.latestVersion).toBe("string");
            expect(typeof info.isContainer).toBe("boolean");
            expect(typeof info.dev).toBe("boolean");
            // In --dev mode, dev should be true
            expect(info.dev).toBe(true);
        });
    });
});
