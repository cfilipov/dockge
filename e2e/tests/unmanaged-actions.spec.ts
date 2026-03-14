import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";

// Progress terminal locator — scoped to the ProgressTerminal wrapper.
const terminal = (page: import("@playwright/test").Page) =>
    page.locator(".progress-terminal .xterm-rows");

// ── Section 1: Unmanaged Stack — Stacks Tab (Container Cards) ──────────────
//
// 10-unmanaged is an external stack: containers have com.docker.compose.project
// labels but no compose file on disk. Service-level actions use plain docker
// commands (start/stop/restart) since compose up/recreate need a compose file.
// Both services (web, cache) start as "running" in DefaultDevState.
//
// Buttons on Container.vue have aria-label matching the docker command.

test.describe("Unmanaged Stack — Stacks Tab", () => {

    test.beforeAll(async ({ request }) => {
        await request.post("/api/mock/reset");
    });
    test.afterAll(async ({ request }) => {
        await request.post("/api/mock/reset");
    });

    // Test 1: Stop then start cache service (single test avoids cross-test state dependency)
    test("stop then start cache service", async ({ page }) => {
        await page.goto("/stacks/10-unmanaged");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: /10-unmanaged/, level: 1 })).toBeVisible({ timeout: 15000 });

        // --- Phase 1: Stop cache ---
        // cache is running → stop button visible (plain docker command for unmanaged)
        const stopBtn = page.getByRole("button", { name: "docker stop 10-unmanaged-cache-1" });
        await expect(stopBtn).toBeVisible({ timeout: 15000 });
        await stopBtn.evaluate((el: HTMLElement) => el.click());

        // Verify stop terminal output
        const term = terminal(page);
        await expect(term).toContainText("$ docker stop 10-unmanaged-cache-1", { timeout: 15000 });
        await expect(term).toContainText("[Done]", { timeout: 15000 });

        // UI: start button appears for cache
        const startBtn = page.getByRole("button", { name: "docker start 10-unmanaged-cache-1" });
        await expect(startBtn).toBeVisible({ timeout: 10000 });

        // --- Phase 2: Start cache ---
        await startBtn.evaluate((el: HTMLElement) => el.click());

        // Verify start terminal output
        await expect(term).toContainText("$ docker start 10-unmanaged-cache-1", { timeout: 15000 });
        await expect(term).toContainText("[Done]", { timeout: 15000 });

        // UI: restart and stop buttons visible for cache
        await expect(page.getByRole("button", { name: "docker restart 10-unmanaged-cache-1" })).toBeVisible({ timeout: 10000 });
        await expect(page.getByRole("button", { name: "docker stop 10-unmanaged-cache-1" })).toBeVisible();
    });

    // Test 2: Restart web service
    test("restart web service", async ({ page }) => {
        await page.goto("/stacks/10-unmanaged");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: /10-unmanaged/, level: 1 })).toBeVisible({ timeout: 15000 });

        // web is running → restart button visible (plain docker command for unmanaged)
        const restartBtn = page.getByRole("button", { name: "docker restart 10-unmanaged-web-1" });
        await expect(restartBtn).toBeVisible({ timeout: 15000 });
        await restartBtn.evaluate((el: HTMLElement) => el.click());

        // Verify terminal output
        const term = terminal(page);
        await expect(term).toContainText("$ docker restart 10-unmanaged-web-1", { timeout: 15000 });
        await expect(term).toContainText("[Done]", { timeout: 15000 });

        // UI: restart and stop still visible (web is still running)
        await expect(page.getByRole("button", { name: "docker restart 10-unmanaged-web-1" })).toBeVisible();
        await expect(page.getByRole("button", { name: "docker stop 10-unmanaged-web-1" })).toBeVisible();
    });

    // Test 3: No recreate or update buttons for unmanaged stacks
    // Recreate and Update require a compose file, so they're hidden for unmanaged stacks.
    test("no recreate or update buttons", async ({ page }) => {
        await page.goto("/stacks/10-unmanaged");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: /10-unmanaged/, level: 1 })).toBeVisible({ timeout: 15000 });

        // The web service card should have restart and stop but NOT recreate or update
        const webRegion = page.getByRole("region", { name: "web" });
        await expect(webRegion.getByRole("button", { name: "docker restart 10-unmanaged-web-1" })).toBeVisible({ timeout: 10000 });
        await expect(webRegion.getByRole("button", { name: "docker stop 10-unmanaged-web-1" })).toBeVisible();

        // Recreate (rocket icon) and Update (cloud-arrow-down icon) should not be present
        // These buttons would have tooltipServiceRecreate / tooltipServiceUpdate aria-labels
        // for managed stacks, but should be completely absent for unmanaged
        const buttons = await webRegion.getByRole("button").all();
        const labels = await Promise.all(buttons.map(b => b.getAttribute("aria-label")));
        expect(labels.some(l => l?.includes("force-recreate"))).toBe(false);
        expect(labels.some(l => l?.includes("pull"))).toBe(false);
    });
});

// ── Section 2: Unmanaged Stack — Containers Tab ────────────────────────────
//
// Container inspect page for 10-unmanaged containers. ServiceActionBar renders
// because stackName is set from the compose project label. For unmanaged stacks,
// actions use plain docker commands (same as standalone containers).

test.describe("Unmanaged Stack — Containers Tab", () => {

    test.beforeAll(async ({ request }) => {
        await request.post("/api/mock/reset");
    });
    test.afterAll(async ({ request }) => {
        await request.post("/api/mock/reset");
    });

    // Test 1: Stop web via container inspect page
    test("stop web from container page", async ({ page }) => {
        await page.goto("/containers/10-unmanaged-web-1");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: /10-unmanaged-web-1/, level: 1 })).toBeVisible({ timeout: 15000 });

        // ServiceActionBar Stop button (visible text "Stop")
        const stopBtn = page.getByRole("button", { name: "Stop", exact: true });
        await expect(stopBtn).toBeVisible({ timeout: 15000 });
        await stopBtn.evaluate((el: HTMLElement) => el.click());

        // Verify terminal output — plain docker command for unmanaged
        const term = terminal(page);
        await expect(term).toContainText("$ docker stop 10-unmanaged-web-1", { timeout: 15000 });
        await expect(term).toContainText("[Done]", { timeout: 15000 });
    });

    // Test 2: Restart cache via container inspect page
    test("restart cache from container page", async ({ page }) => {
        await page.goto("/containers/10-unmanaged-cache-1");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: /10-unmanaged-cache-1/, level: 1 })).toBeVisible({ timeout: 15000 });

        // ServiceActionBar Restart button
        const restartBtn = page.getByRole("button", { name: "Restart", exact: true });
        await expect(restartBtn).toBeVisible({ timeout: 15000 });
        await restartBtn.evaluate((el: HTMLElement) => el.click());

        // Verify terminal output — plain docker command for unmanaged
        const term = terminal(page);
        await expect(term).toContainText("$ docker restart 10-unmanaged-cache-1", { timeout: 15000 });
        await expect(term).toContainText("[Done]", { timeout: 15000 });
    });
});

// ── Section 3: Standalone Containers — Containers Tab ──────────────────────
//
// Standalone containers have no compose project label. ContainerInspect renders
// a simple action bar with Start/Stop/Restart. Actions use plain docker commands.
// portainer and watchtower start as "running"; homeassistant starts as "exited".

test.describe("Standalone Containers — Containers Tab", () => {

    test.beforeAll(async ({ request }) => {
        await request.post("/api/mock/reset");
    });
    test.afterAll(async ({ request }) => {
        await request.post("/api/mock/reset");
    });

    // Test 1: Stop portainer
    test("stop portainer", async ({ page }) => {
        await page.goto("/containers/portainer");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: /portainer/, level: 1 })).toBeVisible({ timeout: 15000 });

        // Standalone Stop button
        const stopBtn = page.getByRole("button", { name: "Stop", exact: true });
        await expect(stopBtn).toBeVisible({ timeout: 15000 });
        await stopBtn.evaluate((el: HTMLElement) => el.click());

        // Verify terminal output
        const term = terminal(page);
        await expect(term).toContainText("$ docker stop portainer", { timeout: 15000 });
        await expect(term).toContainText("[Done]", { timeout: 15000 });

        // UI: Start appears, no Restart/Stop
        await expect(page.getByRole("button", { name: "Start", exact: true })).toBeVisible({ timeout: 10000 });
        await expect(page.getByRole("button", { name: "Restart", exact: true })).not.toBeVisible();
        await expect(page.getByRole("button", { name: "Stop", exact: true })).not.toBeVisible();
    });

    // Test 2: Start homeassistant (initially exited)
    test("start homeassistant", async ({ page }) => {
        await page.goto("/containers/homeassistant");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: /homeassistant/, level: 1 })).toBeVisible({ timeout: 15000 });

        // Container is exited → Start button visible
        const startBtn = page.getByRole("button", { name: "Start", exact: true });
        await expect(startBtn).toBeVisible({ timeout: 15000 });
        await expect(page.getByRole("button", { name: "Restart", exact: true })).not.toBeVisible();

        await startBtn.evaluate((el: HTMLElement) => el.click());

        // Verify terminal output
        const term = terminal(page);
        await expect(term).toContainText("$ docker start homeassistant", { timeout: 15000 });
        await expect(term).toContainText("[Done]", { timeout: 15000 });

        // UI: Restart and Stop appear
        await expect(page.getByRole("button", { name: "Restart", exact: true })).toBeVisible({ timeout: 10000 });
        await expect(page.getByRole("button", { name: "Stop", exact: true })).toBeVisible();
    });

    // Test 3: Restart watchtower
    test("restart watchtower", async ({ page }) => {
        await page.goto("/containers/watchtower");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: /watchtower/, level: 1 })).toBeVisible({ timeout: 15000 });

        // Container is running → Restart visible
        const restartBtn = page.getByRole("button", { name: "Restart", exact: true });
        await expect(restartBtn).toBeVisible({ timeout: 15000 });
        await restartBtn.evaluate((el: HTMLElement) => el.click());

        // Verify terminal output
        const term = terminal(page);
        await expect(term).toContainText("$ docker restart watchtower", { timeout: 15000 });
        await expect(term).toContainText("[Done]", { timeout: 15000 });

        // UI: Restart and Stop still visible (still running)
        await expect(page.getByRole("button", { name: "Restart", exact: true })).toBeVisible();
        await expect(page.getByRole("button", { name: "Stop", exact: true })).toBeVisible();
    });

    // Test 4: Standalone containers should NOT have Recreate or Update buttons
    test("no recreate or update buttons", async ({ page }) => {
        await page.goto("/containers/portainer");
        await waitForApp(page);
        await expect(page.getByRole("heading", { name: /portainer/, level: 1 })).toBeVisible({ timeout: 15000 });

        // Standalone action bar does not include Recreate or Update
        await expect(page.getByRole("button", { name: "Recreate" })).not.toBeVisible();
        await expect(page.getByRole("button", { name: "Update" })).not.toBeVisible();
    });
});
