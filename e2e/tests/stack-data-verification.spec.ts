/**
 * stack-data-verification.spec.ts — Data-driven E2E tests that parse compose.yaml
 * and mock.yaml for stacks 00-08, compute expected UI state, and assert every
 * visible element matches.
 *
 * Design: 2 tests using test.step() for granular failure reporting without the
 * overhead of separate pages.
 */

import * as path from "node:path";
import { fileURLToPath } from "node:url";
import { test, expect } from "../fixtures/auth.fixture";
import { waitForApp } from "../helpers/wait-for-app";
import {
    loadAllStacks,
} from "../helpers/stack-data";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const STACKS_DIR = path.resolve(__dirname, "../../test-data/stacks");

// Badge text values from web/src/lang/en.json
const BADGE_LABELS: Record<string, string> = {
    active: "active",
    partially: "active\u207b",   // Unicode superscript minus
    unhealthy: "unhealthy",
    exited: "exited",
    down: "down",
    running: "running",
};

// Load all stack data once (pure computation, no I/O after initial read)
const allStacks = loadAllStacks(STACKS_DIR);

test.describe("Stack data verification", () => {
    test.beforeAll(async ({ request }) => {
        await request.post("/api/mock/reset");
    });
    test.afterAll(async ({ request }) => {
        await request.post("/api/mock/reset");
    });

    test("stack list and detail pages match compose/mock data", async ({ page }) => {
        test.setTimeout(120_000);

        // ── Stack List ──
        await test.step("stack list — all 9 stacks have correct badges and icons", async () => {
            await page.goto("/");
            await waitForApp(page);
            // Wait for stack items to appear
            await expect(page.locator(".item").first()).toBeVisible({ timeout: 10000 });

            for (const stack of allStacks) {
                await test.step(`stack list: ${stack.name}`, async () => {
                    const item = page.locator(".item").filter({ hasText: stack.name });
                    await expect.soft(item.first()).toBeVisible({ timeout: 5000 });

                    // Badge text
                    const badgeText = BADGE_LABELS[stack.expectedStackBadge.label] || stack.expectedStackBadge.label;
                    const badge = item.locator(`.badge.${stack.expectedStackBadge.color}`);
                    await expect.soft(badge.first()).toBeVisible();
                    await expect.soft(badge.first()).toHaveText(badgeText);

                    // Update icon — FontAwesome renders title as SVG <title> child,
                    // accessible via role="img" + aria-labelledby
                    const updateIcon = item.getByRole("img", { name: "Image update available" });
                    if (stack.expectedHasUpdateIcon) {
                        await expect.soft(updateIcon.first()).toBeVisible();
                    } else {
                        await expect.soft(updateIcon).toHaveCount(0);
                    }

                    // Recreate icon
                    const recreateIcon = item.getByRole("img", { name: "Container needs recreation" });
                    if (stack.expectedHasRecreateIcon) {
                        await expect.soft(recreateIcon.first()).toBeVisible();
                    } else {
                        await expect.soft(recreateIcon).toHaveCount(0);
                    }
                });
            }
        });

        // ── Detail Pages ──
        for (const stack of allStacks) {
            await test.step(`${stack.name} — container cards, chips, buttons, editor`, async () => {
                await page.goto(`/stacks/${stack.name}`);
                await waitForApp(page);

                // Heading contains badge text + stack name, e.g. "active 00-single-service"
                await expect(page.getByRole("heading", { name: new RegExp(stack.name) }).first()).toBeVisible({ timeout: 10000 });

                // ── Stack-level buttons ──
                await test.step("stack-level buttons", async () => {
                    const stackStarted = stack.expectedStackStatus === 3 /* RUNNING */ ||
                        stack.expectedStackStatus === 5 /* RUNNING_AND_EXITED */ ||
                        stack.expectedStackStatus === 6 /* UNHEALTHY */;

                    if (stackStarted) {
                        // Active: Restart, Stop visible; Start hidden
                        await expect.soft(page.getByRole("button", { name: "Restart", exact: true })).toBeVisible();
                        await expect.soft(page.getByRole("button", { name: "Stop", exact: true })).toBeVisible();
                        await expect.soft(page.getByRole("button", { name: "Start", exact: true })).not.toBeVisible();
                    } else {
                        // Inactive: Start visible; Restart, Stop hidden
                        await expect.soft(page.getByRole("button", { name: "Start", exact: true })).toBeVisible();
                        await expect.soft(page.getByRole("button", { name: "Restart", exact: true })).not.toBeVisible();
                        await expect.soft(page.getByRole("button", { name: "Stop", exact: true })).not.toBeVisible();
                    }

                    // Edit always visible
                    await expect.soft(page.getByRole("button", { name: "Edit", exact: true })).toBeVisible();

                    // Update button always present; class depends on whether updates available
                    const updateBtn = page.getByTitle("docker compose pull && up -d --remove-orphans && image prune");
                    await expect.soft(updateBtn).toBeVisible();
                    if (stack.expectedHasUpdateIcon) {
                        await expect.soft(updateBtn).toHaveClass(/btn-info/);
                    } else {
                        await expect.soft(updateBtn).toHaveClass(/btn-normal/);
                    }
                });

                // ── Container cards ──
                for (const svc of stack.services) {
                    await test.step(`service: ${svc.name}`, async () => {
                        const card = page.getByRole("region", { name: svc.name });
                        await expect.soft(card).toBeVisible({ timeout: 5000 });

                        // Badge
                        const badgeText = BADGE_LABELS[svc.expectedBadge.label] || svc.expectedBadge.label;
                        const badge = card.locator(`.badge.${svc.expectedBadge.color}`);
                        await expect.soft(badge).toBeVisible();
                        await expect.soft(badge).toHaveText(badgeText);

                        // SERVICE chip
                        const serviceChip = card.locator(".network-chip").filter({ hasText: svc.name }).first();
                        await expect.soft(serviceChip).toBeVisible();

                        // IMAGE chip — shows running image for started services, compose image for down
                        const displayImage = svc.isStarted || svc.mockState === "exited" ? svc.runningImage : svc.composeImage;
                        if (displayImage) {
                            const imageChip = card.locator(".network-chip").filter({ has: page.locator(".chip-label", { hasText: /^Image$/i }) });
                            await expect.soft(imageChip).toBeVisible();
                            const [imgName, imgTag] = displayImage.includes(":") ? displayImage.split(":") : [displayImage, "latest"];
                            await expect.soft(imageChip.locator("code")).toContainText(`${imgName}:${imgTag}`);
                        }

                        // PORT chips — check actual host port display text
                        if (svc.portDisplays.length > 0) {
                            const portChip = card.locator(".network-chip").filter({ has: page.locator(".chip-label", { hasText: /^Ports?$/i }) });
                            await expect.soft(portChip).toBeVisible();
                            for (const portDisplay of svc.portDisplays) {
                                await expect.soft(portChip.locator("code").filter({ hasText: portDisplay })).toBeVisible();
                            }
                        }

                        // ── Action buttons ──
                        // Note: exact: true on Start button to avoid matching the Update button
                        // whose label contains "docker compose pull {name} && docker compose up -d {name}"
                        const startBtn = card.getByRole("button", { name: `docker compose up -d ${svc.name}`, exact: true });
                        if (svc.isStarted) {
                            // Started: Restart, Stop visible; Start hidden; Log/Terminal links visible
                            await expect.soft(card.getByRole("button", { name: `docker compose restart ${svc.name}` })).toBeVisible();
                            await expect.soft(card.getByRole("button", { name: `docker compose stop ${svc.name}` })).toBeVisible();
                            await expect.soft(startBtn).not.toBeVisible();
                            await expect.soft(card.getByRole("link", { name: `docker compose logs ${svc.name}` })).toBeVisible();
                            await expect.soft(card.getByRole("link", { name: `docker compose exec ${svc.name}` })).toBeVisible();
                        } else {
                            // Exited or down: Start visible; Restart, Stop hidden
                            await expect.soft(startBtn).toBeVisible();
                            await expect.soft(card.getByRole("button", { name: `docker compose restart ${svc.name}` })).not.toBeVisible();
                            await expect.soft(card.getByRole("button", { name: `docker compose stop ${svc.name}` })).not.toBeVisible();
                        }

                        // Recreate button: btn-info if hasRecreate, else btn-normal
                        const recreateBtn = card.getByRole("button", { name: `docker compose up -d --force-recreate ${svc.name}` });
                        await expect.soft(recreateBtn).toBeVisible();
                        if (svc.hasRecreate) {
                            await expect.soft(recreateBtn).toHaveClass(/btn-info/);
                        } else {
                            await expect.soft(recreateBtn).toHaveClass(/btn-normal/);
                        }

                        // Update button: btn-info if updateAvailable, else btn-normal
                        const updateBtn = card.getByRole("button", { name: new RegExp(`docker compose pull ${svc.name}`) });
                        await expect.soft(updateBtn).toBeVisible();
                        if (svc.updateAvailable) {
                            await expect.soft(updateBtn).toHaveClass(/btn-info/);
                        } else {
                            await expect.soft(updateBtn).toHaveClass(/btn-normal/);
                        }
                    });
                }

                // ── Compose editor content ──
                await test.step("compose editor content", async () => {
                    // Switch to raw YAML view
                    await page.getByTitle("Show YAML").click();

                    const editorContent = page.locator(".cm-content").first();
                    await expect(editorContent).toBeVisible({ timeout: 5000 });

                    // CM6 only renders lines visible in the browser viewport.
                    // Temporarily expand the viewport to render all editor lines.
                    const origSize = page.viewportSize();
                    await page.setViewportSize({ width: origSize?.width || 1280, height: 8000 });
                    // Allow CM6 to re-render with the new viewport
                    await page.waitForTimeout(100);

                    const editorText = await editorContent.innerText();

                    // Restore original viewport
                    await page.setViewportSize(origSize || { width: 1280, height: 720 });

                    // Verify each service name and image from compose.yaml
                    for (const svc of stack.services) {
                        expect.soft(editorText, `editor should contain service "${svc.name}"`).toContain(svc.name);
                        if (svc.composeImage) {
                            expect.soft(editorText, `editor should contain image "${svc.composeImage}"`).toContain(svc.composeImage);
                        }
                    }
                });
            });
        }
    });

    test("global resource tabs match compose/mock data", async ({ page }) => {
        test.setTimeout(120_000);

        // ── Containers tab ──
        await test.step("Containers tab — containers from stacks 00-08", async () => {
            await page.goto("/containers");
            await waitForApp(page);
            await expect(page.locator(".item").first()).toBeVisible({ timeout: 10000 });

            // Check a subset of containers from running stacks
            for (const stack of allStacks) {
                if (stack.mockStatus === "inactive") continue; // inactive stacks have no containers

                for (const svc of stack.services) {
                    await test.step(`container: ${stack.name}-${svc.name}`, async () => {
                        const containerName = `${stack.name}-${svc.name}-1`;

                        // Search for the container
                        const searchInput = page.getByPlaceholder("Search...");
                        await searchInput.fill(containerName);

                        // Verify it appears in the list
                        const item = page.locator(".item").filter({ hasText: containerName });
                        await expect.soft(item.first()).toBeVisible({ timeout: 5000 });

                        // Verify status badge
                        const badgeText = BADGE_LABELS[svc.expectedBadge.label] || svc.expectedBadge.label;
                        const badge = item.locator(`.badge.${svc.expectedBadge.color}`).first();
                        await expect.soft(badge).toBeVisible();
                        await expect.soft(badge).toHaveText(badgeText);
                    });
                }
            }

            // Clear search
            const searchInput = page.getByPlaceholder("Search...");
            await searchInput.fill("");
        });

        // ── Images tab ──
        await test.step("Images tab — images with correct badges", async () => {
            await page.goto("/images");
            await waitForApp(page);
            await expect(page.locator(".item").first()).toBeVisible({ timeout: 10000 });

            // Check a representative subset of images
            const imagesToCheck = [
                "alpine:latest",
                "nginx:latest",
                "postgres:16",
                "redis:7",
                "grafana/grafana:latest",
            ];

            for (const imageRef of imagesToCheck) {
                await test.step(`image: ${imageRef}`, async () => {
                    const searchInput = page.getByPlaceholder("Search...");
                    await searchInput.fill(imageRef);

                    const item = page.locator(".item").filter({ hasText: imageRef });
                    await expect.soft(item.first()).toBeVisible({ timeout: 5000 });
                });
            }

            // Check one detail page
            await test.step("image detail page", async () => {
                const searchInput = page.getByPlaceholder("Search...");
                await searchInput.fill("nginx:latest");

                const item = page.locator(".item").filter({ hasText: "nginx:latest" });
                await expect(item.first()).toBeVisible({ timeout: 5000 });
                await item.first().click();

                const overview = page.getByRole("region", { name: "Overview" });
                await expect(overview).toBeVisible({ timeout: 10000 });
                await expect.soft(overview.getByText("Architecture")).toBeVisible();
                await expect.soft(overview.getByText("OS")).toBeVisible();
            });
        });

        // ── Networks tab ──
        await test.step("Networks tab — networks with correct detail", async () => {
            await page.goto("/networks");
            await waitForApp(page);
            await expect(page.locator(".item").first()).toBeVisible({ timeout: 10000 });

            // The mock creates _default networks for most stacks, and named networks
            // only for 07-full-features (frontend, backend). Verify a subset.
            const networksToCheck = [
                "01-web-app_default",
                "04-database_default",
                "07-full-features_frontend",
                "07-full-features_backend",
            ];

            for (const netName of networksToCheck) {
                await test.step(`network: ${netName}`, async () => {
                    const searchInput = page.getByPlaceholder("Search...");
                    await searchInput.fill(netName);

                    const item = page.locator(".item").filter({ hasText: netName });
                    await expect.soft(item.first()).toBeVisible({ timeout: 5000 });
                });
            }

            // Check one detail page
            await test.step("network detail page", async () => {
                const searchInput = page.getByPlaceholder("Search...");
                await searchInput.fill("07-full-features_frontend");

                const item = page.locator(".item").filter({ hasText: "07-full-features_frontend" });
                await expect(item.first()).toBeVisible({ timeout: 5000 });
                await item.first().click();

                const overview = page.getByRole("region", { name: "Overview" });
                await expect(overview).toBeVisible({ timeout: 10000 });
                await expect.soft(overview.getByText("Driver")).toBeVisible();
                await expect.soft(overview.getByText("bridge")).toBeVisible();
            });
        });

        // ── Volumes tab ──
        await test.step("Volumes tab — volumes with correct detail", async () => {
            await page.goto("/volumes");
            await waitForApp(page);
            await expect(page.locator(".item").first()).toBeVisible({ timeout: 10000 });

            // Verify compose-derived volumes exist. The mock creates volumes for
            // stacks with non-null volume values; 05-multi-service uses `null` values
            // which the mock parser skips.
            const volumesToCheck = [
                "04-database_pgdata",
                "07-full-features_db-data",
                "07-full-features_search-data",
                "07-full-features_web-data",
                "08-env-config_dbdata",
            ];

            for (const volName of volumesToCheck) {
                await test.step(`volume: ${volName}`, async () => {
                    const searchInput = page.getByPlaceholder("Search...");
                    await searchInput.fill(volName);

                    const item = page.locator(".item").filter({ hasText: volName });
                    await expect.soft(item.first()).toBeVisible({ timeout: 5000 });
                });
            }

            // Check one detail page
            await test.step("volume detail page", async () => {
                const searchInput = page.getByPlaceholder("Search...");
                await searchInput.fill("04-database_pgdata");

                const item = page.locator(".item").filter({ hasText: "04-database_pgdata" });
                await expect(item.first()).toBeVisible({ timeout: 5000 });
                await item.first().click();

                const overview = page.getByRole("region", { name: "Overview" });
                await expect(overview).toBeVisible({ timeout: 10000 });
                await expect.soft(overview.getByText("Driver")).toBeVisible();
                await expect.soft(overview.getByText("Scope")).toBeVisible();
            });
        });
    });
});
