import { test, expect } from "../fixtures/auth.fixture";

test.describe("Performance", () => {
    test("stack list populates within 1 second", async ({ page }) => {
        const start = Date.now();

        await page.goto("/");

        // Wait for all 202 stacks to appear in the sidebar
        // (201 managed stack directories + 1 unmanaged stack "10-unmanaged")
        const stackItems = page.locator(".stack-list .item");
        await expect(stackItems).toHaveCount(202, { timeout: 10000 });

        const elapsed = Date.now() - start;

        // eslint-disable-next-line no-console
        console.log(`Stack list populated in ${elapsed}ms (202 stacks)`);
        expect(elapsed, `Stack list took ${elapsed}ms, expected <1000ms`).toBeLessThan(1000);
    });

    test("navigating to a stack loads within 1 second", async ({ page }) => {
        await page.goto("/");
        // Wait for app to be ready
        await expect(page.getByRole("link", { name: "Stacks" })).toBeVisible({ timeout: 15000 });

        const start = Date.now();

        await page.goto("/stacks/01-web-app");
        await expect(page.getByRole("heading", { name: /01-web-app/, level: 1 })).toBeVisible({ timeout: 10000 });

        const elapsed = Date.now() - start;

        // eslint-disable-next-line no-console
        console.log(`Stack page loaded in ${elapsed}ms`);
        expect(elapsed, `Stack page took ${elapsed}ms, expected <1000ms`).toBeLessThan(1000);
    });
});
