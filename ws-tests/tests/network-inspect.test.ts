import { describe, test, expect, beforeAll } from "vitest";
import { resetMockState, withAuthClient } from "../src/helpers.js";

describe("network", () => {
    beforeAll(async () => {
        await resetMockState();
    });

    test("networkInspect — returns shaped NetworkDetail", async () => {
        await withAuthClient(async (client) => {
            // Get the networks broadcast to find a real network name
            const networks = await client.waitForEvent("networks");
            const name = Object.keys(networks).find((k) => networks[k] !== null);
            expect(name, "should have at least one network").toBeTruthy();

            const resp = await client.sendAndReceive("networkInspect", name);
            expect(resp.ok).toBe(true);

            const detail = resp.networkDetail as Record<string, unknown>;
            expect(detail).toBeTruthy();
            expect(typeof detail.name).toBe("string");
            expect(typeof detail.id).toBe("string");
            expect(typeof detail.driver).toBe("string");
            expect(typeof detail.scope).toBe("string");
            expect(typeof detail.internal).toBe("boolean");
            expect(typeof detail.attachable).toBe("boolean");
            expect(typeof detail.ingress).toBe("boolean");
            expect(typeof detail.ipv6).toBe("boolean");
            expect(typeof detail.created).toBe("string");
            expect(Array.isArray(detail.ipam)).toBe(true);
            expect(Array.isArray(detail.containers)).toBe(true);
        });
    });

    test("networks broadcast — includes internal, attachable, ingress, labels", async () => {
        await withAuthClient(async (client) => {
            const broadcast = await client.waitForEvent("networks");
            const name = Object.keys(broadcast).find((k) => broadcast[k] !== null);
            expect(name, "should have at least one network").toBeTruthy();

            const net = broadcast[name!] as Record<string, unknown>;
            expect(typeof net.internal).toBe("boolean");
            expect(typeof net.attachable).toBe("boolean");
            expect(typeof net.ingress).toBe("boolean");
            expect(net.labels).toBeDefined();
        });
    });
});
