import { describe, test, expect, beforeAll } from "vitest";
import { resetMockState, connectClient } from "../src/helpers.js";

describe("concurrent", () => {
    beforeAll(async () => {
        await resetMockState();
    });

    test("concurrent stack operations — 5 parallel stopStack", async () => {
        // test-stack starts running via .mock.yaml
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

        // All operations should succeed (serialized via per-stack mutex)
        for (const ok of results) {
            expect(ok).toBe(true);
        }
    });
});
