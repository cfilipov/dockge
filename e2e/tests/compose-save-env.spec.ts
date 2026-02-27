import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";
import { Locator } from "@playwright/test";

function evalClick(locator: Locator) {
    return locator.evaluate((el: HTMLElement) => el.click());
}

test.describe("Compose — Save .env File", () => {

    test.beforeAll(async ({ request }) => {
        await request.post("/api/mock/reset");
    });
    test.afterAll(async ({ request }) => {
        await request.post("/api/mock/reset");
    });

    test("save persists .env file changes", async ({ page }) => {
        // Use a filler stack that won't interfere with other tests
        await page.goto("/stacks/stack-014");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: /stack-014/, level: 1 })).toBeVisible({ timeout: 15000 });

        // Enter edit mode
        await evalClick(page.getByRole("button", { name: "Edit", exact: true }));
        await expect(page.getByRole("button", { name: "Save" })).toBeVisible();

        // The .env editor section appears in edit mode with aria-label=".env"
        const envRegion = page.getByRole("region", { name: ".env" });
        await expect(envRegion).toBeVisible({ timeout: 10000 });

        // Find the CodeMirror editor inside the .env region
        const envEditor = envRegion.locator(".cm-content");
        await expect(envEditor).toBeVisible();

        // Click into the env editor and type a key-value pair
        await envEditor.click();
        await page.keyboard.type("MY_TEST_VAR=hello123");

        // Save the stack (page-level Save button saves YAML + ENV + override together)
        await evalClick(page.getByRole("button", { name: "Save" }));
        await expect(page.getByText("Saved")).toBeVisible({ timeout: 5000 });

        // Reload and verify the .env content persisted
        await page.reload();
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: /stack-014/, level: 1 })).toBeVisible({ timeout: 15000 });

        // Enter edit mode again to see the .env editor
        await evalClick(page.getByRole("button", { name: "Edit", exact: true }));
        await expect(page.getByRole("button", { name: "Save" })).toBeVisible();

        // Verify the .env content persisted
        const reloadedEnvRegion = page.getByRole("region", { name: ".env" });
        await expect(reloadedEnvRegion).toBeVisible({ timeout: 10000 });
        const reloadedEnvEditor = reloadedEnvRegion.locator(".cm-content");
        await expect(reloadedEnvEditor).toContainText("MY_TEST_VAR=hello123");
    });
});
