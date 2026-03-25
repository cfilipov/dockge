import { Page, expect } from "@playwright/test";

/**
 * Waits until the app is fully loaded: WebSocket connected, authenticated,
 * and all 6 initial data channels received (stacks, containers, networks,
 * images, volumes, updates).  The frontend sets data-ready="true" on
 * document.body once every channel has arrived at least once.
 */
export async function waitForApp(page: Page): Promise<void> {
    await expect(page.getByRole("link", { name: "Stacks" })).toBeVisible({ timeout: 15000 });
    await page.waitForFunction(() => document.body.dataset.ready === "true", null, { timeout: 15000 });
}
