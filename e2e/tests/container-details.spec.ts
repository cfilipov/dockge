import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";

test.describe("Container Inspect", () => {
    test.beforeEach(async ({ page }) => {
        await page.goto("/containers/01-web-app-nginx-1");
        await waitForApp(page);
    });

    test("displays inspect UI elements", async ({ page }) => {
        await expect.soft(page.getByRole("heading", { name: /running\s+01-web-app-nginx-1/i })).toBeVisible({ timeout: 10000 });
        await expect.soft(page.locator(".inspect-grid").first()).toBeVisible({ timeout: 10000 });
    });

    test("raw toggle shows CodeMirror editor", async ({ page }) => {
        await page.locator("label[for='view-raw']").click();
        await expect(page.locator(".cm-editor").first()).toBeVisible({ timeout: 10000 });
    });

    test("screenshot: container inspect", async ({ page }) => {
        await expect(page.locator(".inspect-grid").first()).toBeVisible({ timeout: 10000 });
        await expect(page).toHaveScreenshot("container-inspect.png");
    });
});

test.describe("Container Log", () => {
    test.beforeEach(async ({ page }) => {
        // Navigate via the container card log button (not a direct URL)
        await page.goto("/stacks/01-web-app");
        await waitForApp(page);
        const logLink = page.locator("a[title='docker compose logs nginx']");
        await expect(logLink).toBeVisible({ timeout: 10000 });
        await logLink.click();
        await expect(page).toHaveURL("/logs/01-web-app-nginx-1");
    });

    test("displays log UI elements", async ({ page }) => {
        await expect.soft(page.getByRole("heading", { name: /running\s+01-web-app-nginx-1/i })).toBeVisible({ timeout: 10000 });
        await expect.soft(page.locator(".shadow-box.terminal")).toBeVisible({ timeout: 10000 });
    });

    test("screenshot: container log", async ({ page }) => {
        await expect(page.locator(".shadow-box.terminal")).toBeVisible({ timeout: 10000 });
        await expect(page).toHaveScreenshot("container-log.png");
    });
});

test.describe("Container Terminal", () => {
    test.beforeEach(async ({ page }) => {
        await page.goto("/terminal/01-web-app/nginx/bash");
        await waitForApp(page);
    });

    test("displays terminal UI elements", async ({ page }) => {
        await expect.soft(page.getByRole("heading", { name: /Terminal.*nginx.*01-web-app/i })).toBeVisible({ timeout: 10000 });
        await expect.soft(page.getByRole("link", { name: /Switch to sh/i })).toBeVisible();
        await expect.soft(page.locator(".shadow-box.terminal")).toBeVisible({ timeout: 10000 });
    });

    test("screenshot: container terminal", async ({ page }) => {
        await expect(page.locator(".shadow-box.terminal")).toBeVisible({ timeout: 10000 });
        await expect(page).toHaveScreenshot("container-terminal.png");
    });
});
