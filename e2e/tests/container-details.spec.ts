import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";
import { takeLightScreenshot } from "../helpers/light-mode";

test.describe("Container Inspect", () => {
    test.beforeEach(async ({ page }) => {
        await page.goto("/containers/01-web-app-nginx-1");
        await waitForApp(page);
    });

    test("displays inspect UI elements", async ({ page }) => {
        await expect.soft(page.getByRole("heading", { name: /running\s+01-web-app-nginx-1/i })).toBeVisible({ timeout: 10000 });
        await expect.soft(page.getByRole("region", { name: "Overview" })).toBeVisible({ timeout: 10000 });
    });

    test("raw toggle shows CodeMirror editor", async ({ page }) => {
        await page.getByTitle("Show YAML").click();
        await expect(page.locator(".cm-editor").first()).toBeVisible({ timeout: 10000 });
    });

    test("screenshot: container inspect", async ({ page }) => {
        await expect(page.getByRole("region", { name: "Overview" })).toBeVisible({ timeout: 10000 });
        await expect(page).toHaveScreenshot("container-inspect.png");
        await takeLightScreenshot(page, "container-inspect-light.png");
    });
});

test.describe("Container Log", () => {
    test.beforeEach(async ({ page }) => {
        // Navigate via the container card log button (not a direct URL)
        await page.goto("/stacks/01-web-app");
        await waitForApp(page);
        const logLink = page.getByRole("link", { name: "docker compose logs nginx" });
        await expect(logLink).toBeVisible({ timeout: 10000 });
        await logLink.click();
        await expect(page).toHaveURL("/containers/01-web-app-nginx-1/logs");
    });

    test("displays log UI elements", async ({ page }) => {
        await expect.soft(page.getByRole("heading", { name: /running\s+01-web-app-nginx-1/i })).toBeVisible({ timeout: 10000 });
        await expect.soft(page.getByRole("region", { name: "Logs" })).toBeVisible({ timeout: 10000 });
    });

    test("screenshot: container log", async ({ page }) => {
        await expect(page.getByRole("region", { name: "Logs" })).toBeVisible({ timeout: 10000 });
        await expect(page).toHaveScreenshot("container-log.png");
        await takeLightScreenshot(page, "container-log-light.png");
    });
});

test.describe("Container Log Lifecycle Banners", () => {
    // This test modifies mock state (stop/start), so reset before and after.
    test.beforeAll(async ({ request }) => {
        await request.post("/api/mock/reset");
    });
    test.afterAll(async ({ request }) => {
        await request.post("/api/mock/reset");
    });

    test("shows shutdown logs, stop banner, start banner, and startup logs", async ({ page }) => {
        // Navigate to the individual log view for alpine (running in DefaultDevState)
        await page.goto("/containers/00-single-service-alpine-1/logs");
        await waitForApp(page);

        // The page has two Terminal regions: the progress terminal (inside
        // .progress-terminal) and the log terminal (with class .terminal).
        // Scope to the log terminal's .xterm-rows.
        const logTerminal = page.locator(".terminal.shadow-box .xterm-rows");
        await expect(logTerminal).toBeVisible({ timeout: 10000 });

        // Verify log content is loaded (startup + 1 heartbeat in e2e mode)
        await expect(logTerminal).toContainText("alpine container started", { timeout: 10000 });

        // Stop the container
        const stopBtn = page.getByRole("button", { name: "Stop", exact: true });
        await expect(stopBtn).toBeVisible({ timeout: 10000 });
        await stopBtn.evaluate((el: HTMLElement) => el.click());

        // Wait for the progress terminal to show the stop command completed
        const progressTerm = page.locator(".progress-terminal .xterm-rows");
        await expect(progressTerm).toContainText("[Done]", { timeout: 15000 });

        // Verify shutdown log appears (from log-templates.yaml alpine.shutdown)
        await expect(logTerminal).toContainText("Received SIGTERM, exiting", { timeout: 10000 });

        // Verify stop banner appears
        await expect(logTerminal).toContainText("CONTAINER STOP", { timeout: 10000 });

        // Start the container back up
        const startBtn = page.getByRole("button", { name: "Start", exact: true });
        await expect(startBtn).toBeVisible({ timeout: 10000 });
        await startBtn.evaluate((el: HTMLElement) => el.click());

        // Wait for start to complete
        await expect(progressTerm).toContainText("Started", { timeout: 15000 });

        // Verify start banner appears
        await expect(logTerminal).toContainText("CONTAINER START", { timeout: 10000 });

        // This causes the next assert to pass
        // await page.screenshot({ path: 'ContainerStart-DEBUG.png' });

        // Force a rendering fence before assertion
        await page.evaluate(() => new Promise(resolve => requestAnimationFrame(resolve)))

        // Verify startup logs appear again after restart (from log-templates.yaml alpine.startup).
        // Use a regex to confirm startup text follows the CONTAINER START banner,
        // proving the log stream reconnected after the restart.
        await expect(logTerminal).toContainText(/CONTAINER START.*alpine container started/s, { timeout: 15000 });

        // Dismiss progress terminal so screenshots are deterministic
        await page.getByRole("region", { name: "Progress" }).getByTitle("Close").click();

        // Screenshot after full stop/start cycle
        await expect(page).toHaveScreenshot("container-log-lifecycle.png");
        await takeLightScreenshot(page, "container-log-lifecycle-light.png");
    });

    test("restart triggers banners and logs", async ({ page, request }) => {
        await request.post("/api/mock/reset");
        await page.goto("/containers/00-single-service-alpine-1/logs");
        await waitForApp(page);

        const logTerminal = page.locator(".terminal.shadow-box .xterm-rows");
        await expect(logTerminal).toBeVisible({ timeout: 10000 });
        await expect(logTerminal).toContainText("[INFO] Health check OK", { timeout: 10000 });

        const restartBtn = page.getByRole("button", { name: "Restart", exact: true });
        await expect(restartBtn).toBeVisible({ timeout: 10000 });
        await restartBtn.evaluate((el: HTMLElement) => el.click());

        const progressTerm = page.locator(".progress-terminal .xterm-rows");
        await expect(progressTerm).toContainText("Started", { timeout: 15000 });

        await expect(logTerminal).toContainText("Received SIGTERM, exiting", { timeout: 10000 });
        await expect(logTerminal).toContainText("CONTAINER STOP", { timeout: 10000 });
        await expect(logTerminal).toContainText("CONTAINER START", { timeout: 10000 });

        // Force a rendering fence before assertion
        await page.evaluate(() => new Promise(resolve => requestAnimationFrame(resolve)))

        // Verify startup logs reappear after the CONTAINER START banner,
        // confirming the log stream reconnected after restart.
        await expect(logTerminal).toContainText(/CONTAINER START.*alpine container started/s, { timeout: 15000 });

        await page.getByRole("region", { name: "Progress" }).getByTitle("Close").click();
        await expect(page).toHaveScreenshot("container-log-restart.png");
        await takeLightScreenshot(page, "container-log-restart-light.png");
    });

    test("recreate triggers banners and logs", async ({ page, request }) => {
        await request.post("/api/mock/reset");
        await page.goto("/containers/00-single-service-alpine-1/logs");
        await waitForApp(page);

        const logTerminal = page.locator(".terminal.shadow-box .xterm-rows");
        await expect(logTerminal).toBeVisible({ timeout: 10000 });
        await expect(logTerminal).toContainText("[INFO] Health check OK", { timeout: 10000 });

        const recreateBtn = page.getByRole("button", { name: "Recreate", exact: true });
        await expect(recreateBtn).toBeVisible({ timeout: 10000 });
        await recreateBtn.evaluate((el: HTMLElement) => el.click());

        const progressTerm = page.locator(".progress-terminal .xterm-rows");
        await expect(progressTerm).toContainText("Started", { timeout: 15000 });

        await expect(logTerminal).toContainText("Received SIGTERM, exiting", { timeout: 10000 });
        await expect(logTerminal).toContainText("CONTAINER STOP", { timeout: 10000 });
        await expect(logTerminal).toContainText("CONTAINER START", { timeout: 10000 });
        // Verify startup logs reappear after the CONTAINER START banner
        await expect(logTerminal).toContainText(/CONTAINER START.*alpine container started/s, { timeout: 15000 });

        await page.getByRole("region", { name: "Progress" }).getByTitle("Close").click();
        await expect(page).toHaveScreenshot("container-log-recreate.png");
        await takeLightScreenshot(page, "container-log-recreate-light.png");
    });

    test("update triggers banners and logs", async ({ page, request }) => {
        await request.post("/api/mock/reset");
        await page.goto("/containers/00-single-service-alpine-1/logs");
        await waitForApp(page);

        const logTerminal = page.locator(".terminal.shadow-box .xterm-rows");
        await expect(logTerminal).toBeVisible({ timeout: 10000 });
        await expect(logTerminal).toContainText("[INFO] Health check OK", { timeout: 10000 });

        // Click "Update" button to open the dialog
        const updateBtn = page.getByRole("button", { name: "Update", exact: true });
        await expect(updateBtn).toBeVisible({ timeout: 10000 });
        await updateBtn.evaluate((el: HTMLElement) => el.click());

        // Confirm the update in the dialog
        const dialogUpdateBtn = page.locator(".modal-footer").getByRole("button", { name: "Update" });
        await expect(dialogUpdateBtn).toBeVisible({ timeout: 5000 });
        await dialogUpdateBtn.evaluate((el: HTMLElement) => el.click());

        const progressTerm = page.locator(".progress-terminal .xterm-rows");
        await expect(progressTerm).toContainText("Started", { timeout: 15000 });

        await expect(logTerminal).toContainText("Received SIGTERM, exiting", { timeout: 10000 });
        await expect(logTerminal).toContainText("CONTAINER STOP", { timeout: 10000 });
        await expect(logTerminal).toContainText("CONTAINER START", { timeout: 10000 });
        // Verify startup logs reappear after the CONTAINER START banner
        await expect(logTerminal).toContainText(/CONTAINER START.*alpine container started/s, { timeout: 15000 });

        await page.getByRole("region", { name: "Progress" }).getByTitle("Close").click();
        await expect(page).toHaveScreenshot("container-log-update.png");
        await takeLightScreenshot(page, "container-log-update-light.png");
    });
});

test.describe("Container Terminal", () => {
    test.beforeEach(async ({ page }) => {
        await page.goto("/terminal/01-web-app/nginx/bash");
        await waitForApp(page);
    });

    test("displays terminal UI elements", async ({ page }) => {
        await expect.soft(page.getByRole("heading", { name: /Terminal.*nginx.*01-web-app/i })).toBeVisible({ timeout: 10000 });
        await expect.soft(page.getByRole("link", { name: /Switch to sh/i })).toBeVisible();
        await expect.soft(page.getByRole("region", { name: "Shell" })).toBeVisible({ timeout: 10000 });
    });

    test("screenshot: container terminal", async ({ page }) => {
        await expect(page.getByRole("region", { name: "Shell" })).toBeVisible({ timeout: 10000 });
        await expect(page).toHaveScreenshot("container-terminal.png");
        await takeLightScreenshot(page, "container-terminal-light.png");
    });
});
