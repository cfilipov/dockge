import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";

test.describe("Container Inspect", () => {
    test.beforeEach(async ({ page }) => {
        await page.goto("/containers/01-web-app-nginx-1");
        await waitForApp(page);
    });

    test("displays inspect heading", async ({ page }) => {
        await expect(page.getByRole("heading", { name: /Inspect/i })).toBeVisible({ timeout: 10000 });
    });

    test("shows CodeMirror editor with read-only content", async ({ page }) => {
        await expect(page.locator(".cm-editor").first()).toBeVisible({ timeout: 10000 });
    });

    test("screenshot: container inspect", async ({ page }) => {
        await expect(page.locator(".cm-editor").first()).toBeVisible({ timeout: 10000 });
        await expect(page).toHaveScreenshot("container-inspect.png");
    });
});
