import { describe, test, expect, beforeAll } from "vitest";
import { resetMockState, withAuthClient } from "../src/helpers.js";

describe("docker-resources", () => {
    beforeAll(async () => {
        await resetMockState();
    });

    test("dockerStats — subscribe, receive push, unsubscribe", async () => {
        await withAuthClient(async (client) => {
            const subResp = await client.sendAndReceive("subscribeStats", "test-stack-web-1");
            expect(subResp.ok).toBe(true);

            const pushed = await client.waitForEvent("dockerStats");
            expect(pushed.ok).toBe(true);
            const stats = pushed.dockerStats as Record<string, unknown>;
            expect(stats).toBeTruthy();
            expect(stats["test-stack-web-1"]).toBeTruthy();

            await client.sendAndReceive("unsubscribeStats");
        });
    });

    test("containerTop — subscribe, receive push, unsubscribe", async () => {
        await withAuthClient(async (client) => {
            const subResp = await client.sendAndReceive("subscribeTop", "test-stack-web-1");
            expect(subResp.ok).toBe(true);

            const pushed = await client.waitForEvent("containerTop");
            expect(pushed.ok).toBe(true);
            const processes = pushed.processes as unknown[];
            expect(Array.isArray(processes)).toBe(true);
            expect(processes.length).toBeGreaterThan(0);

            await client.sendAndReceive("unsubscribeTop");
        });
    });

    test("serviceStatusList", async () => {
        await withAuthClient(async (client) => {
            const resp = await client.sendAndReceive("serviceStatusList", "test-stack");
            expect(resp.ok).toBe(true);
            expect(resp.serviceStatusList).toBeTruthy();
        });
    });

    test("getDockerNetworkList", async () => {
        await withAuthClient(async (client) => {
            const resp = await client.sendAndReceive("getDockerNetworkList");
            expect(resp.ok).toBe(true);
            expect(Array.isArray(resp.dockerNetworkList)).toBe(true);
        });
    });

    test("getDockerImageList", async () => {
        await withAuthClient(async (client) => {
            const resp = await client.sendAndReceive("getDockerImageList");
            expect(resp.ok).toBe(true);
            const images = resp.dockerImageList as unknown[];
            expect(Array.isArray(images)).toBe(true);
            expect(images.length).toBeGreaterThan(0);
        });
    });

    test("imageInspect — inspect nginx:latest", async () => {
        await withAuthClient(async (client) => {
            const resp = await client.sendAndReceive("imageInspect", "nginx:latest");
            expect(resp.ok).toBe(true);
            expect(resp.imageDetail).toBeTruthy();
        });
    });

    test("getDockerVolumeList", async () => {
        await withAuthClient(async (client) => {
            const resp = await client.sendAndReceive("getDockerVolumeList");
            expect(resp.ok).toBe(true);
            expect(Array.isArray(resp.dockerVolumeList)).toBe(true);
        });
    });

    test("volumeInspect — inspect first volume", async () => {
        await withAuthClient(async (client) => {
            const listResp = await client.sendAndReceive("getDockerVolumeList");
            const volumes = listResp.dockerVolumeList as Record<string, unknown>[];
            if (!volumes || volumes.length === 0) {
                // Skip if no volumes in mock daemon
                return;
            }

            const volName = volumes[0].name as string;
            expect(volName).toBeTruthy();

            const resp = await client.sendAndReceive("volumeInspect", volName);
            expect(resp.ok).toBe(true);
            expect(resp.volumeDetail).toBeTruthy();
        });
    });

    test("networkInspect — inspect first network", async () => {
        await withAuthClient(async (client) => {
            const listResp = await client.sendAndReceive("getDockerNetworkList");
            const networks = listResp.dockerNetworkList as Record<string, unknown>[];
            expect(networks.length).toBeGreaterThan(0);

            const netName = networks[0].name as string;
            expect(netName).toBeTruthy();

            const resp = await client.sendAndReceive("networkInspect", netName);
            expect(resp.ok).toBe(true);
            expect(resp.networkDetail).toBeTruthy();
        });
    });

    test("containerInspect", async () => {
        await withAuthClient(async (client) => {
            const resp = await client.sendAndReceive("containerInspect", "test-stack-web-1");
            expect(resp.ok).toBe(true);
            // Verify inspectData is a JSON object, not a double-encoded string (bug fix in 6629346)
            expect(resp.inspectData).toBeTruthy();
            expect(typeof resp.inspectData).not.toBe("string");
            expect(typeof resp.inspectData).toBe("object");
        });
    });
});
