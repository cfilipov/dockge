import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";

test.describe("Console Page", () => {
    test.beforeEach(async ({ page }) => {
        await page.goto("/console");
        await waitForApp(page);
    });

    test("displays Console heading", async ({ page }) => {
        await expect(page.getByRole("heading", { name: "Console" })).toBeVisible({ timeout: 10000 });
    });

    test("shows terminal or disabled alert", async ({ page }) => {
        // Console may be enabled (terminal visible) or disabled (warning alert)
        const terminal = page.locator(".shadow-box.terminal");
        const alert = page.locator(".alert-warning");
        await expect(terminal.or(alert)).toBeVisible({ timeout: 10000 });
    });

    test("screenshot: console page", async ({ page }) => {
        // Wait for content to settle
        const terminal = page.locator(".shadow-box.terminal");
        const alert = page.locator(".alert-warning");
        await expect(terminal.or(alert)).toBeVisible({ timeout: 10000 });
        await expect(page).toHaveScreenshot("console.png");
    });
});
