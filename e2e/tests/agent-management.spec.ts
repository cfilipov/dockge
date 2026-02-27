import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";

test.describe("Agent Management", () => {
    // Clean up any agents added during tests so BoltDB state doesn't leak
    test.afterAll(async ({ request }) => {
        await request.post("/api/mock/reset");
    });

    test.beforeEach(async ({ page }) => {
        await page.goto("/");
        await waitForApp(page);
        // Wait for the agent section to load
        await expect(page.getByText("Dockge Agent")).toBeVisible({ timeout: 10000 });
    });

    test("agent section heading and add button visible", async ({ page }) => {
        // The "Dockge Agents" heading with beta badge should be visible
        await expect(page.getByRole("heading", { name: /Dockge Agent/i, level: 4 })).toBeVisible();
        await expect(page.getByRole("button", { name: "Add Agent" })).toBeVisible();
    });

    test("add agent form opens on button click", async ({ page }) => {
        const addBtn = page.getByRole("button", { name: "Add Agent" });
        await addBtn.click();

        // Verify form fields are visible
        await expect(page.getByLabel(/Dockge URL/i)).toBeVisible();
        await expect(page.getByLabel("Username")).toBeVisible();
        await expect(page.getByLabel("Password")).toBeVisible();
        await expect(page.getByLabel("Friendly Name")).toBeVisible();
        await expect(page.getByRole("button", { name: "Connect" })).toBeVisible();
    });

    test("add agent submission succeeds and form hides", async ({ page }) => {
        // Open form
        await page.getByRole("button", { name: "Add Agent" }).click();

        // Fill in the form
        await page.getByLabel(/Dockge URL/i).fill("http://192.168.1.100:5001");
        await page.getByLabel("Username").fill("admin");
        await page.getByLabel("Password").fill("testpass");
        await page.getByLabel("Friendly Name").fill("Test Agent");

        // Submit
        await page.getByRole("button", { name: "Connect" }).click();

        // Form should hide on success and Add Agent button should reappear
        await expect(page.getByRole("button", { name: "Add Agent" })).toBeVisible({ timeout: 10000 });
        // The form fields should no longer be visible
        await expect(page.getByLabel(/Dockge URL/i)).not.toBeVisible();
    });
});
