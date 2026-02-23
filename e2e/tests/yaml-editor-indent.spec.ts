import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";

// Use evaluate(el.click()) instead of Playwright's .click() for the Edit
// button to avoid scroll interference from concurrent WebSocket events.
function clickEdit(page: import("@playwright/test").Page) {
    return page.getByRole("button", { name: "Edit" }).evaluate((el: HTMLElement) => el.click());
}

/**
 * Type Enter then a marker character in the CodeMirror editor, read back the
 * indentation of the new line, then undo all changes. Returns the number of
 * leading spaces before the marker.
 */
async function measureIndentAfterEnter(page: import("@playwright/test").Page, lineIndex: number): Promise<number> {
    const lines = page.locator(".cm-line");
    await lines.nth(lineIndex).click();
    await page.keyboard.press("End");
    await page.keyboard.press("Enter");
    await page.keyboard.type("X");
    // Read back the newly inserted line (it is right after the clicked line)
    const newLine = await lines.nth(lineIndex + 1).textContent();
    const spaces = newLine ? newLine.indexOf("X") : -1;

    // Undo the marker character, newline, and any auto-indent
    await page.keyboard.press("Control+z");
    await page.keyboard.press("Control+z");
    await page.keyboard.press("Control+z");

    return spaces;
}

test.describe("YAML Editor — Indentation", () => {
    test.beforeEach(async ({ page }) => {
        await page.goto("/stacks/01-web-app");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: "01-web-app" })).toBeVisible({ timeout: 15000 });
        await clickEdit(page);
        await expect(page.locator(".editor-box.edit-mode").first()).toBeVisible();
    });

    test("auto-indents 2 spaces after a mapping key (services:)", async ({ page }) => {
        // Line 0: "services:" — pressing Enter should indent to 2 spaces
        const spaces = await measureIndentAfterEnter(page, 0);
        expect(spaces).toBe(2);
    });

    test("auto-indents 2 deeper after a nested mapping key (nginx:)", async ({ page }) => {
        // Line 1: "  nginx:" — pressing Enter should indent to 4 spaces
        const spaces = await measureIndentAfterEnter(page, 1);
        expect(spaces).toBe(4);
    });

    test("maintains indent level after a value line (image: nginx:latest)", async ({ page }) => {
        // Line 2: "    image: nginx:latest" — pressing Enter should stay at 4 spaces
        const spaces = await measureIndentAfterEnter(page, 2);
        expect(spaces).toBe(4);
    });

    test("auto-indents after a list parent key (ports:)", async ({ page }) => {
        // Line 4: "    ports:" — pressing Enter should indent to 6 spaces
        const spaces = await measureIndentAfterEnter(page, 4);
        expect(spaces).toBe(6);
    });

    test("maintains indent after a list item (- 8080:80)", async ({ page }) => {
        // Line 5: "      - 8080:80" — pressing Enter should stay at 6 spaces
        const spaces = await measureIndentAfterEnter(page, 5);
        expect(spaces).toBe(6);
    });

    test("Tab key indents by 2 spaces", async ({ page }) => {
        const lines = page.locator(".cm-line");
        // Line 2: "    image: nginx:latest" (4 spaces indent)
        await lines.nth(2).click();
        await page.keyboard.press("Home");
        await page.keyboard.press("Tab");
        const afterTab = await lines.nth(2).textContent();
        const indent = afterTab ? afterTab.match(/^\s*/)?.[0].length : -1;
        expect(indent).toBe(6);

        // Undo
        await page.keyboard.press("Control+z");
    });

    test("Shift+Tab dedents by 2 spaces", async ({ page }) => {
        const lines = page.locator(".cm-line");
        // Line 2: "    image: nginx:latest" (4 spaces indent)
        await lines.nth(2).click();
        await page.keyboard.press("Home");
        await page.keyboard.press("Shift+Tab");
        const afterShiftTab = await lines.nth(2).textContent();
        const indent = afterShiftTab ? afterShiftTab.match(/^\s*/)?.[0].length : -1;
        expect(indent).toBe(2);

        // Undo
        await page.keyboard.press("Control+z");
    });
});
