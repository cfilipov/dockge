import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";
import { takeLightScreenshot } from "../helpers/light-mode";

test.describe("Settings — General", () => {
    test.beforeEach(async ({ page }) => {
        await page.goto("/settings/general");
        await waitForApp(page);
    });

    test("displays settings UI elements", async ({ page }) => {
        // Sidebar menu items
        await expect.soft(page.getByRole("link", { name: "General" })).toBeVisible();
        await expect.soft(page.getByRole("link", { name: "Appearance" })).toBeVisible();
        await expect.soft(page.getByRole("link", { name: "Security" })).toBeVisible();
        await expect.soft(page.getByRole("link", { name: /Global/i })).toBeVisible();
        await expect.soft(page.getByRole("link", { name: "About" })).toBeVisible();
        // Content header
        await expect.soft(page.locator(".settings-content-header")).toContainText("General");
        // Primary Hostname
        await expect.soft(page.getByText("Primary Hostname")).toBeVisible();
        await expect.soft(page.getByRole("button", { name: /Auto Get/i })).toBeVisible();
        // Image Update Checking
        await expect.soft(page.getByText(/Image Update Check/i)).toBeVisible();
        // Save button
        await expect.soft(page.getByRole("button", { name: "Save" })).toBeVisible();
    });

    test("screenshot: settings general", async ({ page }) => {
        await expect(page.getByRole("button", { name: "Save" })).toBeVisible();
        await expect(page).toHaveScreenshot("settings-general.png");
        await takeLightScreenshot(page, "settings-general-light.png");
    });
});

test.describe("Settings — Appearance", () => {
    test.beforeEach(async ({ page }) => {
        await page.goto("/settings/appearance");
        await waitForApp(page);
    });

    test("displays appearance UI elements", async ({ page }) => {
        await expect.soft(page.locator(".settings-content-header")).toContainText("Appearance");
        await expect.soft(page.locator("#language")).toBeVisible();
        await expect.soft(page.getByText("Light")).toBeVisible();
        await expect.soft(page.getByText("Dark")).toBeVisible();
        await expect.soft(page.getByText("Auto")).toBeVisible();
    });

    test("theme switching works", async ({ page }) => {
        // Click Light theme
        await page.getByText("Light", { exact: true }).click();
        await page.waitForTimeout(500);

        // Click Dark theme
        await page.getByText("Dark", { exact: true }).click();
        await page.waitForTimeout(500);
    });

    test("screenshot: appearance settings", async ({ page }) => {
        await expect(page.getByText("Light")).toBeVisible();
        await expect(page).toHaveScreenshot("settings-appearance.png");
        await takeLightScreenshot(page, "settings-appearance-lightmode.png");
    });

    test("screenshot: light theme", async ({ page }) => {
        await page.getByText("Light", { exact: true }).click();
        await page.waitForTimeout(1000);
        await page.evaluate(() => window.scrollTo(0, 0));
        await expect(page).toHaveScreenshot("settings-appearance-light.png");
    });

    test("screenshot: dark theme", async ({ page }) => {
        await page.getByText("Dark", { exact: true }).click();
        await page.waitForTimeout(1000);
        await page.evaluate(() => window.scrollTo(0, 0));
        await expect(page).toHaveScreenshot("settings-appearance-dark.png");
    });
});

test.describe("Settings — Security", () => {
    test.beforeEach(async ({ page }) => {
        await page.goto("/settings/security");
        await waitForApp(page);
    });

    test("displays security UI elements", async ({ page }) => {
        await expect.soft(page.locator(".settings-content-header")).toContainText("Security");
        await expect.soft(page.getByText(/Current User/i)).toBeVisible();
        await expect.soft(page.locator("#logout-btn")).toBeVisible();
        await expect.soft(page.getByText("Change Password")).toBeVisible();
        await expect.soft(page.locator("#current-password").first()).toBeVisible();
        await expect.soft(page.locator("#new-password")).toBeVisible();
        await expect.soft(page.locator("#repeat-new-password")).toBeVisible();
        await expect.soft(page.getByRole("button", { name: "Update Password" })).toBeVisible();
        await expect.soft(page.getByText("Advanced")).toBeVisible();
        await expect.soft(page.locator("#disableAuth-btn")).toBeVisible();
    });

    test("screenshot: settings security", async ({ page }) => {
        await expect(page.locator("#disableAuth-btn")).toBeVisible();
        await expect(page).toHaveScreenshot("settings-security.png");
        await takeLightScreenshot(page, "settings-security-light.png");
    });
});

test.describe("Settings — Global .env", () => {
    test.beforeEach(async ({ page }) => {
        await page.goto("/settings/globalEnv");
        await waitForApp(page);
    });

    test("displays global env UI elements", async ({ page }) => {
        await expect.soft(page.locator(".settings-content-header")).toContainText("Global");
        await expect.soft(page.locator(".cm-editor").first()).toBeVisible({ timeout: 10000 });
        await expect.soft(page.getByRole("button", { name: "Save" })).toBeVisible();
    });

    test("screenshot: settings global env", async ({ page }) => {
        await expect(page.locator(".cm-editor").first()).toBeVisible({ timeout: 10000 });
        await expect(page).toHaveScreenshot("settings-globalenv.png");
        await takeLightScreenshot(page, "settings-globalenv-light.png");
    });

    // This test must run AFTER the screenshot test because it modifies the
    // global.env content on the shared backend, which would change the editor
    // contents for subsequent page loads.
    test("saves and persists global env content", async ({ page }) => {
        const editor = page.locator(".cm-content");
        await expect(editor).toBeVisible({ timeout: 10000 });
        // Clear and type new content
        await editor.click();
        await page.keyboard.press("ControlOrMeta+a");
        await page.keyboard.type("MY_GLOBAL_VAR=hello_world");
        // Save
        await page.getByRole("button", { name: "Save" }).click();
        // Wait for save to complete
        await page.waitForTimeout(1000);
        // Reload and verify persistence
        await page.reload();
        await waitForApp(page);
        await expect(editor).toBeVisible({ timeout: 10000 });
        await expect(editor).toContainText("MY_GLOBAL_VAR=hello_world");
    });
});

test.describe("Settings — About", () => {
    test.beforeEach(async ({ page }) => {
        await page.goto("/settings/about");
        await waitForApp(page);
    });

    test("displays about UI elements", async ({ page }) => {
        await expect.soft(page.locator(".settings-content-header")).toContainText("About");
        await expect.soft(page.getByRole("main").getByText("Dockge")).toBeVisible();
        await expect.soft(page.getByText(/Version/i).first()).toBeVisible();
        await expect.soft(page.getByRole("link", { name: /Check Update/i })).toBeVisible();
        await expect.soft(page.getByText(/update if available/i)).toBeVisible();
        await expect.soft(page.getByText(/beta release/i)).toBeVisible();
    });

    test("screenshot: settings about", async ({ page }) => {
        await expect(page.getByRole("main").getByText("Dockge")).toBeVisible();
        await expect(page).toHaveScreenshot("settings-about.png");
        await takeLightScreenshot(page, "settings-about-light.png");
    });
});
