import { test as base, Page } from "@playwright/test";
import { disableAnimations } from "../helpers/disable-animations";

/**
 * Extended test fixture that automatically disables CSS animations on every page load.
 * Use this instead of importing `test` from @playwright/test in spec files.
 */
export const test = base.extend<{ page: Page }>({
    page: async ({ page }, use) => {
        // Disable animations on initial load and after every navigation
        page.on("load", async () => {
            try {
                await disableAnimations(page);
            } catch {
                // Page might have navigated away; ignore
            }
        });
        await use(page);
    },
});

export { expect } from "@playwright/test";
