import { describe, test, expect, beforeAll } from "vitest";
import { resetMockState, connectClient } from "../src/helpers.js";

describe("events", () => {
    beforeAll(async () => {
        await resetMockState();
    });

    test("afterLogin includes events array with timeNano", async () => {
        const client = await connectClient();
        try {
            await client.login();

            const data = await client.waitForEvent("events");
            const events = data?.items ?? data;
            expect(Array.isArray(events)).toBe(true);
            expect(events.length).toBeGreaterThan(0);

            // Each event should have the expected fields
            const first = events[0];
            expect(typeof first.type).toBe("string");
            expect(typeof first.action).toBe("string");
            expect(typeof first.id).toBe("string");
            expect(typeof first.name).toBe("string");
            expect(typeof first.timeNano).toBe("number");
            expect(first.timeNano).toBeGreaterThan(0);
        } finally {
            client.close();
        }
    });

    test("events are sorted by timeNano", async () => {
        const client = await connectClient();
        try {
            await client.login();

            const data = await client.waitForEvent("events");
            const events = data?.items ?? data;
            expect(Array.isArray(events)).toBe(true);

            for (let i = 1; i < events.length; i++) {
                expect(events[i].timeNano).toBeGreaterThanOrEqual(events[i - 1].timeNano);
            }
        } finally {
            client.close();
        }
    });

    test("events include container create and start for running containers", async () => {
        const client = await connectClient();
        try {
            await client.login();

            const data = await client.waitForEvent("events");
            const events = data?.items ?? data;

            // Should have at least one container create event
            const creates = events.filter(
                (e: any) => e.type === "container" && e.action === "create",
            );
            expect(creates.length).toBeGreaterThan(0);

            // Should have at least one container start event
            const starts = events.filter(
                (e: any) => e.type === "container" && e.action === "start",
            );
            expect(starts.length).toBeGreaterThan(0);
        } finally {
            client.close();
        }
    });

    test("events include network create events", async () => {
        const client = await connectClient();
        try {
            await client.login();

            const data = await client.waitForEvent("events");
            const events = data?.items ?? data;

            const networkCreates = events.filter(
                (e: any) => e.type === "network" && e.action === "create",
            );
            expect(networkCreates.length).toBeGreaterThan(0);
        } finally {
            client.close();
        }
    });

    test("resourceEvent includes timeNano", async () => {
        const client = await connectClient();
        try {
            await client.login();

            // Trigger an action to generate a resourceEvent
            const actionPromise = client.sendAction("stopService", "test-stack", "web");

            const event = await client.waitForEvent("resourceEvent");
            expect(event.type).toBe("container");
            expect(typeof event.timeNano).toBe("number");
            expect(event.timeNano).toBeGreaterThan(0);

            await actionPromise;
        } finally {
            client.close();
        }
    });
});
