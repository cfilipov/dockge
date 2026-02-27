import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";

test.describe("Change Password — Validation", () => {
    test.beforeEach(async ({ page }) => {
        await page.goto("/settings/security");
        await waitForApp(page);
        await expect(page.getByText("Change Password")).toBeVisible({ timeout: 10000 });
    });

    test("shows validation error when passwords do not match", async ({ page }) => {
        // Fill in password fields with mismatched passwords
        await page.locator("#current-password").first().fill("testpass123");
        await page.locator("#new-password").fill("newpass456");
        await page.locator("#repeat-new-password").fill("differentpass789");

        // Click "Update Password"
        await page.getByRole("button", { name: "Update Password" }).click();

        // Verify validation error is displayed
        await expect(page.getByText("The repeat password does not match.")).toBeVisible({ timeout: 5000 });

        // Verify the repeat password field has the invalid class
        await expect(page.locator("#repeat-new-password")).toHaveClass(/is-invalid/);
    });

    test("shows error with wrong current password", async ({ page }) => {
        // Fill in password fields with wrong current password
        await page.locator("#current-password").first().fill("wrongpassword");
        await page.locator("#new-password").fill("newpass456");
        await page.locator("#repeat-new-password").fill("newpass456");

        // Click "Update Password"
        await page.getByRole("button", { name: "Update Password" }).click();

        // Verify error toast or feedback appears (password doesn't match)
        // The server will return an error which is shown via toastRes
        await expect(page.getByText(/incorrect|wrong|invalid/i)).toBeVisible({ timeout: 5000 });
    });

    test("disable auth button shows confirmation dialog", async ({ page }) => {
        // Click "Disable Auth" button
        await page.locator("#disableAuth-btn").click();

        // Verify confirmation dialog appears with password field
        await expect(page.getByText("Please use this option carefully!")).toBeVisible({ timeout: 5000 });
        await expect(page.locator("#current-password2")).toBeVisible();
    });
});
