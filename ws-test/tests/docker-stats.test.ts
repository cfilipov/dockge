import { describe, test, expect } from "vitest";
import { resetMockState, withAuthClient } from "../src/helpers.js";

describe("subscribeStats", () => {
    test("receives formatted stats for subscribed container", async () => {
        await resetMockState();

        await withAuthClient(async (client) => {
            const containerName = "test-stack-web-1";

            // Subscribe to stats for a known container
            const ack = await client.sendAndReceive("subscribeStats", containerName);
            expect(ack.ok).toBe(true);

            // Wait for a dockerStats push event
            const stats = await client.waitForEvent("dockerStats");
            expect(stats.ok).toBe(true);

            const dockerStats = stats.dockerStats as Record<string, Record<string, string>>;
            expect(dockerStats).toBeDefined();

            const containerStats = dockerStats[containerName];
            expect(containerStats).toBeDefined();

            // Verify PascalCase field names exist
            expect(containerStats).toHaveProperty("Name");
            expect(containerStats).toHaveProperty("CPUPerc");
            expect(containerStats).toHaveProperty("MemPerc");
            expect(containerStats).toHaveProperty("MemUsage");
            expect(containerStats).toHaveProperty("NetIO");
            expect(containerStats).toHaveProperty("BlockIO");
            expect(containerStats).toHaveProperty("PIDs");

            // Verify Name matches container
            expect(containerStats.Name).toBe(containerName);

            // Verify formatted string patterns
            expect(containerStats.CPUPerc).toMatch(/^\d+\.\d+%$/);
            expect(containerStats.MemPerc).toMatch(/^\d+\.\d+%$/);
            expect(containerStats.MemUsage).toMatch(/^\S+ \/ \S+$/);
            expect(containerStats.NetIO).toMatch(/^\S+ \/ \S+$/);
            expect(containerStats.BlockIO).toMatch(/^\S+ \/ \S+$/);
            expect(containerStats.PIDs).toMatch(/^\d+$/);

            // Unsubscribe
            const unsub = await client.sendAndReceive("unsubscribeStats");
            expect(unsub.ok).toBe(true);
        });
    });
});
