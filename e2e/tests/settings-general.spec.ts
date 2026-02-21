import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";

test.describe("Settings — General", () => {
    test.beforeEach(async ({ page }) => {
        await page.goto("/settings/general");
        await waitForApp(page);
    });

    test("shows settings sidebar with all menu items", async ({ page }) => {
        await expect(page.getByRole("link", { name: "General" })).toBeVisible();
        await expect(page.getByRole("link", { name: "Appearance" })).toBeVisible();
        await expect(page.getByRole("link", { name: "Security" })).toBeVisible();
        await expect(page.getByRole("link", { name: /Global/i })).toBeVisible();
        await expect(page.getByRole("link", { name: "About" })).toBeVisible();
    });

    test("displays General content header", async ({ page }) => {
        await expect(page.locator(".settings-content-header")).toContainText("General");
    });

    test("shows Primary Hostname section", async ({ page }) => {
        await expect(page.getByText("Primary Hostname")).toBeVisible();
        await expect(page.getByRole("button", { name: /Auto Get/i })).toBeVisible();
    });

    test("shows Image Update Checking section", async ({ page }) => {
        await expect(page.getByText(/Image Update Check/i)).toBeVisible();
    });

    test("shows Save button", async ({ page }) => {
        await expect(page.getByRole("button", { name: "Save" })).toBeVisible();
    });

    test("screenshot: settings general", async ({ page }) => {
        await expect(page.getByRole("button", { name: "Save" })).toBeVisible();
        await expect(page).toHaveScreenshot("settings-general.png");
    });
});
