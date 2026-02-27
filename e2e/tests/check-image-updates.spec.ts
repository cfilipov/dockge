import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";

test.describe("Check Image Updates", () => {
    test.beforeAll(async ({ request }) => {
        await request.post("/api/mock/reset");
    });
    test.afterAll(async ({ request }) => {
        await request.post("/api/mock/reset");
    });

    test("check updates via overflow menu succeeds", async ({ page }) => {
        // Use 01-web-app which is running and has services with images
        await page.goto("/stacks/01-web-app");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: /01-web-app/, level: 1 })).toBeVisible({ timeout: 15000 });

        // Open the overflow dropdown menu
        await page.getByRole("button", { name: "More actions" }).click();

        // Click "Check Updates"
        await page.getByText("Check Updates").click();

        // The check runs asynchronously; the backend acks with {ok: true, updated: true}.
        // During the check, action buttons are disabled (processing=true).
        // Wait for processing to finish — the Restart button should become enabled.
        const restartBtn = page.getByRole("button", { name: "Restart", exact: true });
        await expect(restartBtn).toBeEnabled({ timeout: 15000 });

        // Verify we're still on the same page (no crash/redirect)
        await expect(page).toHaveURL(/\/stacks\/01-web-app/);

        // Verify the page is still functional — heading still visible
        await expect(page.getByRole("heading", { name: /01-web-app/, level: 1 })).toBeVisible();
    });
});
