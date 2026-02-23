import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";

// The Compose page has two terminals: the progress terminal (compose commands)
// and the combined log terminal. Use :has(.progress-terminal-header) to scope
// to the ProgressTerminal's shadow-box, then find .xterm-rows inside it.
const terminal = (page: import("@playwright/test").Page) =>
    page.locator(".shadow-box:has(.progress-terminal-header) .xterm-rows");

test.describe("Compose Operations", () => {

    // Reset mock state after all operations so subsequent test files
    // (screenshot tests, etc.) see a clean DefaultDevState.
    test.afterAll(async ({ request }) => {
        await request.post("/api/mock/reset");
    });

    // ── Test 1: Start Stack ──────────────────────────────────────────────
    // 03-monitoring starts as "exited" in DefaultDevState()
    test("start stack (03-monitoring)", async ({ page }) => {
        await page.goto("/stacks/03-monitoring");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: "03-monitoring" })).toBeVisible({ timeout: 15000 });

        // Stack is exited → Start button visible, no Restart
        const startBtn = page.getByRole("button", { name: "Start", exact: true });
        await expect(startBtn).toBeVisible({ timeout: 15000 });
        await expect(page.getByRole("button", { name: "Restart", exact: true })).not.toBeVisible();

        // Click Start
        await startBtn.evaluate((el: HTMLElement) => el.click());

        // Verify terminal output
        const term = terminal(page);
        await expect(term).toContainText("$ docker compose up -d --remove-orphans", { timeout: 15000 });
        await expect(term).toContainText("[Done]", { timeout: 15000 });
        await expect(term).toContainText("✔");
        await expect(term).toContainText("Container 03-monitoring-grafana-1");
        await expect(term).toContainText("Started");

        // UI state: Start gone, Restart and Stop visible
        await expect(page.getByRole("button", { name: "Start", exact: true })).not.toBeVisible({ timeout: 10000 });
        await expect(page.getByRole("button", { name: "Restart", exact: true })).toBeVisible();
        await expect(page.getByRole("button", { name: "Stop", exact: true })).toBeVisible();
    });

    // ── Test 2: Stop Stack ───────────────────────────────────────────────
    // 04-database starts as "running" in DefaultDevState()
    test("stop stack (04-database)", async ({ page }) => {
        await page.goto("/stacks/04-database");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: "04-database" })).toBeVisible({ timeout: 15000 });

        // Stack is running → Stop button visible
        const stopBtn = page.getByRole("button", { name: "Stop", exact: true });
        await expect(stopBtn).toBeVisible({ timeout: 15000 });

        // Click Stop
        await stopBtn.evaluate((el: HTMLElement) => el.click());

        // Verify terminal output
        const term = terminal(page);
        await expect(term).toContainText("$ docker compose stop", { timeout: 15000 });
        await expect(term).toContainText("[Done]", { timeout: 15000 });
        await expect(term).toContainText("✔");
        await expect(term).toContainText("Stopped");

        // UI state: Start visible
        await expect(page.getByRole("button", { name: "Start", exact: true })).toBeVisible({ timeout: 10000 });
    });

    // ── Test 3: Restart Stack ────────────────────────────────────────────
    // 02-blog starts as "running" in DefaultDevState()
    test("restart stack (02-blog)", async ({ page }) => {
        await page.goto("/stacks/02-blog");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: "02-blog" })).toBeVisible({ timeout: 15000 });

        // Stack is running → Restart button visible
        const restartBtn = page.getByRole("button", { name: "Restart", exact: true });
        await expect(restartBtn).toBeVisible({ timeout: 15000 });

        // Click Restart
        await restartBtn.evaluate((el: HTMLElement) => el.click());

        // Verify terminal output
        const term = terminal(page);
        await expect(term).toContainText("$ docker compose restart", { timeout: 15000 });
        await expect(term).toContainText("[Done]", { timeout: 15000 });
        await expect(term).toContainText("✔");
        await expect(term).toContainText("Started");

        // UI state: Restart and Stop still visible (still running)
        await expect(page.getByRole("button", { name: "Restart", exact: true })).toBeVisible();
        await expect(page.getByRole("button", { name: "Stop", exact: true })).toBeVisible();
    });

    // ── Test 4: Down Stack ───────────────────────────────────────────────
    // 00-single-service starts as "running" in DefaultDevState()
    test("down stack (00-single-service)", async ({ page }) => {
        await page.goto("/stacks/00-single-service");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: "00-single-service" })).toBeVisible({ timeout: 15000 });

        // Open the dropdown menu and click "Stop & Inactive"
        await page.locator(".btn-group .dropdown-toggle").last().click();
        await page.getByText("Stop & Inactive").click();

        // Verify terminal output
        const term = terminal(page);
        await expect(term).toContainText("$ docker compose down", { timeout: 15000 });
        await expect(term).toContainText("[Done]", { timeout: 15000 });
        await expect(term).toContainText("✔");
        await expect(term).toContainText("Removed");

        // UI state: Start visible (stack is now inactive/exited)
        await expect(page.getByRole("button", { name: "Start", exact: true })).toBeVisible({ timeout: 10000 });
    });

    // ── Test 5: Update Stack ─────────────────────────────────────────────
    // 01-web-app starts as "running" in DefaultDevState()
    test("update stack (01-web-app)", async ({ page }) => {
        await page.goto("/stacks/01-web-app");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: "01-web-app" })).toBeVisible({ timeout: 15000 });

        // Click Update button to open the confirmation modal
        const updateBtn = page.getByRole("button", { name: "Update", exact: true });
        await expect(updateBtn).toBeVisible({ timeout: 15000 });
        await updateBtn.evaluate((el: HTMLElement) => el.click());

        // Confirm in the modal footer
        const modalUpdateBtn = page.locator(".modal-footer").getByRole("button", { name: "Update" });
        await expect(modalUpdateBtn).toBeVisible({ timeout: 5000 });
        await modalUpdateBtn.click();

        // Verify terminal output — two sequential commands; the pull output
        // gets overwritten by ANSI cursor movements from the subsequent up command,
        // but [Done] appears at the end.
        const term = terminal(page);
        await expect(term).toContainText("[Done]", { timeout: 15000 });
        await expect(term).toContainText("✔");

        // UI state: Restart and Stop still visible (still running)
        await expect(page.getByRole("button", { name: "Restart", exact: true })).toBeVisible();
        await expect(page.getByRole("button", { name: "Stop", exact: true })).toBeVisible();
    });

    // ── Test 6: Service Restart ──────────────────────────────────────────
    // 01-web-app nginx service (running)
    test("service restart (01-web-app nginx)", async ({ page }) => {
        await page.goto("/stacks/01-web-app");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: "01-web-app" })).toBeVisible({ timeout: 15000 });

        // Click the per-service restart button for nginx
        const restartSvc = page.getByTitle("docker compose restart nginx");
        await expect(restartSvc).toBeVisible({ timeout: 15000 });
        await restartSvc.evaluate((el: HTMLElement) => el.click());

        // Verify terminal output
        const term = terminal(page);
        await expect(term).toContainText("$ docker compose restart nginx", { timeout: 15000 });
        await expect(term).toContainText("[Done]", { timeout: 15000 });
        await expect(term).toContainText("✔");
        await expect(term).toContainText("Started");
    });

    // ── Test 7: Service Stop ─────────────────────────────────────────────
    // stack-010 ruby service (running)
    test("service stop (stack-010 ruby)", async ({ page }) => {
        await page.goto("/stacks/stack-010");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: "stack-010" })).toBeVisible({ timeout: 15000 });

        // Stop the ruby service
        const stopSvc = page.getByTitle("docker compose stop ruby");
        await expect(stopSvc).toBeVisible({ timeout: 15000 });
        await stopSvc.evaluate((el: HTMLElement) => el.click());

        // Verify terminal output
        const term = terminal(page);
        await expect(term).toContainText("$ docker compose stop ruby", { timeout: 15000 });
        await expect(term).toContainText("[Done]", { timeout: 15000 });
        await expect(term).toContainText("✔");
        await expect(term).toContainText("Stopped");
    });
});
