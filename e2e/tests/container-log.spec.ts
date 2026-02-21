import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";

test.describe("Container Log", () => {
    test.beforeEach(async ({ page }) => {
        await page.goto("/log/01-web-app/nginx");
        await waitForApp(page);
    });

    test("displays log heading with service and stack name", async ({ page }) => {
        await expect(page.getByRole("heading", { name: /Log.*nginx.*01-web-app/i })).toBeVisible({ timeout: 10000 });
    });

    test("terminal element is visible", async ({ page }) => {
        await expect(page.locator(".shadow-box.terminal")).toBeVisible({ timeout: 10000 });
    });

    test("screenshot: container log", async ({ page }) => {
        await expect(page.locator(".shadow-box.terminal")).toBeVisible({ timeout: 10000 });
        await expect(page).toHaveScreenshot("container-log.png");
    });
});
