import { describe, test, expect, beforeAll } from "vitest";
import { resetMockState, connectClient, waitForContainerState } from "../src/helpers.js";

describe("stack-lifecycle", () => {
    beforeAll(async () => {
        await resetMockState();
    });

    // Ordered: stop → start → pause/resume → restart → update → down → self-contained → error cases

    test("stopStack — containers reach exited state", async () => {
        const cmd = await connectClient();
        const obs = await connectClient();
        try {
            await cmd.login();
            await obs.login();
            await obs.waitForEvent("containers"); // drain AfterLogin

            const resp = await cmd.sendAndReceive("stopStack", "test-stack");
            expect(resp.ok).toBe(true);

            await waitForContainerState(obs, "test-stack-web-1", "exited");
        } finally {
            cmd.close();
            obs.close();
        }
    });

    test("startStack — containers reach running state", async () => {
        const cmd = await connectClient();
        const obs = await connectClient();
        try {
            await cmd.login();
            await obs.login();
            await obs.waitForEvent("containers"); // drain AfterLogin

            const resp = await cmd.sendAndReceive("startStack", "test-stack");
            expect(resp.ok).toBe(true);

            await waitForContainerState(obs, "test-stack-web-1", "running");
        } finally {
            cmd.close();
            obs.close();
        }
    });

    test("pauseAndResumeStack — containers transition through paused", async () => {
        const cmd = await connectClient();
        const obs = await connectClient();
        try {
            await cmd.login();
            await obs.login();
            await obs.waitForEvent("containers"); // drain AfterLogin

            const pauseResp = await cmd.sendAndReceive("pauseStack", "test-stack");
            expect(pauseResp.ok).toBe(true);

            await waitForContainerState(obs, "test-stack-web-1", "paused");

            const resumeResp = await cmd.sendAndReceive("resumeStack", "test-stack");
            expect(resumeResp.ok).toBe(true);

            await waitForContainerState(obs, "test-stack-web-1", "running");
        } finally {
            cmd.close();
            obs.close();
        }
    });

    test("restartStack — containers end up running", async () => {
        const cmd = await connectClient();
        const obs = await connectClient();
        try {
            await cmd.login();
            await obs.login();
            await obs.waitForEvent("containers"); // drain AfterLogin

            const resp = await cmd.sendAndReceive("restartStack", "test-stack");
            expect(resp.ok).toBe(true);

            // Restart goes stop→start; final state should be running
            await waitForContainerState(obs, "test-stack-web-1", "running");
        } finally {
            cmd.close();
            obs.close();
        }
    });

    test("updateStack — containers end up running after pull+deploy", async () => {
        const cmd = await connectClient();
        const obs = await connectClient();
        try {
            await cmd.login();
            await obs.login();
            await obs.waitForEvent("containers"); // drain AfterLogin

            const resp = await cmd.sendAndReceive("updateStack", "test-stack");
            expect(resp.ok).toBe(true);

            await waitForContainerState(obs, "test-stack-web-1", "running");
        } finally {
            cmd.close();
            obs.close();
        }
    });

    test("downStack — containers are destroyed (null in broadcast)", async () => {
        const cmd = await connectClient();
        const obs = await connectClient();
        try {
            await cmd.login();
            await obs.login();
            await obs.waitForEvent("containers"); // drain AfterLogin

            const resp = await cmd.sendAndReceive("downStack", "test-stack");
            expect(resp.ok).toBe(true);

            await waitForContainerState(obs, "test-stack-web-1", null);
        } finally {
            cmd.close();
            obs.close();
        }
    });

    test("deleteStackWithFiles (protocol-only)", async () => {
        const cmd = await connectClient();
        try {
            await cmd.login();

            const yaml = "services:\n  app:\n    image: alpine\n";
            await cmd.sendAndReceive("saveStack", "to-delete", yaml, "", "", false);

            const resp = await cmd.sendAndReceive("deleteStack", "to-delete", { deleteStackFiles: true });
            expect(resp.ok).toBe(true);
        } finally {
            cmd.close();
        }
    });

    test("forceDeleteStack (protocol-only)", async () => {
        const cmd = await connectClient();
        try {
            await cmd.login();

            const yaml = "services:\n  app:\n    image: alpine\n";
            await cmd.sendAndReceive("saveStack", "force-delete-me", yaml, "", "", false);

            const resp = await cmd.sendAndReceive("forceDeleteStack", "force-delete-me");
            expect(resp.ok).toBe(true);
        } finally {
            cmd.close();
        }
    });

    test("startStack with empty name — fails", async () => {
        const cmd = await connectClient();
        try {
            await cmd.login();
            const resp = await cmd.sendAndReceive("startStack", "");
            expect(resp.ok).toBe(false);
        } finally {
            cmd.close();
        }
    });

    test("stopStack nonexistent stack — acks but no state change broadcast", async () => {
        const cmd = await connectClient();
        const obs = await connectClient();
        try {
            await cmd.login();
            await obs.login();
            await obs.waitForEvent("containers"); // drain AfterLogin

            const resp = await cmd.sendAndReceive("stopStack", "no-such-stack");
            expect(resp.ok).toBe(true);

            // No containers broadcast should arrive for a nonexistent stack
            const evt = await obs.tryWaitForEvent("containers", 500);
            expect(evt).toBeNull();
        } finally {
            cmd.close();
            obs.close();
        }
    });
});
