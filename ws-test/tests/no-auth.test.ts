import { describe, test, expect } from "vitest";
import { connectClient } from "../src/helpers.js";

const isNoAuth = !!process.env.DOCKGE_NO_AUTH;

describe("no-auth mode", () => {
    test("protected endpoint works without login", async () => {
        if (!isNoAuth) {
            // Auth mode: this test verifies no-auth behavior — pass trivially
            expect(true).toBe(true);
            return;
        }
        const client = await connectClient();
        try {
            const resp = await client.sendAndReceive("getStack", "test-stack");
            expect(resp.ok).toBe(true);
        } finally {
            client.close();
        }
    });

    test("initial data events arrive without login", async () => {
        if (!isNoAuth) {
            // Auth mode: this test verifies no-auth behavior — pass trivially
            expect(true).toBe(true);
            return;
        }
        const client = await connectClient();
        try {
            const [stacks, containers] = await Promise.all([
                client.waitForEvent("stacks"),
                client.waitForEvent("containers"),
            ]);
            expect(stacks).toBeDefined();
            expect(containers).toBeDefined();
        } finally {
            client.close();
        }
    });
});
