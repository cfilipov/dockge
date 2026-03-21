import { describe, test, expect, beforeAll } from "vitest";
import { resetMockState, withAuthClient } from "../src/helpers.js";

describe("image", () => {
    beforeAll(async () => {
        await resetMockState();
    });

    test("imageInspect — returns shaped ImageDetail", async () => {
        await withAuthClient(async (client) => {
            // Get the images broadcast (auto-unwrapped from {items: map} → map)
            const images = await client.waitForEvent("images");
            const ids = Object.keys(images);
            expect(ids.length).toBeGreaterThan(0);

            const imageRef = ids[0];

            const resp = await client.sendAndReceive("imageInspect", imageRef);
            expect(resp.ok).toBe(true);

            const detail = resp.imageDetail as Record<string, unknown>;
            expect(detail).toBeTruthy();
            expect(typeof detail.id).toBe("string");
            expect(Array.isArray(detail.repoTags)).toBe(true);
            expect(typeof detail.size).toBe("string");
            expect(typeof detail.created).toBe("string");
            expect(typeof detail.dangling).toBe("boolean");
            expect(typeof detail.architecture).toBe("string");
            expect(typeof detail.os).toBe("string");
            expect(typeof detail.workingDir).toBe("string");
            expect(Array.isArray(detail.layers)).toBe(true);
        });
    });

    test("images broadcast — ImageSummary has string size, string created, dangling", async () => {
        await withAuthClient(async (client) => {
            const images = await client.waitForEvent("images");
            const ids = Object.keys(images);
            expect(ids.length).toBeGreaterThan(0);

            const first = images[ids[0]] as Record<string, unknown>;
            expect(typeof first.id).toBe("string");
            expect(typeof first.size).toBe("string");
            expect(typeof first.created).toBe("string");
            expect(typeof first.dangling).toBe("boolean");
            expect(Array.isArray(first.repoTags)).toBe(true);
        });
    });
});
