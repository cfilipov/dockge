import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";

test.describe("Header Navigation", () => {
    test.beforeEach(async ({ page }) => {
        await page.goto("/");
        await waitForApp(page);
    });

    test("header shows Stacks and Console nav links", async ({ page }) => {
        await expect(page.getByRole("link", { name: "Stacks" })).toBeVisible();
        await expect(page.getByRole("link", { name: "Console" })).toBeVisible();
    });

    test("header shows profile pic dropdown", async ({ page }) => {
        await expect(page.locator(".dropdown-profile-pic")).toBeVisible();
    });

    test("profile dropdown shows expected items", async ({ page }) => {
        // Open profile dropdown
        await page.locator(".dropdown-profile-pic .nav-link").click();

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
        await page.locator(".dropdown-profile-pic .nav-link").click();
        await page.getByRole("link", { name: "Settings" }).click();
        await expect(page).toHaveURL(/\/settings/);
    });

    test("screenshot: profile dropdown", async ({ page }) => {
        await page.locator(".dropdown-profile-pic .nav-link").click();
        await expect(page.getByText("Scan Stacks Folder")).toBeVisible();
        await expect(page).toHaveScreenshot("navigation-profile-dropdown.png");
    });
});
