import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";

test.describe("Docker Run → Compose Converter", () => {
    test.beforeEach(async ({ page }) => {
        await page.goto("/");
        await waitForApp(page);
        // Wait for dashboard to load
        await expect(page.getByRole("heading", { name: "Docker Run" })).toBeVisible({ timeout: 10000 });
    });

    test("converts a docker run command to compose YAML", async ({ page }) => {
        // Paste a docker run command into the textarea
        const textarea = page.getByPlaceholder("docker run ...");
        await expect(textarea).toBeVisible();
        await textarea.fill("docker run -d -p 8080:80 --name my-nginx nginx:latest");

        // Click Convert
        await page.getByRole("button", { name: /Convert/i }).click();

        // Should navigate to /stacks/new
        await expect(page).toHaveURL(/\/stacks\/new/, { timeout: 10000 });

        // The YAML editor should contain the converted compose content
        const editor = page.locator(".cm-content").first();
        await expect(editor).toBeVisible({ timeout: 10000 });

        // Verify key compose elements are present in the generated YAML
        await expect(editor).toContainText("nginx:latest");
        await expect(editor).toContainText("8080:80");
    });

    test("shows error for empty docker run command", async ({ page }) => {
        // Click Convert without entering a command
        await page.getByRole("button", { name: /Convert/i }).click();

        // Should show error toast — stay on dashboard
        await expect(page.getByText("Please enter a docker run command")).toBeVisible({ timeout: 5000 });
        // Verify we haven't navigated away
        await expect(page).not.toHaveURL(/\/stacks\/new/);
    });
});
