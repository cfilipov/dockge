import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";
import { Locator, Page } from "@playwright/test";

// Use evaluate(el.click()) instead of Playwright's .click() for buttons that
// may shift due to concurrent WebSocket layout changes.
function evalClick(locator: Locator) {
    return locator.evaluate((el: HTMLElement) => el.click());
}

/**
 * Enter edit mode on a compose page: clicks the page-level Edit button,
 * then the container card's Edit button to expand the config panel.
 */
async function enterEditMode(page: Page, appCard: Locator) {
    // Click page-level "Edit" button
    await evalClick(page.getByRole("button", {
        name: "Edit",
        exact: true,
    }));
    await expect(page.getByRole("button", { name: "Deploy" })).toBeVisible();

    // Click container card's "Edit" button to expand config
    await evalClick(appCard.getByRole("button", { name: "Edit" }));
    await expect(appCard.locator(".config")).toBeVisible();
}

test.describe("Container Card Editing", () => {
    let appCard: Locator;
    let yamlEditor: Locator;

    test.beforeEach(async ({ page }) => {
        await page.goto("/compose/06-mixed-state");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: "06-mixed-state" })).toBeVisible({ timeout: 15000 });

        // Locate the "app" service container card
        appCard = page.locator(".shadow-box.big-padding").filter({
            has: page.locator("h4", { hasText: /^app$/ }),
        });

        // The main compose YAML editor content (first CodeMirror on the page)
        yamlEditor = page.locator(".cm-content").first();

        await enterEditMode(page, appCard);
    });

    test("change image name updates YAML", async () => {
        const imageInput = appCard.locator("input[list=\"image-datalist\"]");
        await imageInput.clear();
        await imageInput.fill("myapp:v2.0");

        await expect(yamlEditor).toContainText("myapp:v2.0");
    });

    test("set restart policy updates YAML", async () => {
        const restartSelect = appCard.locator("select.form-select");
        await restartSelect.selectOption("always");

        await expect(yamlEditor).toContainText("restart: always");
    });

    test("add port updates YAML", async () => {
        await evalClick(appCard.getByRole("button", { name: /Add Port/i }));

        const portInputs = appCard.locator(".domain-input[placeholder=\"HOST:CONTAINER\"]");
        const lastPort = portInputs.last();
        await lastPort.fill("9090:90");

        await expect(yamlEditor).toContainText("9090:90");
    });

    test("add environment variable updates YAML", async () => {
        await evalClick(appCard.getByRole("button", { name: /Add Environment Variable/i }));

        const envInputs = appCard.locator(".domain-input[placeholder=\"KEY=VALUE\"]");
        const lastEnv = envInputs.last();
        await lastEnv.fill("MY_VAR=hello");

        await expect(yamlEditor).toContainText("MY_VAR=hello");
    });

    test("add container dependency updates YAML", async () => {
        await evalClick(appCard.getByRole("button", { name: /Add Container Dependency/i }));

        const depInputs = appCard.locator(".domain-input[placeholder=\"Container Name\"]");
        const lastDep = depInputs.last();
        await lastDep.fill("db");

        await expect(yamlEditor).toContainText("depends_on");
    });

    test("add URL updates YAML", async () => {
        await evalClick(appCard.getByRole("button", { name: /Add URL/i }));

        const urlInput = appCard.locator(".url-list .domain-input").last();
        // URL inputs use :value + @input (not v-model), so type() is needed
        await urlInput.pressSequentially("https://app.example.com", { delay: 50 });

        await expect(yamlEditor).toContainText("dockge.urls");
        await expect(yamlEditor).toContainText("https://app.example.com");
    });

    test("toggle ignore status and image updates check updates YAML", async () => {
        // Toggle "Ignore status" ON
        await appCard.getByText("Ignore status").click();
        await expect(yamlEditor).toContainText("dockge.status.ignore");

        // Toggle "Check for image updates" OFF (default is ON)
        await appCard.getByText("Check for image updates").click();
        await expect(yamlEditor).toContainText("dockge.imageupdates.check");
    });

    test("set changelog URL updates YAML", async () => {
        // Target the changelog input (has placeholder="https://") but NOT the image input (which has list attr)
        const changelogInput = appCard.locator("input.form-control[placeholder=\"https://\"]");
        await changelogInput.fill("https://changelog.example.com");

        await expect(yamlEditor).toContainText("dockge.imageupdates.changelog");
        await expect(yamlEditor).toContainText("https://changelog.example.com");
    });

    test("add network via sidebar and container card updates YAML", async ({ page }) => {
        // 1. Add an internal network in the sidebar NetworkInput section
        //    The NetworkInput is inside the right column's last shadow-box
        const networkSection = page.locator(".shadow-box.big-padding").filter({
            has: page.getByText("Internal Networks"),
        });

        await evalClick(networkSection.getByRole("button", {
            name: "Add",
            exact: true,
        }));

        const networkNameInput = networkSection.locator("input[placeholder=\"Network name...\"]");
        await networkNameInput.fill("my-net");

        // 2. In the app card, click "Add Network" and select "my-net"
        await evalClick(appCard.getByRole("button", { name: /Add Network/i }));

        // Wait for the option to appear in the dropdown
        const networkSelect = appCard.locator("select.domain-input").last();
        await expect(networkSelect.locator("option", { hasText: "my-net" })).toBeAttached();

        await networkSelect.selectOption("my-net");

        // 3. Verify YAML contains network references
        await expect(yamlEditor).toContainText("networks");
        await expect(yamlEditor).toContainText("my-net");
    });
});
