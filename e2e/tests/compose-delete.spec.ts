import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";

test.describe("Delete Stack", () => {

    // Reset mock state before and after since delete modifies state.
    test.beforeAll(async ({ request }) => {
        await request.post("/api/mock/reset");
    });
    test.afterAll(async ({ request }) => {
        await request.post("/api/mock/reset");
    });

    test("delete stack via overflow menu with confirmation modal", async ({ page }) => {
        // Use a filler stack that won't affect other tests
        await page.goto("/stacks/stack-015");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: /stack-015/, level: 1 })).toBeVisible({ timeout: 15000 });

        // Open the overflow dropdown menu
        await page.locator(".btn-group .dropdown-toggle").last().click();

        // Click "Delete" in the dropdown menu (scope to the dropdown that's now visible)
        await page.locator(".dropdown-menu.show").getByText("Delete", { exact: true }).click();

        // Verify confirmation modal appears
        const modal = page.locator(".modal.show");
        await expect(modal.getByText("Are you sure you want to delete this stack?")).toBeVisible({ timeout: 5000 });

        // Verify "delete all stack files" checkbox is present and check it
        const deleteFilesCheckbox = modal.getByRole("checkbox", { name: "delete all stack files" });
        await expect(deleteFilesCheckbox).toBeVisible();
        await deleteFilesCheckbox.check();

        // Click the modal's "Delete" button (in the modal footer)
        const modalDeleteBtn = modal.locator(".modal-footer").getByRole("button", { name: "Delete" });
        await expect(modalDeleteBtn).toBeVisible();
        await modalDeleteBtn.click();

        // Verify redirect to /stacks after deletion
        await expect(page).toHaveURL(/\/stacks$/, { timeout: 15000 });
    });
});
