import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";

test.describe("Container Terminal", () => {
    test.beforeEach(async ({ page }) => {
        await page.goto("/terminal/01-web-app/nginx/bash");
        await waitForApp(page);
    });

    test("displays terminal heading with service and stack name", async ({ page }) => {
        await expect(page.getByRole("heading", { name: /Terminal.*nginx.*01-web-app/i })).toBeVisible({ timeout: 10000 });
    });

    test("shows switch shell link", async ({ page }) => {
        await expect(page.getByRole("link", { name: /Switch to sh/i })).toBeVisible();
    });

    test("terminal element is visible", async ({ page }) => {
        await expect(page.locator(".shadow-box.terminal")).toBeVisible({ timeout: 10000 });
    });

    test("screenshot: container terminal", async ({ page }) => {
        await expect(page.locator(".shadow-box.terminal")).toBeVisible({ timeout: 10000 });
        await expect(page).toHaveScreenshot("container-terminal.png");
    });
});
