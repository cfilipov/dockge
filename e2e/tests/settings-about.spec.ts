import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";

test.describe("Settings — About", () => {
    test.beforeEach(async ({ page }) => {
        await page.goto("/settings/about");
        await waitForApp(page);
    });

    test("displays About content header", async ({ page }) => {
        await expect(page.locator(".settings-content-header")).toContainText("About");
    });

    test("shows Dockge title and version info", async ({ page }) => {
        await expect(page.getByRole("main").getByText("Dockge")).toBeVisible();
        await expect(page.getByText(/Version/i).first()).toBeVisible();
    });

    test("shows Check Update On GitHub link", async ({ page }) => {
        await expect(page.getByRole("link", { name: /Check Update/i })).toBeVisible();
    });

    test("shows update checking checkboxes", async ({ page }) => {
        await expect(page.getByText(/update if available/i)).toBeVisible();
        await expect(page.getByText(/beta release/i)).toBeVisible();
    });

    test("screenshot: settings about", async ({ page }) => {
        await expect(page.getByRole("main").getByText("Dockge")).toBeVisible();
        await expect(page).toHaveScreenshot("settings-about.png");
    });
});
