import { Page } from "@playwright/test";

/**
 * Injects a <style> tag that disables all CSS transitions and animations.
 * Call this after every page navigation for deterministic screenshots.
 */
export async function disableAnimations(page: Page): Promise<void> {
    await page.addStyleTag({
        content: `
            *, *::before, *::after {
                animation-duration: 0s !important;
                animation-delay: 0s !important;
                transition-duration: 0s !important;
                transition-delay: 0s !important;
            }
        `,
    });
}
