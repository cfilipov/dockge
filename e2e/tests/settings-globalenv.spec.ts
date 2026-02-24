import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";

test.describe("Settings — Global .env", () => {
    test.beforeEach(async ({ page }) => {
        await page.goto("/settings/globalEnv");
        await waitForApp(page);
    });

    test("displays Global .env content header", async ({ page }) => {
        await expect(page.locator(".settings-content-header")).toContainText("Global");
    });

    test("shows CodeMirror editor", async ({ page }) => {
        await expect(page.locator(".cm-editor").first()).toBeVisible({ timeout: 10000 });
    });

    test("shows Save button", async ({ page }) => {
        await expect(page.getByRole("button", { name: "Save" })).toBeVisible();
    });

    test("screenshot: settings global env", async ({ page }) => {
        await expect(page.locator(".cm-editor").first()).toBeVisible({ timeout: 10000 });
        await expect(page).toHaveScreenshot("settings-globalenv.png");
    });

    // This test must run AFTER the screenshot test because it modifies the
    // global.env content on the shared backend, which would change the editor
    // contents for subsequent page loads.
    test("saves and persists global env content", async ({ page }) => {
        const editor = page.locator(".cm-content");
        await expect(editor).toBeVisible({ timeout: 10000 });
        // Clear and type new content
        await editor.click();
        await page.keyboard.press("ControlOrMeta+a");
        await page.keyboard.type("MY_GLOBAL_VAR=hello_world");
        // Save
        await page.getByRole("button", { name: "Save" }).click();
        // Wait for save to complete
        await page.waitForTimeout(1000);
        // Reload and verify persistence
        await page.reload();
        await waitForApp(page);
        await expect(editor).toBeVisible({ timeout: 10000 });
        await expect(editor).toContainText("MY_GLOBAL_VAR=hello_world");
    });
});
