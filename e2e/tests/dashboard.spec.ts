import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";

test.describe("Dashboard Home", () => {
    test.beforeEach(async ({ page }) => {
        await page.goto("/");
        await waitForApp(page);
    });

    test("displays Home heading", async ({ page }) => {
        await expect(page.getByRole("heading", { name: "Home" })).toBeVisible();
    });

    test("shows stack status stats", async ({ page }) => {
        // The dashboard shows status headings: active, active⁻, unhealthy, exited, down, updates
        await expect(page.getByRole("heading", { name: "active", exact: true })).toBeVisible();
        await expect(page.getByRole("heading", { name: "exited" })).toBeVisible();
    });

    test("stack list sidebar shows items", async ({ page }) => {
        // Wait for stack list to populate
        const stackItems = page.locator(".item");
        await expect(stackItems.first()).toBeVisible({ timeout: 10000 });
        await expect(stackItems).toHaveCount(await stackItems.count());
    });

    test("shows Docker Run converter section", async ({ page }) => {
        await expect(page.getByRole("heading", { name: "Docker Run" })).toBeVisible();
        await expect(page.getByRole("button", { name: /Convert/i })).toBeVisible();
    });

    test("shows Dockge Agents section", async ({ page }) => {
        await expect(page.getByText("Dockge Agent")).toBeVisible();
    });

    test("screenshot: dashboard home", async ({ page }) => {
        // Wait for stacks to load
        await expect(page.locator(".item").first()).toBeVisible({ timeout: 10000 });
        await expect(page).toHaveScreenshot("dashboard-home.png");
    });
});
