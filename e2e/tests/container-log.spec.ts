import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";

test.describe("Container Log", () => {
    test.beforeEach(async ({ page }) => {
        // Navigate via the container card log button (not a direct URL)
        await page.goto("/stacks/01-web-app");
        await waitForApp(page);
        const logLink = page.locator("a[title='docker compose logs nginx']");
        await expect(logLink).toBeVisible({ timeout: 10000 });
        await logLink.click();
        await expect(page).toHaveURL("/logs/01-web-app-nginx-1");
    });

    test("displays log heading with container name", async ({ page }) => {
        await expect(page.getByRole("heading", { name: /active\s+01-web-app-nginx-1/i })).toBeVisible({ timeout: 10000 });
    });

    test("terminal element is visible", async ({ page }) => {
        await expect(page.locator(".shadow-box.terminal")).toBeVisible({ timeout: 10000 });
    });

    test("screenshot: container log", async ({ page }) => {
        await expect(page.locator(".shadow-box.terminal")).toBeVisible({ timeout: 10000 });
        await expect(page).toHaveScreenshot("container-log.png");
    });
});
