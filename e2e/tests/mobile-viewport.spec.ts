import { test, expect } from "../fixtures/auth.fixture";

// Mobile viewport: 375x667 (iPhone SE)
test.use({ viewport: { width: 375, height: 667 } });

// On mobile, the header nav is hidden, so we can't use waitForApp().
// Instead wait for the main content to load via specific page elements.

test.describe("Mobile Viewport", () => {
    test("dashboard hides sidebar and header nav on mobile", async ({ page }) => {
        await page.goto("/");
        // Wait for the main content area to load
        await expect(page.locator(".app-layout")).toBeVisible({ timeout: 15000 });

        // The app-layout should have the "mobile" class
        await expect(page.locator(".app-layout")).toHaveClass(/mobile/);

        // The stack list sidebar search should NOT be visible
        await expect(page.getByPlaceholder("Search...")).not.toBeVisible();

        // Desktop header nav links should be hidden
        await expect(page.getByRole("link", { name: "Containers" })).not.toBeVisible();
        await expect(page.getByRole("link", { name: "Networks" })).not.toBeVisible();
        await expect(page.getByRole("link", { name: "Images" })).not.toBeVisible();
    });

    test("compose page works on mobile", async ({ page }) => {
        await page.goto("/stacks/01-web-app");
        // Wait for the stack heading
        await expect(page.getByRole("heading", { name: /01-web-app/, level: 1 })).toBeVisible({ timeout: 15000 });

        // Action buttons should still be visible
        await expect(page.getByRole("button", { name: "Edit" })).toBeVisible();
    });

    test("settings page accessible on mobile", async ({ page }) => {
        await page.goto("/settings/appearance");
        // Wait for settings content to load
        await expect(page.getByText("Theme")).toBeVisible({ timeout: 15000 });
    });
});
