import { Page, expect } from "@playwright/test";

/**
 * Waits until the app is fully loaded: WebSocket connected, authenticated, and UI rendered.
 * Detects this by waiting for the "Home" nav link to appear in the header.
 */
export async function waitForApp(page: Page): Promise<void> {
    await expect(page.getByRole("link", { name: "Home" })).toBeVisible({ timeout: 15000 });
}
