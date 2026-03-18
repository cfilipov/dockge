import { describe, test, expect } from "vitest";
import { resetMockState, connectClient, waitForContainerState } from "../src/helpers.js";

describe("services", () => {
    test("stopService — container reaches exited state", async () => {
        await resetMockState();

        const cmd = await connectClient();
        const obs = await connectClient();
        try {
            await cmd.login();
            await obs.login();
            await obs.waitForEvent("containers"); // drain AfterLogin

            const resp = await cmd.sendAndReceive("stopService", "test-stack", "web");
            expect(resp.ok).toBe(true);

            await waitForContainerState(obs, "test-stack-web-1", "exited");
        } finally {
            cmd.close();
            obs.close();
        }
    });

    test("startService — container reaches running state", async () => {
        await resetMockState();

        const cmd = await connectClient();
        const obs = await connectClient();
        try {
            await cmd.login();
            await obs.login();
            await obs.waitForEvent("containers"); // drain AfterLogin

            // Stop first so we can test start
            await cmd.sendAndReceive("stopService", "test-stack", "web");
            await waitForContainerState(obs, "test-stack-web-1", "exited");

            const resp = await cmd.sendAndReceive("startService", "test-stack", "web");
            expect(resp.ok).toBe(true);

            await waitForContainerState(obs, "test-stack-web-1", "running");
        } finally {
            cmd.close();
            obs.close();
        }
    });

    test("restartService — container ends up running", async () => {
        await resetMockState();

        const cmd = await connectClient();
        const obs = await connectClient();
        try {
            await cmd.login();
            await obs.login();
            await obs.waitForEvent("containers"); // drain AfterLogin

            const resp = await cmd.sendAndReceive("restartService", "test-stack", "web");
            expect(resp.ok).toBe(true);

            await waitForContainerState(obs, "test-stack-web-1", "running");
        } finally {
            cmd.close();
            obs.close();
        }
    });

    test("recreateService — container ends up running", async () => {
        await resetMockState();

        const cmd = await connectClient();
        const obs = await connectClient();
        try {
            await cmd.login();
            await obs.login();
            await obs.waitForEvent("containers"); // drain AfterLogin

            const resp = await cmd.sendAndReceive("recreateService", "test-stack", "web");
            expect(resp.ok).toBe(true);

            await waitForContainerState(obs, "test-stack-web-1", "running");
        } finally {
            cmd.close();
            obs.close();
        }
    });

    test("updateService — container ends up running after pull+recreate", async () => {
        await resetMockState();

        const cmd = await connectClient();
        const obs = await connectClient();
        try {
            await cmd.login();
            await obs.login();
            await obs.waitForEvent("containers"); // drain AfterLogin

            const resp = await cmd.sendAndReceive("updateService", "test-stack", "web");
            expect(resp.ok).toBe(true);

            await waitForContainerState(obs, "test-stack-web-1", "running");
        } finally {
            cmd.close();
            obs.close();
        }
    });

    test("checkImageUpdates", async () => {
        await resetMockState();

        const cmd = await connectClient();
        try {
            await cmd.login();
            const resp = await cmd.sendAndReceive("checkImageUpdates", "test-stack");
            expect(resp.ok).toBe(true);
        } finally {
            cmd.close();
        }
    });

    test("stopContainer — container reaches exited state", async () => {
        await resetMockState();

        const cmd = await connectClient();
        const obs = await connectClient();
        try {
            await cmd.login();
            await obs.login();
            await obs.waitForEvent("containers"); // drain AfterLogin

            const resp = await cmd.sendAndReceive("stopContainer", "test-stack-web-1");
            expect(resp.ok).toBe(true);

            await waitForContainerState(obs, "test-stack-web-1", "exited");
        } finally {
            cmd.close();
            obs.close();
        }
    });

    test("restartContainer — container ends up running", async () => {
        await resetMockState();

        const cmd = await connectClient();
        const obs = await connectClient();
        try {
            await cmd.login();
            await obs.login();
            await obs.waitForEvent("containers"); // drain AfterLogin

            const resp = await cmd.sendAndReceive("restartContainer", "test-stack-web-1");
            expect(resp.ok).toBe(true);

            await waitForContainerState(obs, "test-stack-web-1", "running");
        } finally {
            cmd.close();
            obs.close();
        }
    });

    test("startContainer — container reaches running state", async () => {
        await resetMockState();

        const cmd = await connectClient();
        const obs = await connectClient();
        try {
            await cmd.login();
            await obs.login();
            await obs.waitForEvent("containers"); // drain AfterLogin

            // Stop first
            await cmd.sendAndReceive("stopContainer", "test-stack-web-1");
            await waitForContainerState(obs, "test-stack-web-1", "exited");

            const resp = await cmd.sendAndReceive("startContainer", "test-stack-web-1");
            expect(resp.ok).toBe(true);

            await waitForContainerState(obs, "test-stack-web-1", "running");
        } finally {
            cmd.close();
            obs.close();
        }
    });

    test("serviceMissingArgs — empty service/stack name fails", async () => {
        await resetMockState();

        const cmd = await connectClient();
        try {
            await cmd.login();

            // Missing service name
            const resp1 = await cmd.sendAndReceive("startService", "test-stack", "");
            expect(resp1.ok).toBe(false);

            // Missing stack name
            const resp2 = await cmd.sendAndReceive("stopService", "", "web");
            expect(resp2.ok).toBe(false);
        } finally {
            cmd.close();
        }
    });

    test("stopService nonexistent service — acks but no crash", async () => {
        await resetMockState();

        const cmd = await connectClient();
        const obs = await connectClient();
        try {
            await cmd.login();
            await obs.login();
            await obs.waitForEvent("containers"); // drain AfterLogin

            const resp = await cmd.sendAndReceive("stopService", "test-stack", "no-such-service");
            expect(resp.ok).toBe(true);

            // Background goroutine fails silently; no containers broadcast for nonexistent service
            const evt = await obs.tryWaitForEvent("containers", 2000);
            expect(evt).toBeNull();
        } finally {
            cmd.close();
            obs.close();
        }
    });
});
