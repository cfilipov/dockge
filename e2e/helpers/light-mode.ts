import { Page, expect } from "@playwright/test";

/**
 * Takes a screenshot in light mode by toggling Playwright's emulated color scheme.
 * Requires localStorage.theme = "auto" (set in auth.setup.ts) so the Vue useTheme
 * composable follows the system preference via matchMedia.
 *
 * After the screenshot, restores dark mode so subsequent assertions are unaffected.
 */
export async function takeLightScreenshot(page: Page, name: string) {
    await page.emulateMedia({ colorScheme: "light" });
    await page.waitForFunction(() => document.body.classList.contains("light"));
    await expect(page).toHaveScreenshot(name);
    await page.emulateMedia({ colorScheme: "dark" });
    await page.waitForFunction(() => document.body.classList.contains("dark"));
}
