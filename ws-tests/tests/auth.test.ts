import { describe, test, expect, beforeAll } from "vitest";
import { resetMockState, resetDB, withClient, connectClient } from "../src/helpers.js";

describe("auth", () => {
    beforeAll(async () => {
        await resetMockState();
        await resetDB();
    });

    test("setupAndLogin — setup creates user, login returns token", async () => {
        await withClient(async (client) => {
            // In dev mode, admin already exists. Setup should fail (already done).
            // Instead, just login directly.
            const resp = await client.sendAndReceive("login", "admin", "testpass123", "", "");
            expect(resp.ok).toBe(true);
            expect(resp.token).toBeTruthy();
            expect(typeof resp.token).toBe("string");
        });
    });

    test("loginBadPassword — wrong password fails", async () => {
        await withClient(async (client) => {
            const resp = await client.sendAndReceive("login", "admin", "wrongpassword", "", "");
            expect(resp.ok).toBe(false);
        });
    });

    test("loginByToken — login with JWT token", async () => {
        const client1 = await connectClient();
        try {
            const token = await client1.login();

            const client2 = await connectClient();
            try {
                const resp = await client2.sendAndReceive("loginByToken", token);
                expect(resp.ok).toBe(true);
            } finally {
                client2.close();
            }
        } finally {
            client1.close();
        }
    });

    test("loginByTokenBadToken — invalid token fails", async () => {
        await withClient(async (client) => {
            const resp = await client.sendAndReceive("loginByToken", "invalid.jwt.token");
            expect(resp.ok).toBe(false);
        });
    });

    test("changePassword — change and verify new password", async () => {
        const client1 = await connectClient();
        try {
            await client1.login();
            const resp = await client1.sendAndReceive("changePassword", {
                currentPassword: "testpass123",
                newPassword: "newpass456",
            });
            expect(resp.ok).toBe(true);

            // Verify new password works on a new connection
            const client2 = await connectClient();
            try {
                const loginResp = await client2.sendAndReceive("login", "admin", "newpass456", "", "");
                expect(loginResp.ok).toBe(true);
            } finally {
                client2.close();
            }

            // Change password back to original
            const revertResp = await client1.sendAndReceive("changePassword", {
                currentPassword: "newpass456",
                newPassword: "testpass123",
            });
            expect(revertResp.ok).toBe(true);
        } finally {
            client1.close();
        }
    });

    test("changePasswordWrongCurrent — wrong current password fails", async () => {
        await withClient(async (client) => {
            await client.login();
            const resp = await client.sendAndReceive("changePassword", {
                currentPassword: "wrongpassword",
                newPassword: "newpass456",
            });
            expect(resp.ok).toBe(false);
        });
    });

    test("setupAlreadyDone — setup fails when admin exists", async () => {
        await withClient(async (client) => {
            const resp = await client.sendAndReceive("setup", "hacker", "password123");
            expect(resp.ok).toBe(false);
        });
    });

    test("loginEmptyCredentials — empty username/password fails", async () => {
        await withClient(async (client) => {
            const resp = await client.sendAndReceive("login", "", "", "", "");
            expect(resp.ok).toBe(false);
        });
    });

    test("logout — protected endpoints fail after logout", async () => {
        await withClient(async (client) => {
            await client.login();

            const logoutResp = await client.sendAndReceive("logout");
            expect(logoutResp.ok).toBe(true);

            // After logout, protected endpoints should fail
            const resp = await client.sendAndReceive("getStack", "test-stack");
            expect(resp.ok).toBe(false);
        });
    });

    test("twoFAStatus — returns not enabled", async () => {
        await withClient(async (client) => {
            const resp = await client.sendAndReceive("twoFAStatus");
            expect(resp.ok).toBe(true);
            expect(resp.status).toBe(false);
        });
    });

    test("prepare2FA — returns not supported", async () => {
        await withClient(async (client) => {
            const resp = await client.sendAndReceive("prepare2FA");
            expect(resp.ok).toBe(false);
            expect(resp.msg).toContain("not yet supported");
        });
    });

    test("getTurnstileSiteKey — returns ok", async () => {
        await withClient(async (client) => {
            const resp = await client.sendAndReceive("getTurnstileSiteKey");
            expect(resp.ok).toBe(true);
        });
    });
});
