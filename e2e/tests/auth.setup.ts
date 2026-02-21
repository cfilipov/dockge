import { test as setup, expect } from "@playwright/test";

const authFile = "e2e/.auth/user.json";

setup("authenticate", async ({ page }) => {
    await page.goto("/");

    // Wait for login form to appear
    await expect(page.getByPlaceholder("Username")).toBeVisible({ timeout: 15000 });

    // Fill credentials and submit
    await page.getByPlaceholder("Username").fill("admin");
    await page.getByPlaceholder("Password").fill("testpass123");
    await page.getByRole("button", { name: "Login" }).click();

    // Wait for successful login — "Home" heading appears on dashboard
    await expect(page.getByRole("heading", { name: "Home" })).toBeVisible({ timeout: 15000 });

    // Save authentication state
    await page.context().storageState({ path: authFile });
});
