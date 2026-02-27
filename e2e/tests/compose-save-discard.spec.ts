import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";

// Use evaluate(el.click()) to avoid scroll interference from WebSocket layout shifts
function clickEdit(page: import("@playwright/test").Page) {
    return page.getByRole("button", { name: "Edit" }).evaluate((el: HTMLElement) => el.click());
}

test.describe("Compose — Save Draft & Discard", () => {

    // Reset mock state before and after to avoid cross-test interference.
    test.beforeAll(async ({ request }) => {
        await request.post("/api/mock/reset");
    });
    test.afterAll(async ({ request }) => {
        await request.post("/api/mock/reset");
    });

    test("save draft persists YAML changes", async ({ page }) => {
        // Use a stable filler stack for this test
        await page.goto("/stacks/stack-011");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: /stack-011/, level: 1 })).toBeVisible({ timeout: 15000 });

        // The YAML editor is the first .cm-content inside .editor-box
        const yamlEditor = page.locator(".editor-box .cm-content").first();
        await expect(yamlEditor).toBeVisible({ timeout: 10000 });

        // Enter edit mode
        await clickEdit(page);
        await expect(page.getByRole("button", { name: "Save" })).toBeVisible();

        // Modify YAML — append a comment-like line
        await yamlEditor.click();
        await page.keyboard.press("ControlOrMeta+End");
        await page.keyboard.press("Enter");
        await page.keyboard.type("# test-save-marker");

        // Click Save
        await page.getByRole("button", { name: "Save" }).evaluate((el: HTMLElement) => el.click());

        // Wait for save to complete
        await page.waitForTimeout(1000);

        // Reload and verify persistence
        await page.reload();
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: /stack-011/, level: 1 })).toBeVisible({ timeout: 15000 });
        const reloadedEditor = page.locator(".editor-box .cm-content").first();
        await expect(reloadedEditor).toBeVisible({ timeout: 10000 });
        await expect(reloadedEditor).toContainText("test-save-marker");
    });

    test("discard reverts YAML changes", async ({ page }) => {
        await page.goto("/stacks/stack-012");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: /stack-012/, level: 1 })).toBeVisible({ timeout: 15000 });

        // The YAML editor
        const yamlEditor = page.locator(".editor-box .cm-content").first();
        await expect(yamlEditor).toBeVisible({ timeout: 10000 });

        // Enter edit mode
        await clickEdit(page);
        await expect(page.getByRole("button", { name: "Discard" })).toBeVisible();

        // Modify YAML
        await yamlEditor.click();
        await page.keyboard.press("ControlOrMeta+End");
        await page.keyboard.press("Enter");
        await page.keyboard.type("# discard-test-marker");

        // Verify the modification is in the editor
        await expect(yamlEditor).toContainText("discard-test-marker");

        // Click Discard
        await page.getByRole("button", { name: "Discard" }).click();

        // Verify YAML reverts — the marker should be gone
        await expect(yamlEditor).not.toContainText("discard-test-marker", { timeout: 5000 });

        // Verify we're back in view mode
        await expect(page.getByRole("button", { name: "Edit" })).toBeVisible();
    });
});
