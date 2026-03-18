import { describe, test, expect, beforeAll } from "vitest";
import { resetMockState, withClient, connectClient } from "../src/helpers.js";

describe("broadcast", () => {
    beforeAll(async () => {
        await resetMockState();
    });

    test("unauthenticatedAccess — protected endpoints fail without login", async () => {
        await withClient(async (client) => {
            const resp = await client.sendAndReceive("getStack", "test-stack");
            expect(resp.ok).toBe(false);
        });
    });

    test("disconnectOtherSocketClients — conn2 closes after disconnect", async () => {
        const client1 = await connectClient();
        const client2 = await connectClient();
        try {
            await client1.login();
            await client2.login();

            const resp = await client1.sendAndReceive("disconnectOtherSocketClients");
            expect(resp.ok).toBe(true);

            // conn2 should be closed
            await client2.waitForClose();
            expect(client2.isClosed).toBe(true);
        } finally {
            client1.close();
            client2.close();
        }
    });

    test("downStackBroadcastsNullContainers — null entries for destroyed containers", async () => {
        await resetMockState();

        // conn1: will issue the downStack command
        const client1 = await connectClient();
        // conn2: will observe broadcast events
        const client2 = await connectClient();
        try {
            await client1.login();
            await client2.login();

            // Wait for initial containers broadcast on conn2 (test-stack starts running)
            const initial = await client2.waitForEvent("containers");

            // Find a test-stack container key
            let foundKey = "";
            for (const key of Object.keys(initial)) {
                if (key.startsWith("test-stack")) {
                    foundKey = key;
                    break;
                }
            }
            expect(foundKey).toBeTruthy();

            // Down the stack on conn1
            const downResp = await client1.sendAndReceive("downStack", "test-stack");
            expect(downResp.ok).toBe(true);

            // Wait for post-down broadcast where container is explicitly null
            for (let i = 0; i < 10; i++) {
                const postDown = await client2.waitForEvent("containers");
                if (postDown[foundKey] === null) {
                    // Container explicitly null — required
                    return;
                }
                // Container still present (e.g. state transitioning), keep reading
            }
            expect.fail("Expected container to be null in post-down broadcast after 10 attempts");
        } finally {
            client1.close();
            client2.close();
        }
    });

    test("eventBroadcastSendsFilteredContainers — only affected stack in broadcast", async () => {
        await resetMockState();

        // Both test-stack and other-stack start running via .mock.yaml.
        // conn1: will issue commands
        const client1 = await connectClient();
        // conn2: observer for broadcasts
        const client2 = await connectClient();
        try {
            await client1.login();
            await client2.login();

            // Wait for initial full containers broadcast on conn2
            const initial = await client2.waitForEvent("containers");

            // Count containers from each stack
            const testStackKeys: string[] = [];
            const otherStackKeys: string[] = [];
            for (const key of Object.keys(initial)) {
                if (key.startsWith("test-stack")) testStackKeys.push(key);
                if (key.startsWith("other-stack")) otherStackKeys.push(key);
            }
            expect(testStackKeys.length).toBeGreaterThan(0);
            expect(otherStackKeys.length).toBeGreaterThan(0);
            const totalInitial = Object.keys(initial).length;

            // Stop test-stack — generates container stop/die events
            const stopResp = await client1.sendAndReceive("stopStack", "test-stack");
            expect(stopResp.ok).toBe(true);

            // Wait for event-driven containers broadcast on conn2
            const postStop = await client2.waitForEvent("containers");

            // The filtered broadcast should contain ONLY test-stack containers
            // (with updated state), NOT containers from other-stack.
            // If the code does a full-list query, we'd see all containers.
            expect(Object.keys(postStop).length).toBeLessThan(totalInitial);

            // Verify the broadcast contains test-stack containers
            let hasTestStack = false;
            for (const key of Object.keys(postStop)) {
                if (key.startsWith("test-stack")) {
                    hasTestStack = true;
                }
                // other-stack containers should NOT be in the filtered broadcast
                if (key.startsWith("other-stack")) {
                    expect.fail(`filtered broadcast should not contain other-stack container "${key}"`);
                }
            }
            expect(hasTestStack).toBe(true);
        } finally {
            client1.close();
            client2.close();
        }
    });
});
