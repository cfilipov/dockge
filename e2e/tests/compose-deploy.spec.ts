import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";

const terminal = (page: import("@playwright/test").Page) =>
    page.locator(".progress-terminal .xterm-rows");

test.describe("Deploy New Stack", () => {

    // Reset mock state before and after so we don't leave the new stack behind.
    test.beforeAll(async ({ request }) => {
        await request.post("/api/mock/reset");
    });
    test.afterAll(async ({ request }) => {
        await request.post("/api/mock/reset");
    });

    test("deploy new stack end-to-end", async ({ page }) => {
        await page.goto("/stacks/new");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: "Compose" })).toBeVisible({ timeout: 15000 });

        // Fill stack name
        const nameInput = page.locator("#name");
        await expect(nameInput).toBeVisible();
        await nameInput.fill("my-test-deploy");

        // The create page already has a valid default template (nginx:latest with ports).
        // Verify the YAML editor is visible and has content, then deploy directly.
        const cmEditor = page.locator(".cm-editor").first();
        await expect(cmEditor).toBeVisible({ timeout: 10000 });

        // Click Deploy
        const deployBtn = page.getByRole("button", { name: "Deploy" });
        await expect(deployBtn).toBeVisible();
        await deployBtn.evaluate((el: HTMLElement) => el.click());

        // Verify terminal output shows compose up
        const term = terminal(page);
        await expect(term).toContainText("docker compose", { timeout: 15000 });
        await expect(term).toContainText("[Done]", { timeout: 15000 });

        // Verify redirect to new stack page
        await expect(page).toHaveURL(/\/stacks\/my-test-deploy/, { timeout: 10000 });

        // Verify stack heading is displayed
        await expect(page.getByRole("heading", { name: /my-test-deploy/, level: 1 })).toBeVisible({ timeout: 10000 });

        // Verify stack appears in sidebar
        const sidebarItem = page.locator(".item").filter({ hasText: "my-test-deploy" });
        await expect(sidebarItem).toBeVisible({ timeout: 10000 });
    });
});
