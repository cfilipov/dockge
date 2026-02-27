import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";
import { takeLightScreenshot } from "../helpers/light-mode";

test.describe("Networks — List", () => {
    test.beforeEach(async ({ page }) => {
        await page.goto("/networks");
        await waitForApp(page);
        // Wait for network items to load in the sidebar
        await expect(page.locator(".item").first()).toBeVisible({ timeout: 10000 });
    });

    test("displays network list with expected networks", async ({ page }) => {
        // Verify well-known networks exist: bridge, host, none, proxy
        await expect.soft(page.locator(".item").filter({ hasText: "bridge" }).first()).toBeVisible();
        await expect.soft(page.locator(".item").filter({ hasText: "host" }).first()).toBeVisible();
        await expect.soft(page.locator(".item").filter({ hasText: "none" }).first()).toBeVisible();
        await expect.soft(page.locator(".item").filter({ hasText: "proxy" }).first()).toBeVisible();
    });

    test("empty state shows no-network-selected message", async ({ page }) => {
        // On /networks with no network selected, verify empty state
        await expect(page.getByText("Select a network from the list.")).toBeVisible();
    });

    test("search filters network list", async ({ page }) => {
        const searchInput = page.getByPlaceholder("Search...");
        await expect(searchInput).toBeVisible();

        await searchInput.fill("proxy");
        // Should show proxy network
        await expect(page.locator(".item").filter({ hasText: "proxy" }).first()).toBeVisible({ timeout: 5000 });
    });

    test("filter by in use status", async ({ page }) => {
        // Open filter dropdown
        await page.getByRole("button", { name: "Filter" }).click();

        // Check "in use" checkbox
        const inUseCheckbox = page.getByRole("checkbox", { name: "in use" });
        await expect(inUseCheckbox).toBeVisible({ timeout: 5000 });
        await inUseCheckbox.check();

        // Close dropdown by pressing Escape
        await page.keyboard.press("Escape");

        // Verify filter is active
        await expect(page.locator(".filter-icon-active")).toBeVisible();

        // In-use networks should be visible
        const items = page.locator(".item");
        const count = await items.count();
        expect(count).toBeGreaterThan(0);
    });
});

test.describe("Networks — Detail", () => {
    // Reset mock state to ensure consistent containers connected to networks.
    test.beforeAll(async ({ request }) => {
        await request.post("/api/mock/reset");
    });

    test("displays network detail view", async ({ page }) => {
        await page.goto("/networks/bridge");
        await waitForApp(page);

        // Wait for detail to load — the overview list is in the right column
        const overviewList = page.getByRole("region", { name: "Overview" });
        await expect(overviewList).toBeVisible({ timeout: 10000 });

        // Verify overview-specific labels within the overview section
        await expect.soft(overviewList.getByText("Driver")).toBeVisible();
        await expect.soft(overviewList.getByText("Scope")).toBeVisible();
    });

    test("network detail shows connected containers section", async ({ page }) => {
        // proxy network should have connected containers
        await page.goto("/networks/proxy");
        await waitForApp(page);

        // Wait for the overview to load
        await expect(page.getByRole("region", { name: "Overview" })).toBeVisible({ timeout: 10000 });

        // Verify the Containers collapsible section heading is present
        await expect(page.getByText(/^Containers/)).toBeVisible();
    });

    test("screenshot: network detail", async ({ page }) => {
        await page.goto("/networks/bridge");
        await waitForApp(page);
        await expect(page.getByRole("region", { name: "Overview" })).toBeVisible({ timeout: 10000 });
        // Wait for the Containers section to fully load
        await expect(page.getByText(/^Containers/)).toBeVisible({ timeout: 10000 });
        await expect(page).toHaveScreenshot("network-detail.png");
        await takeLightScreenshot(page, "network-detail-light.png");
    });
});
