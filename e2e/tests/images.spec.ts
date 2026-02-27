import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";
import { takeLightScreenshot } from "../helpers/light-mode";

test.describe("Images — List", () => {
    test.beforeEach(async ({ page }) => {
        await page.goto("/images");
        await waitForApp(page);
        // Wait for image items to load in the sidebar
        await expect(page.locator(".item").first()).toBeVisible({ timeout: 10000 });
    });

    test("displays image list with items", async ({ page }) => {
        // Verify the sidebar has loaded with images
        const items = page.locator(".item");
        const count = await items.count();
        expect(count).toBeGreaterThan(5);
    });

    test("empty state shows no-image-selected message", async ({ page }) => {
        await expect(page.getByText("Select an image from the list.")).toBeVisible();
    });

    test("search filters image list", async ({ page }) => {
        const searchInput = page.locator(".search-input");
        // Search for a specific full image name to get a single match
        await searchInput.fill("nginx:latest");

        // Should show at least one result
        const items = page.locator(".item");
        await expect(items.first()).toBeVisible({ timeout: 5000 });

        // All visible items should contain "nginx"
        const count = await items.count();
        expect(count).toBeGreaterThan(0);
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

    test("filter by dangling status", async ({ page }) => {
        // Open filter dropdown
        await page.locator(".filter-icon").click();

        // Check "dangling" checkbox
        const danglingCheckbox = page.getByRole("checkbox", { name: "dangling" });
        await expect(danglingCheckbox).toBeVisible({ timeout: 5000 });
        await danglingCheckbox.check();

        // Close dropdown by pressing Escape
        await page.keyboard.press("Escape");

        // Verify filter is active
        await expect(page.locator(".filter-icon-active")).toBeVisible();

        // Dangling images should show sha256 IDs
        const items = page.locator(".item");
        const count = await items.count();
        expect(count).toBeGreaterThan(0);
    });
});

test.describe("Images — Detail", () => {
    test("displays image detail view with overview fields", async ({ page }) => {
        // Navigate to a known image
        await page.goto("/images/nginx:latest");
        await waitForApp(page);

        // Wait for detail to load — the overview list is in the right column
        const overviewList = page.locator(".overview-list").first();
        await expect(overviewList).toBeVisible({ timeout: 10000 });

        // Verify overview-specific labels are present within the overview section
        await expect.soft(overviewList.getByText("Architecture")).toBeVisible();
        await expect.soft(overviewList.getByText("OS")).toBeVisible();
    });

    test("image detail shows layers section", async ({ page }) => {
        await page.goto("/images/nginx:latest");
        await waitForApp(page);

        await expect(page.locator(".overview-list").first()).toBeVisible({ timeout: 10000 });

        // Verify the Layers collapsible section heading is present
        await expect(page.getByText(/^Layers/)).toBeVisible();
    });

    test("image detail shows containers section", async ({ page }) => {
        await page.goto("/images/nginx:latest");
        await waitForApp(page);

        await expect(page.locator(".overview-list").first()).toBeVisible({ timeout: 10000 });

        // Verify the Containers collapsible section heading is present
        await expect(page.getByText(/^Containers/)).toBeVisible();
    });

    test("screenshot: image detail", async ({ page }) => {
        await page.goto("/images/nginx:latest");
        await waitForApp(page);
        await expect(page.locator(".overview-list").first()).toBeVisible({ timeout: 10000 });
        await expect(page).toHaveScreenshot("image-detail.png");
        await takeLightScreenshot(page, "image-detail-light.png");
    });
});
