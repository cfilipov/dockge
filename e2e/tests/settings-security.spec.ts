import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";

test.describe("Settings — Security", () => {
    test.beforeEach(async ({ page }) => {
        await page.goto("/settings/security");
        await waitForApp(page);
    });

    test("displays Security content header", async ({ page }) => {
        await expect(page.locator(".settings-content-header")).toContainText("Security");
    });

    test("shows current user display and Logout button", async ({ page }) => {
        await expect(page.getByText(/Current User/i)).toBeVisible();
        await expect(page.locator("#logout-btn")).toBeVisible();
    });

    test("shows Change Password form", async ({ page }) => {
        await expect(page.getByText("Change Password")).toBeVisible();
        await expect(page.locator("#current-password").first()).toBeVisible();
        await expect(page.locator("#new-password")).toBeVisible();
        await expect(page.locator("#repeat-new-password")).toBeVisible();
        await expect(page.getByRole("button", { name: "Update Password" })).toBeVisible();
    });

    test("shows Advanced section with Disable Auth button", async ({ page }) => {
        await expect(page.getByText("Advanced")).toBeVisible();
        await expect(page.locator("#disableAuth-btn")).toBeVisible();
    });

    test("screenshot: settings security", async ({ page }) => {
        await expect(page.locator("#disableAuth-btn")).toBeVisible();
        await expect(page).toHaveScreenshot("settings-security.png");
    });
});
