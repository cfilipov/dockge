import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";

test.describe("Container Metrics", () => {
    test.beforeEach(async ({ page }) => {
        await page.goto("/containers/01-web-app-nginx-1");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: /running\s+01-web-app-nginx-1/i })).toBeVisible({ timeout: 10000 });
    });

    test("displays CPU, Memory, Block I/O, and Network I/O metrics", async ({ page }) => {
        // The metrics card has four columns: CPU, Memory, Block I/O, Network I/O.
        // Stats arrive via WebSocket polling every 5 seconds; wait for actual values.
        const metricsCard = page.locator(".metric-cell");

        // Wait for stats to load — CPU value should not be a placeholder "--"
        const cpuNum = metricsCard.first().locator(".num").first();
        await expect(cpuNum).not.toHaveClass(/placeholder-value/, { timeout: 10000 });

        // Verify all four metric labels are visible
        await expect(page.getByText("CPU")).toBeVisible();
        await expect(page.getByText("Memory")).toBeVisible();
        await expect(page.getByText("Block I/O")).toBeVisible();
        await expect(page.getByText("Network I/O")).toBeVisible();

        // Verify metric sub-labels
        await expect(page.locator(".num-tag", { hasText: "usage" }).first()).toBeVisible();
        await expect(page.locator(".num-tag", { hasText: "used" }).first()).toBeVisible();
        await expect(page.locator(".num-tag", { hasText: "avail." }).first()).toBeVisible();
        await expect(page.locator(".num-tag", { hasText: "read" }).first()).toBeVisible();
        await expect(page.locator(".num-tag", { hasText: "write" }).first()).toBeVisible();
        await expect(page.locator(".num-tag", { hasText: "rx" }).first()).toBeVisible();
        await expect(page.locator(".num-tag", { hasText: "tx" }).first()).toBeVisible();

        // Verify numeric values are present (not "--" placeholders)
        // CPU should show a percentage like "0.05" with unit "%"
        const cpuUnit = metricsCard.first().locator(".num-unit").first();
        await expect(cpuUnit).toHaveText("%");

        // Memory should show values with units (MiB or GiB)
        const memUnit = metricsCard.nth(1).locator(".num-unit").first();
        await expect(memUnit).toHaveText(/MiB|GiB/);
    });
});
