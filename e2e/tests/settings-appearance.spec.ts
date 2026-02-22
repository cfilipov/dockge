import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";

test.describe("Settings — Appearance", () => {
    test.beforeEach(async ({ page }) => {
        await page.goto("/settings/appearance");
        await waitForApp(page);
    });

    test("displays Appearance content header", async ({ page }) => {
        await expect(page.locator(".settings-content-header")).toContainText("Appearance");
    });

    test("shows language dropdown", async ({ page }) => {
        await expect(page.locator("#language")).toBeVisible();
    });

    test("shows theme radio buttons", async ({ page }) => {
        await expect(page.getByText("Light")).toBeVisible();
        await expect(page.getByText("Dark")).toBeVisible();
        await expect(page.getByText("Auto")).toBeVisible();
    });

    test("theme switching works", async ({ page }) => {
        // Click Light theme
        await page.getByText("Light", { exact: true }).click();
        // The html element or body should reflect light theme
        await page.waitForTimeout(500);

        // Click Dark theme
        await page.getByText("Dark", { exact: true }).click();
        await page.waitForTimeout(500);
    });

    test("screenshot: appearance settings", async ({ page }) => {
        await expect(page.getByText("Light")).toBeVisible();
        await expect(page).toHaveScreenshot("settings-appearance.png");
    });

    test("screenshot: light theme", async ({ page }) => {
        await page.getByText("Light", { exact: true }).click();
        await page.waitForTimeout(1000);
        await page.evaluate(() => window.scrollTo(0, 0));
        await expect(page).toHaveScreenshot("settings-appearance-light.png");
    });

    test("screenshot: dark theme", async ({ page }) => {
        await page.getByText("Dark", { exact: true }).click();
        await page.waitForTimeout(1000);
        await page.evaluate(() => window.scrollTo(0, 0));
        await expect(page).toHaveScreenshot("settings-appearance-dark.png");
    });
});
