import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";
import { Locator } from "@playwright/test";

function evalClick(locator: Locator) {
    return locator.evaluate((el: HTMLElement) => el.click());
}

test.describe("Container Card — Delete Service", () => {

    test.beforeAll(async ({ request }) => {
        await request.post("/api/mock/reset");
    });
    test.afterAll(async ({ request }) => {
        await request.post("/api/mock/reset");
    });

    test("deleting a container card removes the service from YAML", async ({ page }) => {
        // Use stack-013 (a filler stack) to avoid interfering with other tests
        await page.goto("/stacks/stack-013");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: /stack-013/, level: 1 })).toBeVisible({ timeout: 15000 });

        // The YAML editor
        const yamlEditor = page.locator(".cm-content").first();
        await expect(yamlEditor).toBeVisible({ timeout: 10000 });

        // Verify the stack has a service (filler stacks have a single "alpine" service)
        await expect(yamlEditor).toContainText("alpine");

        // Enter page-level edit mode
        await evalClick(page.getByRole("button", { name: "Edit", exact: true }));
        await expect(page.getByRole("button", { name: "Deploy" })).toBeVisible();

        // Find the "alpine" service container card
        const serviceCard = page.getByRole("region", { name: "alpine" });
        await expect(serviceCard).toBeVisible();

        // The Delete button is directly visible in edit mode (no config expansion needed)
        const deleteBtn = serviceCard.getByRole("button", { name: "Delete" });
        await expect(deleteBtn).toBeVisible();

        // Click Delete — this removes the service from jsonConfig immediately
        await evalClick(deleteBtn);

        // The container card should disappear
        await expect(serviceCard).not.toBeVisible({ timeout: 5000 });

        // The YAML should no longer contain the "alpine" service definition
        await expect(yamlEditor).not.toContainText("alpine:");
    });
});
