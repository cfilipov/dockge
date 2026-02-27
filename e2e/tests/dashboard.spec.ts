import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";
import { takeLightScreenshot } from "../helpers/light-mode";

test.describe("Dashboard Home", () => {
    test.beforeEach(async ({ page }) => {
        await page.goto("/");
        await waitForApp(page);
    });

    test("displays dashboard UI elements", async ({ page }) => {
        await expect.soft(page.getByRole("heading", { name: "Stacks" })).toBeVisible();
        await expect.soft(page.getByRole("heading", { name: "active", exact: true })).toBeVisible();
        await expect.soft(page.getByRole("heading", { name: "exited" })).toBeVisible();
        await expect.soft(page.locator(".item").first()).toBeVisible({ timeout: 10000 });
        await expect.soft(page.getByRole("heading", { name: "Docker Run" })).toBeVisible();
        await expect.soft(page.getByRole("button", { name: /Convert/i })).toBeVisible();
        await expect.soft(page.getByText("Dockge Agent")).toBeVisible();
    });

    test("screenshot: dashboard home", async ({ page }) => {
        // Wait for stacks to load
        await expect(page.locator(".item").first()).toBeVisible({ timeout: 10000 });
        await expect(page).toHaveScreenshot("dashboard-home.png");
        await takeLightScreenshot(page, "dashboard-home-light.png");
    });
});

test.describe("Header Navigation", () => {
    test.beforeEach(async ({ page }) => {
        await page.goto("/");
        await waitForApp(page);
    });

    test("displays header nav elements", async ({ page }) => {
        await expect.soft(page.getByRole("link", { name: "Stacks" })).toBeVisible();
        await expect.soft(page.getByRole("link", { name: "Console" })).toBeVisible();
        await expect.soft(page.getByRole("button", { name: "User menu" })).toBeVisible();
    });

    test("profile dropdown shows expected items", async ({ page }) => {
        // Open profile dropdown
        await page.getByRole("button", { name: "User menu" }).click();

        await expect(page.getByText(/Signed in as/i)).toBeVisible();
        await expect(page.getByText("admin")).toBeVisible();
        await expect(page.getByText("Scan Stacks Folder")).toBeVisible();
        await expect(page.getByRole("link", { name: "Settings" })).toBeVisible();
        await expect(page.getByText("Logout")).toBeVisible();
    });

    test("clicking Compose link in sidebar navigates to /stacks/new", async ({ page }) => {
        // The "+" or compose link in the stack list
        const composeLink = page.locator("a[href='/stacks/new']").first();
        if (await composeLink.isVisible()) {
            await composeLink.click();
            await expect(page).toHaveURL(/\/stacks\/new$/);
        }
    });

    test("Settings link navigates to /settings", async ({ page }) => {
        // Open profile dropdown and click Settings
        await page.getByRole("button", { name: "User menu" }).click();
        await page.getByRole("link", { name: "Settings" }).click();
        await expect(page).toHaveURL(/\/settings/);
    });

    test("screenshot: profile dropdown", async ({ page }) => {
        await page.getByRole("button", { name: "User menu" }).click();
        await expect(page.getByText("Scan Stacks Folder")).toBeVisible();
        await expect(page).toHaveScreenshot("navigation-profile-dropdown.png");
        await takeLightScreenshot(page, "navigation-profile-dropdown-light.png");
    });
});
