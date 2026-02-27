import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";
import { takeLightScreenshot } from "../helpers/light-mode";

test.describe("Volumes — List", () => {
    test.beforeEach(async ({ page }) => {
        await page.goto("/volumes");
        await waitForApp(page);
        // Wait for volume items to load in the sidebar
        await expect(page.locator(".item").first()).toBeVisible({ timeout: 10000 });
    });

    test("displays volume list with volumes", async ({ page }) => {
        const items = page.locator(".item");
        const count = await items.count();
        expect(count).toBeGreaterThan(0);
    });

    test("empty state shows no-volume-selected message", async ({ page }) => {
        await expect(page.getByText("Select a volume from the list.")).toBeVisible();
    });

    test("search filters volume list", async ({ page }) => {
        const searchInput = page.locator(".search-input");
        // Search for a known volume name pattern
        await searchInput.fill("pgdata");

        // Should show results containing pgdata
        const items = page.locator(".item");
        await expect(items.first()).toBeVisible({ timeout: 5000 });

        // All visible items should contain "pgdata"
        const count = await items.count();
        expect(count).toBeGreaterThan(0);
        for (let i = 0; i < count; i++) {
            await expect(items.nth(i)).toContainText("pgdata");
        }
    });

    test("filter by in use status", async ({ page }) => {
        // Open filter dropdown
        await page.locator(".filter-icon").click();

        // Check "in use" checkbox
        const inUseCheckbox = page.getByRole("checkbox", { name: "in use" });
        await expect(inUseCheckbox).toBeVisible({ timeout: 5000 });
        await inUseCheckbox.check();

        // Close dropdown by pressing Escape
        await page.keyboard.press("Escape");

        // Verify filter is active
        await expect(page.locator(".filter-icon-active")).toBeVisible();

        const items = page.locator(".item");
        const count = await items.count();
        expect(count).toBeGreaterThan(0);
    });
});

test.describe("Volumes — Detail", () => {
    test("displays volume detail view with overview fields", async ({ page }) => {
        await page.goto("/volumes/04-database_pgdata");
        await waitForApp(page);

        // Wait for detail to load — the overview list is in the right column
        const overviewList = page.locator(".overview-list").first();
        await expect(overviewList).toBeVisible({ timeout: 10000 });

        // Verify overview-specific labels within the overview section
        await expect.soft(overviewList.getByText("Driver")).toBeVisible();
        await expect.soft(overviewList.getByText("Scope")).toBeVisible();
        await expect.soft(overviewList.getByText("Mountpoint")).toBeVisible();
    });

    test("volume detail shows containers section", async ({ page }) => {
        await page.goto("/volumes/04-database_pgdata");
        await waitForApp(page);

        await expect(page.locator(".overview-list").first()).toBeVisible({ timeout: 10000 });

        // Verify Containers collapsible section heading is present
        await expect(page.getByText(/^Containers/)).toBeVisible();
    });

    test("screenshot: volume detail", async ({ page }) => {
        await page.goto("/volumes/04-database_pgdata");
        await waitForApp(page);
        await expect(page.locator(".overview-list").first()).toBeVisible({ timeout: 10000 });
        await expect(page).toHaveScreenshot("volume-detail.png");
        await takeLightScreenshot(page, "volume-detail-light.png");
    });
});
