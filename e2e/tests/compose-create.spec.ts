import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";

test.describe("Compose Create — New Stack", () => {
    test.beforeEach(async ({ page }) => {
        await page.goto("/stacks/new");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: "Compose" })).toBeVisible({ timeout: 15000 });
    });

    test("displays Compose heading", async ({ page }) => {
        await expect(page.getByRole("heading", { name: "Compose" })).toBeVisible();
    });

    test("shows Stack Name input", async ({ page }) => {
        await expect(page.locator("#name")).toBeVisible();
    });

    test("shows Containers section with Add Container", async ({ page }) => {
        await expect(page.getByRole("heading", { name: "Containers" })).toBeVisible();
        await expect(page.getByPlaceholder("New Container Name...")).toBeVisible();
        await expect(page.getByRole("button", { name: "Add Container" })).toBeVisible();
    });

    test("shows Networks heading", async ({ page }) => {
        await expect(page.getByRole("heading", { name: "Networks", exact: true })).toBeVisible();
    });

    test("shows Deploy and Save buttons", async ({ page }) => {
        await expect(page.getByRole("button", { name: "Deploy" })).toBeVisible();
        await expect(page.getByRole("button", { name: "Save" })).toBeVisible();
    });

    test("shows CodeMirror editor", async ({ page }) => {
        await expect(page.locator(".cm-editor").first()).toBeVisible();
    });

    test("screenshot: compose create page", async ({ page }) => {
        await expect(page.locator(".cm-editor").first()).toBeVisible();
        await expect(page).toHaveScreenshot("compose-create.png");
    });
});
