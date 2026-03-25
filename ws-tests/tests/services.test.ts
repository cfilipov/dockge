import { describe, test, expect, beforeAll } from "vitest";
import { resetMockState, connectClient, waitForContainerState } from "../src/helpers.js";

describe("services", () => {
    beforeAll(async () => {
        await resetMockState();
    });

    // Ordered: stop → start → restart → recreate → update → read-only → container ops → errors

    test("stopService — container reaches exited state", async () => {
        const cmd = await connectClient();
        const obs = await connectClient();
        try {
            await cmd.login();
            await obs.login();
            await obs.waitForEvent("containers"); // drain AfterLogin

            const { ack } = await cmd.sendAction("stopService", "test-stack", "web");
            expect(ack.ok).toBe(true);

            await waitForContainerState(obs, "test-stack-web-1", "exited");
        } finally {
            cmd.close();
            obs.close();
        }
    });

    test("startService — container reaches running state", async () => {
        // web is stopped from previous test
        const cmd = await connectClient();
        const obs = await connectClient();
        try {
            await cmd.login();
            await obs.login();
            await obs.waitForEvent("containers"); // drain AfterLogin

            const { ack } = await cmd.sendAction("startService", "test-stack", "web");
            expect(ack.ok).toBe(true);

            await waitForContainerState(obs, "test-stack-web-1", "running");
        } finally {
            cmd.close();
            obs.close();
        }
    });

    test("restartService — container ends up running", async () => {
        const cmd = await connectClient();
        const obs = await connectClient();
        try {
            await cmd.login();
            await obs.login();
            await obs.waitForEvent("containers"); // drain AfterLogin

            const { ack } = await cmd.sendAction("restartService", "test-stack", "web");
            expect(ack.ok).toBe(true);

            await waitForContainerState(obs, "test-stack-web-1", "running");
        } finally {
            cmd.close();
            obs.close();
        }
    });

    test("recreateService — container ends up running", async () => {
        const cmd = await connectClient();
        const obs = await connectClient();
        try {
            await cmd.login();
            await obs.login();
            await obs.waitForEvent("containers"); // drain AfterLogin

            const { ack } = await cmd.sendAction("recreateService", "test-stack", "web");
            expect(ack.ok).toBe(true);

            await waitForContainerState(obs, "test-stack-web-1", "running");
        } finally {
            cmd.close();
            obs.close();
        }
    });

    test("updateService — container ends up running after pull+recreate", async () => {
        const cmd = await connectClient();
        const obs = await connectClient();
        try {
            await cmd.login();
            await obs.login();
            await obs.waitForEvent("containers"); // drain AfterLogin

            const { ack } = await cmd.sendAction("updateService", "test-stack", "web");
            expect(ack.ok).toBe(true);

            await waitForContainerState(obs, "test-stack-web-1", "running");
        } finally {
            cmd.close();
            obs.close();
        }
    });

    test("checkImageUpdates", async () => {
        const cmd = await connectClient();
        try {
            await cmd.login();
            const resp = await cmd.sendAndReceive("checkImageUpdates", "test-stack");
            expect(resp.ok).toBe(true);
        } finally {
            cmd.close();
        }
    });

    // Container ops: stop → start → restart
    test("stopContainer — container reaches exited state", async () => {
        const cmd = await connectClient();
        const obs = await connectClient();
        try {
            await cmd.login();
            await obs.login();
            await obs.waitForEvent("containers"); // drain AfterLogin

            const { ack } = await cmd.sendAction("stopContainer", "test-stack-web-1");
            expect(ack.ok).toBe(true);

            await waitForContainerState(obs, "test-stack-web-1", "exited");
        } finally {
            cmd.close();
            obs.close();
        }
    });

    test("startContainer — container reaches running state", async () => {
        // web-1 is stopped from previous test
        const cmd = await connectClient();
        const obs = await connectClient();
        try {
            await cmd.login();
            await obs.login();
            await obs.waitForEvent("containers"); // drain AfterLogin

            const { ack } = await cmd.sendAction("startContainer", "test-stack-web-1");
            expect(ack.ok).toBe(true);

            await waitForContainerState(obs, "test-stack-web-1", "running");
        } finally {
            cmd.close();
            obs.close();
        }
    });

    test("restartContainer — container ends up running", async () => {
        const cmd = await connectClient();
        const obs = await connectClient();
        try {
            await cmd.login();
            await obs.login();
            await obs.waitForEvent("containers"); // drain AfterLogin

            const { ack } = await cmd.sendAction("restartContainer", "test-stack-web-1");
            expect(ack.ok).toBe(true);

            await waitForContainerState(obs, "test-stack-web-1", "running");
        } finally {
            cmd.close();
            obs.close();
        }
    });

    test("serviceMissingArgs — empty service/stack name fails", async () => {
        const cmd = await connectClient();
        try {
            await cmd.login();

            // Missing service name
            const { ack: resp1 } = await cmd.sendAction("startService", "test-stack", "");
            expect(resp1.ok).toBe(false);

            // Missing stack name
            const { ack: resp2 } = await cmd.sendAction("stopService", "", "web");
            expect(resp2.ok).toBe(false);
        } finally {
            cmd.close();
        }
    });

    test("stopService nonexistent service — acks but no crash", async () => {
        const cmd = await connectClient();
        const obs = await connectClient();
        try {
            await cmd.login();
            await obs.login();
            await obs.waitForEvent("containers"); // drain AfterLogin

            const { ack } = await cmd.sendAction("stopService", "test-stack", "no-such-service");
            expect(ack.ok).toBe(true);

            // Background goroutine fails silently; no containers broadcast for nonexistent service
            const evt = await obs.tryWaitForEvent("containers", 500);
            expect(evt).toBeNull();
        } finally {
            cmd.close();
            obs.close();
        }
    });
});
