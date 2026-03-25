import { describe, test, expect, beforeAll } from "vitest";
import { resetMockState, connectClient } from "../src/helpers.js";

describe("resource-events", () => {
    beforeAll(async () => {
        await resetMockState();
    });

    test("stopService triggers resourceEvent with container metadata", async () => {
        const client = await connectClient();
        try {
            await client.login();

            // Stop a service — this triggers Docker events that produce resourceEvents
            const actionPromise = client.sendAction("stopService", "test-stack", "web");

            // The resourceEvent should arrive as a push before or during the stop
            const event = await client.waitForEvent("resourceEvent");

            expect(event.type).toBe("container");
            expect(typeof event.action).toBe("string");
            expect(typeof event.id).toBe("string");

            await actionPromise;
        } finally {
            client.close();
        }
    });

    test("resourceEvent has stackName and serviceName for compose containers", async () => {
        const client = await connectClient();
        try {
            await client.login();

            // Restart a service to trigger events with compose labels
            const actionPromise = client.sendAction("restartService", "test-stack", "web");

            const event = await client.waitForEvent("resourceEvent");

            expect(event.type).toBe("container");
            expect(typeof event.action).toBe("string");
            // Compose-managed containers should have stackName and serviceName
            expect(event.stackName).toBe("test-stack");
            expect(event.serviceName).toBe("web");

            await actionPromise;
        } finally {
            client.close();
        }
    });

    test("mutation triggers both resourceEvent and containers broadcast", async () => {
        const client = await connectClient();
        try {
            await client.login();

            // Drain the initial AfterLogin containers broadcast
            await client.waitForEvent("containers");

            // Now stop a service — should trigger both resourceEvent and a new containers broadcast
            const actionPromise = client.sendAction("stopService", "test-stack", "redis");

            const resourceEvent = await client.waitForEvent("resourceEvent");
            expect(resourceEvent.type).toBe("container");

            // A fresh containers broadcast should also arrive from the state change
            const containers = await client.waitForEvent("containers");
            expect(containers).toBeTruthy();

            await actionPromise;
        } finally {
            client.close();
        }
    });
});
