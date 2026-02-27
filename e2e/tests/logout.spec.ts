import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";

test.describe("Logout Flow", () => {
    test("logout via profile dropdown redirects to login form", async ({ browser }) => {
        // Create a fresh browser context with stored auth state so we start logged in.
        // This isolates the logout from other tests' shared context.
        const context = await browser.newContext({
            storageState: ".auth/user.json",
            colorScheme: "dark",
            viewport: { width: 1280, height: 720 },
        });
        const page = await context.newPage();

        await page.goto("/");
        await waitForApp(page);

        // Open profile dropdown
        await page.getByRole("button", { name: "User menu" }).click();
        await expect(page.getByText("Logout")).toBeVisible();

        // Click Logout
        await page.getByText("Logout").click();

        // Verify redirect to login form
        await expect(page.getByPlaceholder("Username")).toBeVisible({ timeout: 15000 });
        await expect(page.getByPlaceholder("Password")).toBeVisible();
        await expect(page.getByRole("button", { name: "Login" })).toBeVisible();

        // Verify we can log back in
        await page.getByPlaceholder("Username").fill("admin");
        await page.getByPlaceholder("Password").fill("testpass123");
        await page.getByRole("button", { name: "Login" }).click();
        await expect(page.getByRole("heading", { name: "Stacks" })).toBeVisible({ timeout: 15000 });

        await context.close();
    });
});
