import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";

test.describe("Compose Edit Mode", () => {
    test.beforeEach(async ({ page }) => {
        await page.goto("/compose/01-web-app");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: "01-web-app" })).toBeVisible({ timeout: 15000 });
    });

    test("clicking Edit shows Deploy, Save, and Discard buttons", async ({ page }) => {
        await page.getByRole("button", { name: "Edit" }).click();

        await expect(page.getByRole("button", { name: "Deploy" })).toBeVisible();
        await expect(page.getByRole("button", { name: "Save" })).toBeVisible();
        await expect(page.getByRole("button", { name: "Discard" })).toBeVisible();
    });

    test("editor gets edit-mode class", async ({ page }) => {
        await page.getByRole("button", { name: "Edit" }).click();

        await expect(page.locator(".editor-box.edit-mode").first()).toBeVisible();
    });

    test("screenshot: compose edit mode", async ({ page }) => {
        await page.getByRole("button", { name: "Edit" }).click();
        await expect(page.getByRole("button", { name: "Deploy" })).toBeVisible();
        await expect(page.locator(".editor-box.edit-mode").first()).toBeVisible();
        // Edit mode adds .env editor below, which can cause a scroll — reset to top
        await page.evaluate(() => window.scrollTo(0, 0));
        await page.waitForTimeout(500);
        await expect(page).toHaveScreenshot("compose-edit-mode.png");
    });
});
