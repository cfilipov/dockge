import { describe, test, expect } from "vitest";
import { resetMockState, connectClient, waitForContainerState } from "../src/helpers.js";

describe("concurrent", () => {
    test("concurrent stack operations — 5 parallel stopStack all succeed and containers stop", async () => {
        await resetMockState();

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

        // Verify containers actually reached exited state.
        // The background goroutines may still be running, so poll broadcasts
        // rather than relying on the AfterLogin snapshot.
        const obs = await connectClient();
        try {
            await obs.login();
            await waitForContainerState(obs, "test-stack-web-1", "exited");
        } finally {
            obs.close();
        }
    });

    test("concurrent different stacks — cross-stack operations run in parallel", async () => {
        await resetMockState();

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

        // Verify containers from both stacks reached exited state
        const obs = await connectClient();
        try {
            await obs.login();
            await waitForContainerState(obs, "test-stack-web-1", "exited");
        } finally {
            obs.close();
        }

        // Check other-stack separately (fresh client to avoid consumed broadcasts)
        const obs2 = await connectClient();
        try {
            await obs2.login();
            await waitForContainerState(obs2, "other-stack-app-1", "exited");
        } finally {
            obs2.close();
        }
    });
});
