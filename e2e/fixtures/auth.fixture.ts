import { test as base, Page } from "@playwright/test";
import { disableAnimations } from "../helpers/disable-animations";
import { PerfCollector } from "../helpers/perf-collector";

/**
 * Extended test fixture that:
 * 1. Disables CSS animations on every page load (for deterministic screenshots)
 * 2. Runs a worker-scoped PerfCollector that resets the server-side memory tracker and records WebSocket frames
 *
 * Use this instead of importing `test` from @playwright/test in spec files.
 */
export const test = base.extend<
    { page: Page },
    { perfCollector: PerfCollector }
>({
    perfCollector: [async ({}, use) => {
        const collector = new PerfCollector();
        await collector.resetMemoryBaseline();
        await use(collector);
    }, { scope: "worker" }],

    page: async ({ page, perfCollector }, use, testInfo) => {
        // Disable animations on initial load and after every navigation
        page.on("load", async () => {
            try {
                await disableAnimations(page);
            } catch {
                // Page might have navigated away; ignore
            }
        });

        const testName = testInfo.titlePath.join(" > ");
        perfCollector.beginTest(testName);

        // Track clientWarning frames — any occurrence fails the test
        const clientWarnings: string[] = [];

        // Intercept WebSocket frames for socket tracking
        page.on("websocket", (ws) => {
            perfCollector.recordNewConnection(testName);

            ws.on("framereceived", (frame) => {
                if (typeof frame.payload === "string") {
                    perfCollector.recordServerFrame(testName, frame.payload);
                }
            });
            ws.on("framesent", (frame) => {
                if (typeof frame.payload === "string") {
                    perfCollector.recordClientFrame(testName, frame.payload);
                    if (frame.payload.includes('"clientWarning"')) {
                        clientWarnings.push(frame.payload);
                    }
                }
            });
        });

        await use(page);

        // Fail the test if any clientWarning was sent
        if (clientWarnings.length > 0) {
            throw new Error(`clientWarning during "${testName}": ${clientWarnings[0]}`);
        }

        perfCollector.endTest(testName);
    },
});

export { expect } from "@playwright/test";
