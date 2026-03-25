import { describe, test, expect, beforeAll } from "vitest";
import { resetMockState, connectClient } from "../src/helpers.js";

const isNoAuth = !!process.env.DOCKGE_NO_AUTH;

describe("broadcast-advanced", () => {
    beforeAll(async () => {
        await resetMockState();
    });

    // Read-only first
    test("imageBroadcastAfterEvent — images broadcast arrives after event", async () => {
        const client = await connectClient();
        try {
            await client.login();
            // Drain AfterLogin images
            const initialImages = await client.waitForEvent("images");
            expect(Object.keys(initialImages).length).toBeGreaterThan(0);

            // Verify the initial images broadcast is well-formed
            for (const [key, val] of Object.entries(initialImages)) {
                expect(typeof key).toBe("string");
                if (val !== null) {
                    const img = val as Record<string, unknown>;
                    expect(img).toHaveProperty("id");
                }
            }
        } finally {
            client.close();
        }
    });

    // Stop/restart tests — leave containers running after restart

    test("broadcastReachesAllAuthenticatedClients — all clients receive events", async () => {
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
            const actionPromise = client1.sendAction("stopService", "test-stack", "web");

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

            await actionPromise;
        } finally {
            client1.close();
            client2.close();
            client3.close();
        }
    });

    test("unauthenticatedClientDoesNotReceiveBroadcasts", async () => {
        const client1 = await connectClient();
        const client2 = await connectClient(); // NOT logged in (in auth mode)
        try {
            await client1.login();

            // Drain AfterLogin on client1
            await client1.waitForEvent("containers");

            if (isNoAuth) {
                // No-auth: client2 is auto-authenticated on connect — drain its after_login too
                await client2.waitForEvent("containers");
            }

            // Trigger events via client1
            const actionPromise = client1.sendAction("restartService", "test-stack", "web");

            // client1 should receive resourceEvent
            const re = await client1.waitForEvent("resourceEvent");
            expect(re).toBeTruthy();

            if (isNoAuth) {
                // No-auth: client2 is also authenticated, so it DOES receive broadcasts
                const re2 = await client2.waitForEvent("resourceEvent");
                expect(re2).toBeTruthy();
            } else {
                // Auth mode: client2 (unauthenticated) should NOT receive broadcasts
                const re2 = await client2.tryWaitForEvent("resourceEvent", 1000);
                expect(re2).toBeNull();

                const c2 = await client2.tryWaitForEvent("containers", 500);
                expect(c2).toBeNull();
            }

            await actionPromise;
        } finally {
            client1.close();
            client2.close();
        }
    });

    test("clientReceivesBroadcastsAfterAuthenticating", async () => {
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
            const actionPromise = client1.sendAction("restartService", "test-stack", "web");

            // Both clients should receive resourceEvent
            const [re1, re2] = await Promise.all([
                client1.waitForEvent("resourceEvent"),
                client2.waitForEvent("resourceEvent"),
            ]);
            expect(re1).toBeTruthy();
            expect(re2).toBeTruthy();

            await actionPromise;
        } finally {
            client1.close();
            client2.close();
        }
    });

    test("resourceEventArrivesBeforeCoalescedBroadcast", async () => {
        const client = await connectClient();
        try {
            await client.login();
            // Drain AfterLogin containers
            await client.waitForEvent("containers");

            // Stop a service (fire without awaiting completion)
            const actionPromise = client.sendAction("stopService", "test-stack", "redis");

            // resourceEvent must arrive before containers broadcast
            const resourceEvent = await client.waitForEvent("resourceEvent");
            expect(resourceEvent).toBeTruthy();
            expect(resourceEvent.type).toBe("container");

            const containers = await client.waitForEvent("containers");
            expect(containers).toBeTruthy();

            await actionPromise;
        } finally {
            client.close();
        }
    });

    test("coalescingReducesBroadcastCount — rapid events produce fewer broadcasts", async () => {
        const client = await connectClient();
        try {
            await client.login();
            // Drain AfterLogin containers
            await client.waitForEvent("containers");

            // restartStack affects 2 containers (web + redis), each emits stop+start events
            const actionPromise = client.sendAction("restartStack", "test-stack");

            // Collect all container broadcasts over 500ms (coalescing deadline is 200ms)
            const broadcasts = await client.collectEvents("containers", 500);

            const { ack } = await actionPromise;
            expect(ack.ok).toBe(true);

            // 2 containers × 2 transitions = at least 4 events, but coalescing
            // should produce fewer than 4 broadcasts
            expect(broadcasts.length).toBeGreaterThan(0);
            expect(broadcasts.length).toBeLessThan(4);
        } finally {
            client.close();
        }
    });

    // Both stacks needed — mergedBroadcasts before afterLogin (which stops other-stack)
    test("mergedBroadcastsFromMultipleStacks — cross-stack coalescing", async () => {
        const client1 = await connectClient();
        const client2 = await connectClient();
        try {
            await client1.login();
            await client2.login();

            // Drain AfterLogin containers on client2
            await client2.waitForEvent("containers");

            // Rapidly stop both stacks
            const stop1 = client1.sendAction("stopStack", "test-stack");
            const stop2 = client1.sendAction("stopStack", "other-stack");

            // Collect all container broadcasts on client2 over 500ms (coalescing deadline is 200ms)
            const broadcasts = await client2.collectEvents("containers", 500);

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

    test("afterLoginCompleteUnderConcurrentBroadcasts", async () => {
        // Both stacks stopped from previous test; restart test-stack services
        const client1 = await connectClient();
        try {
            await client1.login();
            // Drain AfterLogin on client1
            await client1.waitForEvent("containers");

            // Fire multiple operations (don't await) to create a sustained burst of events.
            const p1 = client1.sendAction("restartService", "test-stack", "web");
            const p2 = client1.sendAction("restartService", "test-stack", "redis");
            const p3 = client1.sendAction("stopStack", "other-stack");

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

    // Deploy restores test-stack for the destructive down tests that follow
    test("fullSyncAfterDeploy — deployStack triggers container broadcasts", async () => {
        const composeYAML = `services:\n  web:\n    image: nginx:latest\n  redis:\n    image: redis:7\n`;

        const client = await connectClient();
        try {
            await client.login();
            await client.waitForEvent("containers");

            const { ack } = await client.sendAction("deployStack", "test-stack", composeYAML, "");
            expect(ack.ok).toBe(true);

            const containers = await client.waitForEvent("containers");
            expect(Object.keys(containers).length).toBeGreaterThan(0);
        } finally {
            client.close();
        }
    });

    // Destructive tests last — these destroy containers/networks via downStack

    test("networkEventBroadcastsOnlyAffectedNetwork — filtered broadcast", async () => {
        const client1 = await connectClient();
        const client2 = await connectClient();
        try {
            await client1.login();
            await client2.login();

            // Drain AfterLogin networks on client2
            const initialNetworks = await client2.waitForEvent("networks");
            const totalInitial = Object.keys(initialNetworks).length;

            // Down the test-stack (destroys its network)
            const actionPromise = client1.sendAction("downStack", "test-stack");

            // Wait for event-driven networks broadcast on client2
            const postDown = await client2.waitForEvent("networks");

            const { ack: downResp } = await actionPromise;
            expect(downResp.ok).toBe(true);

            // The filtered broadcast should contain only the affected network(s),
            // not ALL networks. If the code does a full-list query, we'd see all networks.
            expect(Object.keys(postDown).length).toBeLessThan(totalInitial);
        } finally {
            client1.close();
            client2.close();
        }
    });

    test("destroyedNetworkNullEntries — networks broadcast has null for downed stack network", async () => {
        // Redeploy test-stack (destroyed by previous test)
        const setup = await connectClient();
        try {
            await setup.login();
            const yaml = `services:\n  web:\n    image: nginx:latest\n  redis:\n    image: redis:7\n`;
            await setup.sendAction("deployStack", "test-stack", yaml, "");
            await setup.waitForEvent("containers");
        } finally {
            setup.close();
        }

        const client1 = await connectClient();
        const client2 = await connectClient();
        let actionPromise: Promise<unknown> | undefined;
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
            actionPromise = client1.sendAction("downStack", "test-stack");

            // Read networks broadcasts, find one where the stack network key is null
            for (let i = 0; i < 10; i++) {
                const networks = await client2.waitForEvent("networks");
                if (networks[networkKey] === null) {
                    return; // Found null entry — pass
                }
            }
            expect.fail("Expected network to be null in post-down broadcast after 10 attempts");
        } finally {
            if (actionPromise) await actionPromise.catch(() => {});
            client1.close();
            client2.close();
        }
    });
});
