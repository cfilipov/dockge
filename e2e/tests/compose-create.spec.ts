import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";

test.describe("Compose Create — New Stack", () => {
    test.beforeEach(async ({ page }) => {
        await page.goto("/stacks/new");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: "Compose" })).toBeVisible({ timeout: 15000 });
    });

    test("displays new stack form elements", async ({ page }) => {
        await expect.soft(page.getByRole("heading", { name: "Compose" })).toBeVisible();
        await expect.soft(page.locator("#name")).toBeVisible();
        await expect.soft(page.getByRole("heading", { name: "Containers" })).toBeVisible();
        await expect.soft(page.getByPlaceholder("New Container Name...")).toBeVisible();
        await expect.soft(page.getByRole("button", { name: "Add Container" })).toBeVisible();
        await expect.soft(page.getByRole("heading", { name: /^Networks/ })).toBeVisible();
        await expect.soft(page.getByRole("button", { name: "Deploy" })).toBeVisible();
        await expect.soft(page.getByRole("button", { name: "Save" })).toBeVisible();
        await expect.soft(page.locator(".cm-editor").first()).toBeVisible();
    });

    test("screenshot: compose create page", async ({ page }) => {
        await expect(page.locator(".cm-editor").first()).toBeVisible();
        await expect(page).toHaveScreenshot("compose-create.png");
    });
});
