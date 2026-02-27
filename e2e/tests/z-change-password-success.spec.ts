import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";

test.describe("Change Password — Success", () => {
    // This test actually changes the admin password, so we need to restore it
    // afterwards. The test verifies the full happy path:
    //   1. Fill current + new + repeat → submit
    //   2. Backend broadcasts "refresh" → user is forced to re-login
    //   3. Logging in with the new password works
    //   4. Restore original password so other tests aren't broken

    test("successfully changes password and can log in with new one", async ({ page }) => {
        await page.goto("/settings/security");
        await waitForApp(page);
        await expect(page.getByText("Change Password")).toBeVisible({ timeout: 10000 });

        const currentPw = "testpass123";
        const newPw = "newpass456";

        // Fill in the change password form
        await page.getByLabel("Current Password").first().fill(currentPw);
        await page.getByLabel("New Password", { exact: true }).fill(newPw);
        await page.getByLabel("Repeat New Password").fill(newPw);

        // Submit
        await page.getByRole("button", { name: "Update Password" }).click();

        // The backend broadcasts "refresh" which forces page reload → login page.
        // The redirect to login confirms the password was changed and sessions invalidated.
        await expect(page.getByRole("button", { name: "Login" })).toBeVisible({ timeout: 15000 });

        // Log in with the NEW password
        await page.getByLabel("Username").fill("admin");
        await page.getByLabel("Password").fill(newPw);
        await page.getByRole("button", { name: "Login" }).click();

        // Verify we're back in the app
        await waitForApp(page);

        // Restore the original password so other tests aren't broken
        await page.goto("/settings/security");
        await waitForApp(page);
        await expect(page.getByText("Change Password")).toBeVisible({ timeout: 10000 });

        await page.getByLabel("Current Password").first().fill(newPw);
        await page.getByLabel("New Password", { exact: true }).fill(currentPw);
        await page.getByLabel("Repeat New Password").fill(currentPw);
        await page.getByRole("button", { name: "Update Password" }).click();

        // Wait for the refresh → login redirect to confirm restore worked
        await expect(page.getByRole("button", { name: "Login" })).toBeVisible({ timeout: 15000 });
    });
});
