import { describe, test, expect, beforeAll } from "vitest";
import { resetMockState, withAuthClient } from "../src/helpers.js";

describe("stack-lifecycle", () => {
    beforeAll(async () => {
        await resetMockState();
    });

    test("startStack", async () => {
        await withAuthClient(async (client) => {
            const resp = await client.sendAndReceive("startStack", "test-stack");
            expect(resp.ok).toBe(true);
        });
    });

    test("stopStack", async () => {
        await withAuthClient(async (client) => {
            const resp = await client.sendAndReceive("stopStack", "test-stack");
            expect(resp.ok).toBe(true);
        });
    });

    test("restartStack", async () => {
        await withAuthClient(async (client) => {
            const resp = await client.sendAndReceive("restartStack", "test-stack");
            expect(resp.ok).toBe(true);
        });
    });

    test("downStack", async () => {
        await withAuthClient(async (client) => {
            const resp = await client.sendAndReceive("downStack", "test-stack");
            expect(resp.ok).toBe(true);
        });
    });

    test("pauseAndResumeStack", async () => {
        await withAuthClient(async (client) => {
            const pauseResp = await client.sendAndReceive("pauseStack", "test-stack");
            expect(pauseResp.ok).toBe(true);

            const resumeResp = await client.sendAndReceive("resumeStack", "test-stack");
            expect(resumeResp.ok).toBe(true);
        });
    });

    test("updateStack", async () => {
        await withAuthClient(async (client) => {
            const resp = await client.sendAndReceive("updateStack", "test-stack");
            expect(resp.ok).toBe(true);
        });
    });

    test("deleteStackWithFiles (protocol-only)", async () => {
        await withAuthClient(async (client) => {
            // Save a stack first so it exists
            const yaml = "services:\n  app:\n    image: alpine\n";
            await client.sendAndReceive("saveStack", "to-delete", yaml, "", "", false);

            const resp = await client.sendAndReceive("deleteStack", "to-delete", { deleteStackFiles: true });
            expect(resp.ok).toBe(true);
        });
    });

    test("forceDeleteStack (protocol-only)", async () => {
        await withAuthClient(async (client) => {
            // Save a stack first
            const yaml = "services:\n  app:\n    image: alpine\n";
            await client.sendAndReceive("saveStack", "force-delete-me", yaml, "", "", false);

            const resp = await client.sendAndReceive("forceDeleteStack", "force-delete-me");
            expect(resp.ok).toBe(true);
        });
    });

    test("startStackMissingName — fails with empty name", async () => {
        await withAuthClient(async (client) => {
            const resp = await client.sendAndReceive("startStack", "");
            expect(resp.ok).toBe(false);
        });
    });
});
