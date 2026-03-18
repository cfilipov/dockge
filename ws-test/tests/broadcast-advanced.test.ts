import { describe, test, expect, beforeAll } from "vitest";
import { resetMockState, connectClient } from "../src/helpers.js";

describe("broadcast-advanced", () => {
    beforeAll(async () => {
        await resetMockState();
    });

    test("destroyedNetworkNullEntries — networks broadcast has null for downed stack network", async () => {
        await resetMockState();

        const client1 = await connectClient();
        const client2 = await connectClient();
        try {
            await client1.login();
            await client2.login();

            // Drain AfterLogin networks on client2, find test-stack network key
            const initialNetworks = await client2.waitForEvent("networks");
            let networkKey = "";
            for (const key of Object.keys(initialNetworks)) {
                if (key.includes("test-stack") || key.startsWith("test-stack")) {
                    networkKey = key;
                    break;
                }
            }
            expect(networkKey).toBeTruthy();

            // Down the stack on client1
            const downResp = await client1.sendAndReceive("downStack", "test-stack");
            expect(downResp.ok).toBe(true);

            // Read networks broadcasts, find one where the stack network key is null
            for (let i = 0; i < 10; i++) {
                const networks = await client2.waitForEvent("networks");
                if (networks[networkKey] === null) {
                    return; // Found null entry — pass
                }
            }
            expect.fail("Expected network to be null in post-down broadcast after 10 attempts");
        } finally {
            client1.close();
            client2.close();
        }
    });

    test("broadcastReachesAllAuthenticatedClients — all clients receive events", async () => {
        await resetMockState();

        const client1 = await connectClient();
        const client2 = await connectClient();
        const client3 = await connectClient();
        try {
            await client1.login();
            await client2.login();
            await client3.login();

            // Drain AfterLogin containers on all
            await client1.waitForEvent("containers");
            await client2.waitForEvent("containers");
            await client3.waitForEvent("containers");

            // Trigger an event
            const stopResp = await client1.sendAndReceive("stopService", "test-stack", "web");
            expect(stopResp.ok).toBe(true);

            // All 3 clients should receive resourceEvent and containers broadcast
            const [re1, re2, re3] = await Promise.all([
                client1.waitForEvent("resourceEvent"),
                client2.waitForEvent("resourceEvent"),
                client3.waitForEvent("resourceEvent"),
            ]);
            expect(re1).toBeTruthy();
            expect(re2).toBeTruthy();
            expect(re3).toBeTruthy();

            const [c1, c2, c3] = await Promise.all([
                client1.waitForEvent("containers"),
                client2.waitForEvent("containers"),
                client3.waitForEvent("containers"),
            ]);
            expect(c1).toBeTruthy();
            expect(c2).toBeTruthy();
            expect(c3).toBeTruthy();
        } finally {
            client1.close();
            client2.close();
            client3.close();
        }
    });

    test("unauthenticatedClientDoesNotReceiveBroadcasts", async () => {
        await resetMockState();

        const client1 = await connectClient();
        const client2 = await connectClient(); // NOT logged in
        try {
            await client1.login();

            // Drain AfterLogin on client1
            await client1.waitForEvent("containers");

            // Trigger events via client1
            const resp = await client1.sendAndReceive("restartService", "test-stack", "web");
            expect(resp.ok).toBe(true);

            // client1 should receive resourceEvent
            const re = await client1.waitForEvent("resourceEvent");
            expect(re).toBeTruthy();

            // client2 (unauthenticated) should NOT receive broadcasts
            const re2 = await client2.tryWaitForEvent("resourceEvent", 1000);
            expect(re2).toBeNull();

            const c2 = await client2.tryWaitForEvent("containers", 500);
            expect(c2).toBeNull();
        } finally {
            client1.close();
            client2.close();
        }
    });

    test("clientReceivesBroadcastsAfterAuthenticating", async () => {
        await resetMockState();

        const client1 = await connectClient();
        const client2 = await connectClient();
        try {
            await client1.login();
            // Drain AfterLogin on client1
            await client1.waitForEvent("containers");

            // Now login client2 and drain its AfterLogin pushes
            await client2.login();
            await client2.waitForEvent("containers");

            // Trigger events
            const resp = await client1.sendAndReceive("restartService", "test-stack", "web");
            expect(resp.ok).toBe(true);

            // Both clients should receive resourceEvent
            const [re1, re2] = await Promise.all([
                client1.waitForEvent("resourceEvent"),
                client2.waitForEvent("resourceEvent"),
            ]);
            expect(re1).toBeTruthy();
            expect(re2).toBeTruthy();
        } finally {
            client1.close();
            client2.close();
        }
    });

    test("resourceEventArrivesBeforeCoalescedBroadcast", async () => {
        await resetMockState();

        const client = await connectClient();
        try {
            await client.login();
            // Drain AfterLogin containers
            await client.waitForEvent("containers");

            // Stop a service (send without awaiting ack yet)
            const stopPromise = client.sendAndReceive("stopService", "test-stack", "redis");

            // resourceEvent must arrive before containers broadcast
            const resourceEvent = await client.waitForEvent("resourceEvent");
            expect(resourceEvent).toBeTruthy();
            expect(resourceEvent.type).toBe("container");

            const containers = await client.waitForEvent("containers");
            expect(containers).toBeTruthy();

            await stopPromise;
        } finally {
            client.close();
        }
    });

    test("coalescingReducesBroadcastCount — rapid events produce fewer broadcasts", async () => {
        await resetMockState();

        const client = await connectClient();
        try {
            await client.login();
            // Drain AfterLogin containers
            await client.waitForEvent("containers");

            // restartStack affects 2 containers (web + redis), each emits stop+start events
            const resp = await client.sendAndReceive("restartStack", "test-stack");
            expect(resp.ok).toBe(true);

            // Collect all container broadcasts over 2s
            const broadcasts = await client.collectEvents("containers", 2000);

            // 2 containers × 2 transitions = at least 4 events, but coalescing
            // should produce fewer than 4 broadcasts
            expect(broadcasts.length).toBeGreaterThan(0);
            expect(broadcasts.length).toBeLessThan(4);
        } finally {
            client.close();
        }
    });

    test("afterLoginCompleteUnderConcurrentBroadcasts", async () => {
        await resetMockState();

        const client1 = await connectClient();
        try {
            await client1.login();
            // Drain AfterLogin on client1
            await client1.waitForEvent("containers");

            // Fire multiple operations (don't await) to create a sustained burst of events.
            // A single restart may complete before client2 connects; multiple operations
            // affecting multiple containers across both stacks widen the broadcast window.
            const p1 = client1.sendAndReceive("restartService", "test-stack", "web");
            const p2 = client1.sendAndReceive("restartService", "test-stack", "redis");
            const p3 = client1.sendAndReceive("stopStack", "other-stack");

            // Immediately connect and login client2
            const client2 = await connectClient();
            try {
                await client2.login();

                // client2's AfterLogin push should be a complete snapshot
                const initialContainers = await client2.waitForEvent("containers");

                // Should contain entries from both stacks
                let hasTestStack = false;
                let hasOtherStack = false;
                for (const key of Object.keys(initialContainers)) {
                    if (key.startsWith("test-stack")) hasTestStack = true;
                    if (key.startsWith("other-stack")) hasOtherStack = true;
                }
                expect(hasTestStack).toBe(true);
                expect(hasOtherStack).toBe(true);
            } finally {
                client2.close();
            }

            await Promise.all([p1, p2, p3]);
        } finally {
            client1.close();
        }
    });

    test("mergedBroadcastsFromMultipleStacks — cross-stack coalescing", async () => {
        await resetMockState();

        const client1 = await connectClient();
        const client2 = await connectClient();
        try {
            await client1.login();
            await client2.login();

            // Drain AfterLogin containers on client2
            await client2.waitForEvent("containers");

            // Rapidly stop both stacks
            const stop1 = client1.sendAndReceive("stopStack", "test-stack");
            const stop2 = client1.sendAndReceive("stopStack", "other-stack");

            // Collect all container broadcasts on client2 over 2s
            const broadcasts = await client2.collectEvents("containers", 2000);

            await Promise.all([stop1, stop2]);

            // Two stacks with 3 total containers stopping would produce many events,
            // but coalescing should merge them into few broadcasts (≤ 3)
            expect(broadcasts.length).toBeGreaterThan(0);
            expect(broadcasts.length).toBeLessThanOrEqual(3);
        } finally {
            client1.close();
            client2.close();
        }
    });

    test("fullSyncAfterDeploy — deployStack triggers container broadcasts", async () => {
        await resetMockState();

        const composeYAML = `services:\n  web:\n    image: nginx:latest\n  redis:\n    image: redis:7\n`;

        const client = await connectClient();
        try {
            await client.login();
            await client.waitForEvent("containers");

            const resp = await client.sendAndReceive("deployStack", "test-stack", composeYAML, "");
            expect(resp.ok).toBe(true);

            const containers = await client.waitForEvent("containers");
            expect(Object.keys(containers).length).toBeGreaterThan(0);
        } finally {
            client.close();
        }
    });
});
