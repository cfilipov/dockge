import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";

test.describe("Stack List — Search", () => {
    test.beforeEach(async ({ page }) => {
        await page.goto("/");
        await waitForApp(page);
        // Wait for stack items to load
        await expect(page.locator(".item").first()).toBeVisible({ timeout: 10000 });
    });

    test("search filters stacks by name", async ({ page }) => {
        const searchInput = page.locator(".search-input");
        await expect(searchInput).toBeVisible();

        // Type a search query that matches a specific stack
        await searchInput.fill("01-web-app");

        // Verify only matching stacks are visible
        const items = page.locator(".item");
        await expect(items).toHaveCount(1, { timeout: 5000 });
        await expect(items.first()).toContainText("01-web-app");
    });

    test("clearing search restores full list", async ({ page }) => {
        const searchInput = page.locator(".search-input");
        await searchInput.fill("01-web-app");
        await expect(page.locator(".item")).toHaveCount(1, { timeout: 5000 });

        // Clear the search
        await searchInput.fill("");

        // Verify full list is restored (should have many stacks)
        const itemCount = await page.locator(".item").count();
        expect(itemCount).toBeGreaterThan(10);
    });

    test("search with no matches shows empty state", async ({ page }) => {
        const searchInput = page.locator(".search-input");
        await searchInput.fill("nonexistent-stack-xyz-999");

        // Verify no items are visible
        await expect(page.locator(".item")).toHaveCount(0, { timeout: 5000 });
    });
});

test.describe("Stack List — Filter by Status", () => {
    test.beforeEach(async ({ page }) => {
        await page.goto("/");
        await waitForApp(page);
        await expect(page.locator(".item").first()).toBeVisible({ timeout: 10000 });
    });

    test("filter by exited status shows only exited stacks", async ({ page }) => {
        // Open the filter dropdown
        await page.locator(".filter-icon").click();

        // Click the "exited" checkbox
        const exitedCheckbox = page.getByRole("checkbox", { name: "exited" });
        await expect(exitedCheckbox).toBeVisible({ timeout: 5000 });
        await exitedCheckbox.check();

        // Close dropdown by pressing Escape
        await page.keyboard.press("Escape");

        // Verify filter icon is active (purple)
        await expect(page.locator(".filter-icon-active")).toBeVisible();

        // All visible items should be exited stacks
        const items = page.locator(".item");
        const count = await items.count();
        expect(count).toBeGreaterThan(0);

        // Verify at least one known exited stack is present (03-monitoring is exited)
        await expect(page.locator(".item").filter({ hasText: "03-monitoring" })).toBeVisible();
    });

    test("filter by active status shows only running stacks", async ({ page }) => {
        // Open the filter dropdown
        await page.locator(".filter-icon").click();

        // The "active" checkbox — use first() since there may be multiple filter dropdowns
        const activeCheckbox = page.getByRole("checkbox", { name: "active" }).first();
        await expect(activeCheckbox).toBeVisible({ timeout: 5000 });
        await activeCheckbox.check();

        // Close dropdown by pressing Escape
        await page.keyboard.press("Escape");

        // Verify known running stacks are present
        await expect(page.locator(".item").filter({ hasText: "01-web-app" })).toBeVisible();

        // Verify known exited stack is NOT visible
        await expect(page.locator(".item").filter({ hasText: "03-monitoring" })).not.toBeVisible();
    });

    test("clear filter restores full list", async ({ page }) => {
        // Apply a filter first
        await page.locator(".filter-icon").click();
        await page.getByRole("checkbox", { name: "exited" }).check();
        await page.keyboard.press("Escape");

        // Verify filter is active
        await expect(page.locator(".filter-icon-active")).toBeVisible();
        const filteredCount = await page.locator(".item").count();

        // Re-open filter dropdown and click "Clear Filter"
        await page.locator(".filter-icon").click();
        await page.getByText("Clear Filter").click();

        // Verify filter icon is no longer active
        await expect(page.locator(".filter-icon-active")).not.toBeVisible();

        // Verify more items are now shown
        const fullCount = await page.locator(".item").count();
        expect(fullCount).toBeGreaterThan(filteredCount);
    });
});

test.describe("Stack List — Filter by Attributes", () => {
    test.beforeEach(async ({ page }) => {
        await page.goto("/");
        await waitForApp(page);
        await expect(page.locator(".item").first()).toBeVisible({ timeout: 10000 });
    });

    test("filter by unmanaged attribute shows unmanaged stacks", async ({ page }) => {
        // Open the filter dropdown
        await page.locator(".filter-icon").click();

        // Click the "unmanaged" checkbox
        const unmanagedCheckbox = page.getByRole("checkbox", { name: "unmanaged" });
        await expect(unmanagedCheckbox).toBeVisible({ timeout: 5000 });
        await unmanagedCheckbox.check();

        // Close dropdown
        await page.keyboard.press("Escape");

        // Verify the unmanaged stack appears
        await expect(page.locator(".item").filter({ hasText: "10-unmanaged" })).toBeVisible({ timeout: 5000 });

        // Verify managed stacks are hidden
        await expect(page.locator(".item").filter({ hasText: "01-web-app" })).not.toBeVisible();
    });
});

test.describe("Stack List — URL Query Sync", () => {
    test("navigating with query params pre-fills search and filter", async ({ page }) => {
        await page.goto("/?q=web-app&status=active");
        await waitForApp(page);
        await expect(page.locator(".item").first()).toBeVisible({ timeout: 10000 });

        // Verify search input is pre-filled
        const searchInput = page.locator(".search-input");
        await expect(searchInput).toHaveValue("web-app");

        // Verify filtered results match
        await expect(page.locator(".item").filter({ hasText: "01-web-app" })).toBeVisible();
    });
});
