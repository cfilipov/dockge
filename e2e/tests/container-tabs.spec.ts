import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";
import { takeLightScreenshot } from "../helpers/light-mode";

/**
 * Tests for the Containers sub-view navigation system.
 *
 * Logs and Shell are now sub-views of the Containers tab, not separate nav tabs.
 * The view mode toggle (parsed/raw/logs/shell) controls which sub-view is shown.
 *
 * Covers:
 * 1. Clicking container card buttons in stacks view navigates to the correct sub-view
 * 2. Sidebar container selection switches the detail view and preserves sub-view
 * 3. Sub-view toggle switches between parsed/raw/logs/shell
 * 4. Clicking the Containers tab while already on it resets to parsed and clears selection
 */

test.describe("Container Tabs — Stacks to Logs", () => {
    test("log button navigates to Containers tab with logs sub-view", async ({ page }) => {
        await page.goto("/stacks/01-web-app");
        await waitForApp(page);

        const logLink = page.getByRole("link", { name: "docker compose logs nginx" });
        await expect(logLink).toBeVisible({ timeout: 10000 });
        await logLink.click();

        // URL and heading
        await expect(page).toHaveURL("/containers/01-web-app-nginx-1/logs");
        await expect(page.getByRole("heading", { name: /running\s+01-web-app-nginx-1/i })).toBeVisible({ timeout: 10000 });

        // Containers nav tab is active
        const containersNav = page.getByRole("link", { name: "Containers" }).first();
        await expect(containersNav).toHaveClass(/active/);

        // Logs sub-view toggle is active
        const logsToggle = page.getByRole("link", { name: "Logs" });
        await expect(logsToggle).toHaveClass(/btn-primary/);

        // Terminal region is visible (log output)
        await expect(page.getByRole("region", { name: "Terminal" })).toBeVisible({ timeout: 10000 });

        // Container sidebar is visible with the item highlighted
        const sidebarItem = page.locator(".item.active", { hasText: "01-web-app-nginx-1" });
        await expect(sidebarItem).toBeVisible();
    });

    test("screenshot: stacks to logs", async ({ page }) => {
        await page.goto("/stacks/01-web-app");
        await waitForApp(page);
        await page.getByRole("link", { name: "docker compose logs nginx" }).click();
        await expect(page.getByRole("region", { name: "Terminal" })).toBeVisible({ timeout: 10000 });
        await expect(page).toHaveScreenshot("container-tabs-stacks-to-logs.png");
        await takeLightScreenshot(page, "container-tabs-stacks-to-logs-light.png");
    });
});

test.describe("Container Tabs — Stacks to Containers", () => {
    test("inspect button navigates to Containers tab with container selected", async ({ page }) => {
        await page.goto("/stacks/01-web-app");
        await waitForApp(page);

        // Click the container name link (in h5 heading) which navigates to inspect
        const inspectLink = page.getByRole("link", { name: /01-web-app-\w+-1/ }).first();
        await expect(inspectLink).toBeVisible({ timeout: 10000 });
        await inspectLink.click();

        // URL and heading
        await expect(page).toHaveURL(/\/containers\/01-web-app-.*-1/);
        await expect(page.getByRole("heading", { name: /running\s+01-web-app-/i })).toBeVisible({ timeout: 10000 });

        // Containers nav tab is active
        const containersNav = page.getByRole("link", { name: "Containers" }).first();
        await expect(containersNav).toHaveClass(/active/);

        // Parsed inspect view is visible
        await expect(page.getByRole("region", { name: "Overview" })).toBeVisible({ timeout: 10000 });

        // Container sidebar is visible with an item highlighted
        await expect(page.locator(".item.active")).toBeVisible();
    });

    test("screenshot: stacks to containers", async ({ page }) => {
        await page.goto("/stacks/01-web-app");
        await waitForApp(page);
        await page.getByRole("link", { name: /01-web-app-\w+-1/ }).first().click();
        await expect(page.getByRole("region", { name: "Overview" })).toBeVisible({ timeout: 10000 });
        await expect(page).toHaveScreenshot("container-tabs-stacks-to-containers.png");
        await takeLightScreenshot(page, "container-tabs-stacks-to-containers-light.png");
    });
});

test.describe("Container Tabs — Stacks to Shell", () => {
    test("shell button navigates to Containers tab with shell sub-view", async ({ page }) => {
        await page.goto("/stacks/01-web-app");
        await waitForApp(page);

        const shellLink = page.getByRole("link", { name: "docker compose exec nginx" });
        await expect(shellLink).toBeVisible({ timeout: 10000 });
        await shellLink.click();

        // URL and heading
        await expect(page).toHaveURL("/containers/01-web-app-nginx-1/shell/bash");
        await expect(page.getByRole("heading", { name: /running\s+01-web-app-nginx-1/i })).toBeVisible({ timeout: 10000 });

        // Containers nav tab is active
        const containersNav = page.getByRole("link", { name: "Containers" }).first();
        await expect(containersNav).toHaveClass(/active/);

        // Shell sub-view toggle is active
        const shellToggle = page.getByRole("link", { name: "Shell" });
        await expect(shellToggle).toHaveClass(/btn-primary/);

        // Terminal region is visible
        await expect(page.getByRole("region", { name: "Terminal" })).toBeVisible({ timeout: 10000 });

        // Switch shell button is visible
        await expect(page.getByRole("link", { name: /Switch to sh/i })).toBeVisible();

        // Container sidebar is visible with the item highlighted
        const sidebarItem = page.locator(".item.active", { hasText: "01-web-app-nginx-1" });
        await expect(sidebarItem).toBeVisible();
    });

    test("screenshot: stacks to shell", async ({ page }) => {
        await page.goto("/stacks/01-web-app");
        await waitForApp(page);
        await page.getByRole("link", { name: "docker compose exec nginx" }).click();
        await expect(page.getByRole("region", { name: "Terminal" })).toBeVisible({ timeout: 10000 });
        await expect(page).toHaveScreenshot("container-tabs-stacks-to-shell.png");
        await takeLightScreenshot(page, "container-tabs-stacks-to-shell-light.png");
    });
});

test.describe("Container Tabs — Sidebar switching", () => {
    test("clicking a different container in the sidebar switches the detail view", async ({ page }) => {
        // Start on logs sub-view with nginx selected
        await page.goto("/containers/01-web-app-nginx-1/logs");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: /running\s+01-web-app-nginx-1/i })).toBeVisible({ timeout: 10000 });

        // Click a different container in the sidebar (redis from the same stack)
        const redisItem = page.locator(".item", { hasText: "01-web-app-redis-1" });
        await expect(redisItem).toBeVisible({ timeout: 10000 });
        await redisItem.click();

        // URL and heading update to the new container, staying on logs sub-view
        await expect(page).toHaveURL("/containers/01-web-app-redis-1/logs");
        await expect(page.getByRole("heading", { name: /exited\s+01-web-app-redis-1/i })).toBeVisible({ timeout: 10000 });

        // The redis item is now active, nginx is not
        await expect(page.locator(".item.active", { hasText: "01-web-app-redis-1" })).toBeVisible();
        await expect(page.locator(".item.active", { hasText: "01-web-app-nginx-1" })).not.toBeVisible();
    });

    test("clicking a different container on containers tab switches inspect view", async ({ page }) => {
        await page.goto("/containers/01-web-app-nginx-1");
        await waitForApp(page);
        await expect(page.getByRole("region", { name: "Overview" })).toBeVisible({ timeout: 10000 });

        // Click redis
        const redisItem = page.locator(".item", { hasText: "01-web-app-redis-1" });
        await expect(redisItem).toBeVisible({ timeout: 10000 });
        await redisItem.click();

        await expect(page).toHaveURL("/containers/01-web-app-redis-1");
        await expect(page.getByRole("heading", { name: /exited\s+01-web-app-redis-1/i })).toBeVisible({ timeout: 10000 });
    });

    test("screenshot: sidebar switch", async ({ page }) => {
        await page.goto("/containers/01-web-app-nginx-1/logs");
        await waitForApp(page);
        await expect(page.getByRole("region", { name: "Terminal" })).toBeVisible({ timeout: 10000 });
        await page.locator(".item", { hasText: "01-web-app-redis-1" }).click();
        await expect(page.getByRole("heading", { name: /exited\s+01-web-app-redis-1/i })).toBeVisible({ timeout: 10000 });
        await expect(page).toHaveScreenshot("container-tabs-sidebar-switch.png");
        await takeLightScreenshot(page, "container-tabs-sidebar-switch-light.png");
    });
});

test.describe("Container Tabs — Sub-view toggle preserves selection", () => {
    test("switching from parsed to logs via toggle preserves container selection", async ({ page }) => {
        await page.goto("/containers/02-blog-mysql-1");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: /running\s+02-blog-mysql-1/i })).toBeVisible({ timeout: 10000 });

        // Click Logs toggle button
        const logsToggle = page.getByRole("link", { name: "Logs" });
        await logsToggle.click();

        // Should navigate to logs for the same container
        await expect(page).toHaveURL("/containers/02-blog-mysql-1/logs");
        await expect(page.getByRole("heading", { name: /running\s+02-blog-mysql-1/i })).toBeVisible({ timeout: 10000 });
    });

    test("switching from logs to shell via toggle preserves container selection", async ({ page }) => {
        await page.goto("/containers/02-blog-mysql-1/logs");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: /running\s+02-blog-mysql-1/i })).toBeVisible({ timeout: 10000 });

        // Click Shell toggle button
        const shellToggle = page.getByRole("link", { name: "Shell" });
        await shellToggle.click();

        await expect(page).toHaveURL("/containers/02-blog-mysql-1/shell");
        await expect(page.getByRole("heading", { name: /running\s+02-blog-mysql-1/i })).toBeVisible({ timeout: 10000 });
    });

    test("switching from shell to parsed via toggle preserves container selection", async ({ page }) => {
        await page.goto("/containers/02-blog-mysql-1/shell");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: /running\s+02-blog-mysql-1/i })).toBeVisible({ timeout: 10000 });

        // Click parsed toggle button (Show UI)
        const parsedToggle = page.getByRole("link", { name: "Show UI" });
        await parsedToggle.click();

        await expect(page).toHaveURL("/containers/02-blog-mysql-1");
        await expect(page.getByRole("heading", { name: /running\s+02-blog-mysql-1/i })).toBeVisible({ timeout: 10000 });
    });

    test("clicking Containers tab while on it clears selection and goes to home view", async ({ page }) => {
        await page.goto("/containers/02-blog-mysql-1/logs");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: /running\s+02-blog-mysql-1/i })).toBeVisible({ timeout: 10000 });

        // Click Containers tab again — should go to /containers (home)
        const containersNav = page.getByRole("link", { name: "Containers" }).first();
        await containersNav.click();

        await expect(page).toHaveURL("/containers");
        // Home view shows "Select a container" text
        await expect(page.getByText("Select a container from the list.")).toBeVisible({ timeout: 10000 });
        // No container should be highlighted in the sidebar
        await expect(page.locator(".item.active")).not.toBeVisible();
    });

    test("screenshot: tab switching preserves selection", async ({ page }) => {
        // Start on Containers tab with a container selected (parsed view)
        await page.goto("/containers/02-blog-mysql-1");
        await waitForApp(page);
        await expect(page.getByRole("region", { name: "Overview" })).toBeVisible({ timeout: 10000 });

        // Switch to Logs sub-view via toggle
        await page.getByRole("link", { name: "Logs" }).click();
        await expect(page.getByRole("heading", { name: /running\s+02-blog-mysql-1/i })).toBeVisible({ timeout: 10000 });
        await expect(page).toHaveScreenshot("container-tabs-preserved-selection.png");
        await takeLightScreenshot(page, "container-tabs-preserved-selection-light.png");
    });
});
