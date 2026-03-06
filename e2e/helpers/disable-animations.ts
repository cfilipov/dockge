import { Page } from "@playwright/test";

/**
 * Injects a <style> tag that disables all CSS transitions and animations,
 * and patches Element.prototype.scrollTo to force instant scrolling.
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
            html {
                scroll-behavior: auto !important;
            }
        `,
    });

    // Patch scrollTo to force instant behavior — smooth scroll animations
    // cause race conditions with rapid list reorders in tests.
    await page.evaluate(() => {
        const origScrollTo = Element.prototype.scrollTo;
        Element.prototype.scrollTo = function (...args: any[]) {
            if (args.length === 1 && typeof args[0] === "object" && args[0] !== null) {
                args[0] = { ...args[0], behavior: "instant" };
            }
            return origScrollTo.apply(this, args);
        };
    });
}
