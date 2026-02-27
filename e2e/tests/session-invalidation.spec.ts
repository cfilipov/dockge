import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";

test.describe("Session Invalidation", () => {
    test("protected route shows login form after logout", async ({ browser }) => {
        // Create an isolated context with stored auth state
        const context = await browser.newContext({
            storageState: ".auth/user.json",
            colorScheme: "dark",
            viewport: { width: 1280, height: 720 },
        });
        const page = await context.newPage();

        // Verify we're logged in
        await page.goto("/");
        await waitForApp(page);

        // Logout via profile dropdown
        await page.getByRole("button", { name: "User menu" }).click();
        await expect(page.getByText("Logout")).toBeVisible();
        await page.getByText("Logout").click();

        // Wait for login form to appear
        await expect(page.getByRole("button", { name: "Login" })).toBeVisible({ timeout: 15000 });

        // Now navigate to a protected route
        await page.goto("/stacks/01-web-app");

        // The login form should appear instead of the stack page.
        // The Layout component hides <router-view> when loggedIn=false
        // and shows the Login component instead.
        await expect(page.getByRole("button", { name: "Login" })).toBeVisible({ timeout: 15000 });

        // The stack heading should NOT be visible
        await expect(page.getByRole("heading", { name: /01-web-app/ })).not.toBeVisible();

        await context.close();
    });
});
