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

            const { ack } = await cmd.sendAction("stopStack", "test-stack");
            expect(ack.ok).toBe(true);

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

            const { ack } = await cmd.sendAction("startStack", "test-stack");
            expect(ack.ok).toBe(true);

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

            const { ack: pauseResp } = await cmd.sendAction("pauseStack", "test-stack");
            expect(pauseResp.ok).toBe(true);

            await waitForContainerState(obs, "test-stack-web-1", "paused");

            const { ack: resumeResp } = await cmd.sendAction("resumeStack", "test-stack");
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

            const { ack } = await cmd.sendAction("restartStack", "test-stack");
            expect(ack.ok).toBe(true);

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

            const { ack } = await cmd.sendAction("updateStack", "test-stack");
            expect(ack.ok).toBe(true);

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

            const { ack } = await cmd.sendAction("downStack", "test-stack");
            expect(ack.ok).toBe(true);

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

            const { ack } = await cmd.sendAction("deleteStack", "to-delete", { deleteStackFiles: true });
            expect(ack.ok).toBe(true);
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

            const { ack } = await cmd.sendAction("forceDeleteStack", "force-delete-me");
            expect(ack.ok).toBe(true);
        } finally {
            cmd.close();
        }
    });

    test("startStack with empty name — fails", async () => {
        const cmd = await connectClient();
        try {
            await cmd.login();
            const { ack } = await cmd.sendAction("startStack", "");
            expect(ack.ok).toBe(false);
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

            const { ack } = await cmd.sendAction("stopStack", "no-such-stack");
            expect(ack.ok).toBe(true);

            // No containers broadcast should arrive for a nonexistent stack
            const evt = await obs.tryWaitForEvent("containers", 500);
            expect(evt).toBeNull();
        } finally {
            cmd.close();
            obs.close();
        }
    });
});
