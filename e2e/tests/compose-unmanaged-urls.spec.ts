import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";
import { takeLightScreenshot } from "../helpers/light-mode";

test.describe("Compose — Unmanaged Stack Banner", () => {
    test("shows unmanaged banner on 10-unmanaged", async ({ page }) => {
        await page.goto("/stacks/10-unmanaged");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: /10-unmanaged/, level: 1 })).toBeVisible({ timeout: 15000 });

        // Verify the unmanaged banner is shown
        await expect(page.getByText("This stack is not managed by Dockge.")).toBeVisible({ timeout: 10000 });
    });

    test("unmanaged stack does not show Edit button", async ({ page }) => {
        await page.goto("/stacks/10-unmanaged");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: /10-unmanaged/, level: 1 })).toBeVisible({ timeout: 15000 });

        // Edit button should NOT be visible for unmanaged stacks
        await expect(page.getByRole("button", { name: "Edit", exact: true })).not.toBeVisible();
    });

    test("screenshot: unmanaged stack banner", async ({ page }) => {
        await page.goto("/stacks/10-unmanaged");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: /10-unmanaged/, level: 1 })).toBeVisible({ timeout: 15000 });
        await expect(page.getByText("This stack is not managed by Dockge.")).toBeVisible({ timeout: 10000 });
        // Scroll to top to keep the dynamic log terminal (with varying timestamps) out of the viewport
        await page.evaluate(() => window.scrollTo(0, 0));
        await expect(page).toHaveScreenshot("compose-unmanaged.png");
        await takeLightScreenshot(page, "compose-unmanaged-light.png");
    });
});

test.describe("Compose — URL Display", () => {
    test("shows URL badges on stack with URL labels", async ({ page }) => {
        await page.goto("/stacks/07-full-features");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: /07-full-features/, level: 1 })).toBeVisible({ timeout: 15000 });

        // 07-full-features has URL labels from dockge.urls.* compose labels
        // Wait for the URL badges to appear (they're rendered as <a> tags with .badge.bg-secondary)
        const urlBadges = page.locator("a .badge.bg-secondary");
        await expect(urlBadges.first()).toBeVisible({ timeout: 10000 });

        // Verify at least one known URL badge is displayed
        // The URLs are rendered as <a> tags wrapping <span class="badge">
        // Use .first() since the same URL may also appear as a port chip link
        await expect(page.locator("a[href='http://localhost'] .badge").first()).toBeVisible();
        await expect(page.locator("a[href='http://localhost:8080'] .badge").first()).toBeVisible();
    });

    test("screenshot: compose view with URLs", async ({ page }) => {
        await page.goto("/stacks/07-full-features");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: /07-full-features/, level: 1 })).toBeVisible({ timeout: 15000 });
        await expect(page.locator(".cm-editor").first()).toBeVisible({ timeout: 10000 });
        await expect(page).toHaveScreenshot("compose-urls.png");
        await takeLightScreenshot(page, "compose-urls-light.png");
    });
});
