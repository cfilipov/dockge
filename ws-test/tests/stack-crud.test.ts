import { describe, test, expect, beforeAll } from "vitest";
import { resetMockState, withAuthClient } from "../src/helpers.js";

describe("stack-crud", () => {
    beforeAll(async () => {
        await resetMockState();
    });

    test("getStack — returns stack data with composeYAML", async () => {
        await withAuthClient(async (client) => {
            const resp = await client.sendAndReceive("getStack", "test-stack");
            expect(resp.ok).toBe(true);

            const stack = resp.stack as Record<string, unknown>;
            expect(stack).toBeTruthy();
            expect(stack.name).toBe("test-stack");
            expect(stack.composeYAML).toBeTruthy();
            expect(typeof stack.composeYAML).toBe("string");
        });
    });

    test("saveStack — saves new stack (protocol-only)", async () => {
        await withAuthClient(async (client) => {
            const yaml = "services:\n  app:\n    image: alpine:3.19\n";
            const resp = await client.sendAndReceive("saveStack", "new-stack", yaml, "", "", false);
            expect(resp.ok).toBe(true);
        });
    });

    test("saveStackWithOverrideAndEnv — saves with env and override (protocol-only)", async () => {
        await withAuthClient(async (client) => {
            const yaml = "services:\n  app:\n    image: nginx:latest\n";
            const envContent = "DB_HOST=localhost\nDB_PORT=5432";
            const overrideYAML = "services:\n  app:\n    ports:\n      - 8080:80\n";
            const resp = await client.sendAndReceive("saveStack", "full-stack", yaml, envContent, overrideYAML, false);
            expect(resp.ok).toBe(true);
        });
    });

    test("deployStack — deploys a stack (protocol-only)", async () => {
        await withAuthClient(async (client) => {
            const yaml = "services:\n  app:\n    image: alpine:3.19\n";
            const resp = await client.sendAndReceive("deployStack", "deploy-test", yaml, "", "", false);
            expect(resp.ok).toBe(true);
        });
    });

    test("deployStackEmptyYAML — fails with empty YAML", async () => {
        await withAuthClient(async (client) => {
            const resp = await client.sendAndReceive("deployStack", "bad-stack", "", "", "", false);
            expect(resp.ok).toBe(false);
        });
    });

    test("getStackNonexistent — returns ok with stack name", async () => {
        await withAuthClient(async (client) => {
            const resp = await client.sendAndReceive("getStack", "nonexistent-stack");
            expect(resp.ok).toBe(true);
            const stack = resp.stack as Record<string, unknown>;
            expect(stack).toBeTruthy();
            expect(stack.name).toBe("nonexistent-stack");
        });
    });

    describe("saveStack invalid names", () => {
        const yaml = "services:\n  app:\n    image: alpine:3.19\n";
        const cases = [
            { label: "path traversal", name: "../traversal" },
            { label: "shell injection", name: "; rm -rf /" },
            { label: "uppercase", name: "UPPERCASE" },
            { label: "dot prefix", name: ".hidden" },
            { label: "space", name: "has space" },
            { label: "leading hyphen", name: "-leading" },
        ];

        for (const { label, name } of cases) {
            test(`rejects ${label}: "${name}"`, async () => {
                await withAuthClient(async (client) => {
                    const resp = await client.sendAndReceive("saveStack", name, yaml, "", "", false);
                    expect(resp.ok).toBe(false);
                });
            });
        }
    });

    describe("deleteStack invalid names", () => {
        const cases = [
            { label: "path traversal", name: "../traversal" },
            { label: "shell injection", name: "; rm -rf /" },
            { label: "null byte", name: "stack\x00evil" },
        ];

        for (const { label, name } of cases) {
            test(`rejects ${label}: "${name}"`, async () => {
                await withAuthClient(async (client) => {
                    const resp = await client.sendAndReceive("deleteStack", name, { deleteStackFiles: true });
                    expect(resp.ok).toBe(false);
                });
            });
        }
    });

    test("getStackInvalidName — rejects path traversal", async () => {
        await withAuthClient(async (client) => {
            const resp = await client.sendAndReceive("getStack", "../etc/passwd");
            expect(resp.ok).toBe(false);
        });
    });
});
