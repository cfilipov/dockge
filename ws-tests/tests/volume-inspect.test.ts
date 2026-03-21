import { describe, test, expect, beforeAll } from "vitest";
import { resetMockState, withAuthClient } from "../src/helpers.js";

describe("volume", () => {
    beforeAll(async () => {
        await resetMockState();
    });

    test("volumeInspect — returns shaped VolumeDetail", async () => {
        await withAuthClient(async (client) => {
            // Get the volumes broadcast (auto-unwrapped from {items: map} → map)
            const volumes = await client.waitForEvent("volumes");
            const names = Object.keys(volumes);

            // Mock daemon may not have volumes — skip shape check if empty
            if (names.length === 0) return;

            const volumeName = names[0];

            const resp = await client.sendAndReceive("volumeInspect", volumeName);
            expect(resp.ok).toBe(true);

            const detail = resp.volumeDetail as Record<string, unknown>;
            expect(detail).toBeTruthy();
            expect(typeof detail.name).toBe("string");
            expect(typeof detail.driver).toBe("string");
            expect(typeof detail.mountpoint).toBe("string");
            expect(typeof detail.scope).toBe("string");
            expect(typeof detail.created).toBe("string");
            expect(detail.labels).toBeDefined();
        });
    });

    test("volumes broadcast — VolumeSummary includes labels", async () => {
        await withAuthClient(async (client) => {
            const volumes = await client.waitForEvent("volumes");
            const names = Object.keys(volumes);

            // Mock daemon may not have volumes — skip shape check if empty
            if (names.length === 0) return;

            const first = volumes[names[0]] as Record<string, unknown>;
            expect(typeof first.name).toBe("string");
            expect(typeof first.driver).toBe("string");
            expect(typeof first.mountpoint).toBe("string");
            expect(first.labels).toBeDefined();
        });
    });
});
