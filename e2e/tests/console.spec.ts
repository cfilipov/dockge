import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";
import { takeLightScreenshot } from "../helpers/light-mode";

test.describe("Console Page", () => {
    test.beforeEach(async ({ page }) => {
        await page.goto("/console");
        await waitForApp(page);
    });

    test("displays console UI elements", async ({ page }) => {
        await expect.soft(page.getByRole("heading", { name: "Console" })).toBeVisible({ timeout: 10000 });
        // Console may be enabled (terminal visible) or disabled (warning alert)
        const terminal = page.getByRole("region", { name: "Terminal" });
        const alert = page.getByRole("alert");
        await expect.soft(terminal.or(alert)).toBeVisible({ timeout: 10000 });
    });

    test("screenshot: console page", async ({ page }) => {
        // Wait for content to settle
        const terminal = page.getByRole("region", { name: "Terminal" });
        const alert = page.getByRole("alert");
        await expect(terminal.or(alert)).toBeVisible({ timeout: 10000 });
        await expect(page).toHaveScreenshot("console.png");
        await takeLightScreenshot(page, "console-light.png");
    });
});
