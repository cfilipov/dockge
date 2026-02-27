import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";

test.describe("Deploy — Invalid YAML", () => {

    test.beforeAll(async ({ request }) => {
        await request.post("/api/mock/reset");
    });
    test.afterAll(async ({ request }) => {
        await request.post("/api/mock/reset");
    });

    test("deploy with no-services YAML creates inactive stack", async ({ page }) => {
        // YAML without a "services:" key passes frontend validation
        // (yamlToJSON auto-adds "services: {}"), but backend validation
        // (docker compose config --dry-run) fails in a background goroutine
        // because the saved file has no "services:" line. As a result,
        // docker compose up is never called and the stack stays inactive.

        await page.goto("/stacks/new");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: "Compose" })).toBeVisible({ timeout: 15000 });

        // Fill stack name
        const nameInput = page.locator("#name");
        await nameInput.fill("invalid-yaml-test");

        // Replace the YAML editor content with invalid YAML (no services: key)
        const cmContent = page.locator(".cm-content").first();
        await expect(cmContent).toBeVisible({ timeout: 10000 });
        await cmContent.click();
        await page.keyboard.press("ControlOrMeta+a");
        await page.keyboard.press("Delete");
        await page.keyboard.type("foo: bar", { delay: 30 });
        await expect(cmContent).toContainText("foo: bar");

        // Click Deploy — backend acks before background validation runs,
        // so the frontend navigates to the new stack page.
        const deployBtn = page.getByRole("button", { name: "Deploy" });
        await deployBtn.evaluate((el: HTMLElement) => el.click());

        // Verify navigation to the new stack page
        await expect(page).toHaveURL(/\/stacks\/invalid-yaml-test/, { timeout: 15000 });
        await expect(page.getByRole("heading", { name: /invalid-yaml-test/ })).toBeVisible({ timeout: 15000 });

        // Wait for the page to fully load (Edit button always present for managed stacks)
        await expect(page.getByRole("button", { name: "Edit" })).toBeVisible({ timeout: 15000 });

        // Stack should NOT be running — Start is shown for inactive stacks,
        // Stop/Restart are only shown for active (running) stacks.
        await expect(page.getByRole("button", { name: "Start" })).toBeVisible();
        await expect(page.getByRole("button", { name: "Stop" })).not.toBeVisible();
    });
});
