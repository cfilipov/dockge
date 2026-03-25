import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";

// Use evaluate(el.click()) to avoid scroll interference from WebSocket layout shifts
function clickEdit(page: import("@playwright/test").Page) {
    return page.getByRole("button", { name: "Edit" }).evaluate((el: HTMLElement) => el.click());
}

test.describe("Fullscreen Editor Modals", () => {
    test.beforeEach(async ({ page }) => {
        await page.goto("/stacks/01-web-app");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: /01-web-app/, level: 1 })).toBeVisible({ timeout: 15000 });

        // Enter edit mode to reveal fullscreen buttons
        await clickEdit(page);
        await expect(page.getByRole("button", { name: "Deploy" })).toBeVisible();
    });

    test("compose YAML fullscreen modal opens and shows editor", async ({ page }) => {
        // Click the fullscreen button inside the compose YAML editor region
        const yamlRegion = page.getByRole("region", { name: "compose.yaml" });
        await yamlRegion.getByRole("button", { name: "Fullscreen" }).click();

        // Verify modal opens
        const modal = page.getByRole("dialog");
        await expect(modal).toBeVisible({ timeout: 5000 });
        await expect(modal.getByText("compose.yaml")).toBeVisible();

        // Verify CodeMirror editor is visible inside the modal
        await expect(modal.locator(".cm-editor")).toBeVisible();

        // Verify the editor contains the stack's YAML content
        await expect(modal.locator(".cm-content")).toContainText("services");
        await expect(modal.locator(".cm-content")).toContainText("nginx");
    });

    test("compose YAML fullscreen modal closes and preserves content", async ({ page }) => {
        const yamlRegion = page.getByRole("region", { name: "compose.yaml" });
        const inlineEditor = yamlRegion.locator(".cm-content");

        // Read initial content
        const initialContent = await inlineEditor.textContent();

        // Open fullscreen modal
        await yamlRegion.getByRole("button", { name: "Fullscreen" }).click();
        const modal = page.getByRole("dialog");
        await expect(modal).toBeVisible({ timeout: 5000 });

        // Close the modal via the X button
        await modal.getByRole("button", { name: "Close" }).click();
        await expect(modal).not.toBeVisible({ timeout: 5000 });

        // Verify inline editor still has the same content
        const afterContent = await inlineEditor.textContent();
        expect(afterContent).toBe(initialContent);
    });

    test(".env fullscreen modal opens and shows editor", async ({ page }) => {
        // The .env editor section is in a region with aria-label=".env"
        const envRegion = page.getByRole("region", { name: ".env" });
        await expect(envRegion).toBeVisible();
        await envRegion.getByRole("button", { name: "Fullscreen" }).click();

        // Verify modal opens with .env title
        const modal = page.getByRole("dialog");
        await expect(modal).toBeVisible({ timeout: 5000 });
        await expect(modal.getByText(".env")).toBeVisible();

        // Verify CodeMirror editor is visible inside the modal
        await expect(modal.locator(".cm-editor")).toBeVisible();
    });
});
