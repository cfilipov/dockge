import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";

// Use evaluate(el.click()) instead of Playwright's .click() for the Edit
// button. Playwright's click() calls scrollIntoViewIfNeeded before clicking,
// which can cause a ~150 px scroll when a concurrent WebSocket event shifts
// the layout during the click preparation.
function clickEdit(page: import("@playwright/test").Page) {
    return page.getByRole("button", { name: "Edit" }).evaluate((el: HTMLElement) => el.click());
}

test.describe("Compose View — Running Stack", () => {
    test.beforeEach(async ({ page }) => {
        await page.goto("/stacks/01-web-app");
        await waitForApp(page);
        // Wait for the stack name heading to appear
        await expect(page.getByRole("heading", { name: /01-web-app/ })).toBeVisible({ timeout: 15000 });
    });

    test("displays stack view UI elements", async ({ page }) => {
        await expect.soft(page.getByRole("heading", { name: /01-web-app/ })).toBeVisible();
        await expect.soft(page.getByRole("button", { name: "Edit", exact: true })).toBeVisible();
        await expect.soft(page.getByRole("button", { name: "Restart", exact: true })).toBeVisible();
        await expect.soft(page.getByRole("button", { name: "Stop", exact: true })).toBeVisible();
        await expect.soft(page.getByRole("button", { name: "Update", exact: true })).toBeVisible();
        await expect.soft(page.locator(".cm-editor").first()).toBeVisible();
        await expect.soft(page.getByRole("heading", { name: "Logs" })).toBeVisible();
    });

    test("screenshot: compose view running stack", async ({ page }) => {
        await expect(page.locator(".cm-editor").first()).toBeVisible();
        await expect(page).toHaveScreenshot("compose-view-running.png");
    });
});

test.describe("Compose Edit Mode", () => {
    test.beforeEach(async ({ page }) => {
        await page.goto("/stacks/01-web-app");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: /01-web-app/ })).toBeVisible({ timeout: 15000 });
    });

    test("edit mode shows deploy/save/discard and editor class", async ({ page }) => {
        await clickEdit(page);

        await expect.soft(page.getByRole("button", { name: "Deploy" })).toBeVisible();
        await expect.soft(page.getByRole("button", { name: "Save" })).toBeVisible();
        await expect.soft(page.getByRole("button", { name: "Discard" })).toBeVisible();
        await expect.soft(page.locator(".editor-box.edit-mode").first()).toBeVisible();
    });

    test("screenshot: compose edit mode", async ({ page }) => {
        await clickEdit(page);
        await expect(page.getByRole("button", { name: "Deploy" })).toBeVisible();
        await expect(page.locator(".editor-box.edit-mode").first()).toBeVisible();
        await expect(page).toHaveScreenshot("compose-edit-mode.png");
    });
});
