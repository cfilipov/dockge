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
});
