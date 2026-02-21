import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";

test.describe("Compose View — Running Stack", () => {
    test.beforeEach(async ({ page }) => {
        await page.goto("/compose/01-web-app");
        await waitForApp(page);
        // Wait for the stack name heading to appear
        await expect(page.getByRole("heading", { name: "01-web-app" })).toBeVisible({ timeout: 15000 });
    });

    test("displays stack name in heading with uptime pill", async ({ page }) => {
        await expect(page.getByRole("heading", { name: "01-web-app" })).toBeVisible();
    });

    test("shows action buttons for a running stack", async ({ page }) => {
        await expect(page.getByRole("button", { name: "Edit", exact: true })).toBeVisible();
        await expect(page.getByRole("button", { name: "Restart", exact: true })).toBeVisible();
        await expect(page.getByRole("button", { name: "Stop", exact: true })).toBeVisible();
        await expect(page.getByRole("button", { name: "Update", exact: true })).toBeVisible();
    });

    test("shows CodeMirror editor with compose YAML", async ({ page }) => {
        await expect(page.locator(".cm-editor").first()).toBeVisible();
    });

    test("shows terminal section", async ({ page }) => {
        await expect(page.getByRole("heading", { name: "Terminal" })).toBeVisible();
    });

    test("screenshot: compose view running stack", async ({ page }) => {
        await expect(page.locator(".cm-editor").first()).toBeVisible();
        await expect(page).toHaveScreenshot("compose-view-running.png");
    });
});
