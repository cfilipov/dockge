import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";

// Use evaluate(el.click()) instead of Playwright's .click() for the Edit
// button. Playwright's click() calls scrollIntoViewIfNeeded before clicking,
// which can cause a ~150 px scroll when a concurrent WebSocket event shifts
// the layout during the click preparation.
function clickEdit(page: import("@playwright/test").Page) {
    return page.getByRole("button", { name: "Edit" }).evaluate((el: HTMLElement) => el.click());
}

test.describe("Compose Edit Mode", () => {
    test.beforeEach(async ({ page }) => {
        await page.goto("/stacks/01-web-app");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: "01-web-app" })).toBeVisible({ timeout: 15000 });
    });

    test("clicking Edit shows Deploy, Save, and Discard buttons", async ({ page }) => {
        await clickEdit(page);

        await expect(page.getByRole("button", { name: "Deploy" })).toBeVisible();
        await expect(page.getByRole("button", { name: "Save" })).toBeVisible();
        await expect(page.getByRole("button", { name: "Discard" })).toBeVisible();
    });

    test("editor gets edit-mode class", async ({ page }) => {
        await clickEdit(page);

        await expect(page.locator(".editor-box.edit-mode").first()).toBeVisible();
    });

    test("screenshot: compose edit mode", async ({ page }) => {
        await clickEdit(page);
        await expect(page.getByRole("button", { name: "Deploy" })).toBeVisible();
        await expect(page.locator(".editor-box.edit-mode").first()).toBeVisible();
        await expect(page).toHaveScreenshot("compose-edit-mode.png");
    });
});
