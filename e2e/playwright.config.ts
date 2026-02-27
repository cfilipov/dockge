import { defineConfig, devices } from "@playwright/test";
import { join } from "path";
import { homedir } from "os";

// In CI, npx playwright install --with-deps puts browsers in the default location.
// Locally, we install to ~/.cache/ms-playwright to avoid permission issues.
// The pnpm scripts set PLAYWRIGHT_BROWSERS_PATH explicitly; this is a fallback
// for direct npx invocations.
if (!process.env.CI) {
    process.env.PLAYWRIGHT_BROWSERS_PATH = join(homedir(), ".cache", "ms-playwright");
}

// Allow overriding the port so VSCode's persistent test-server (default 5051)
// and CLI runs (Taskfile sets 5052) don't conflict with each other.
const port = parseInt(process.env.E2E_PORT || "5051", 10);

export default defineConfig({
    testDir: "./tests",
    snapshotDir: "./__screenshots__",
    snapshotPathTemplate: "{snapshotDir}/{arg}{ext}",
    fullyParallel: false,
    forbidOnly: !!process.env.CI,
    retries: process.env.CI ? 2 : 0,
    workers: 1,
    outputDir: "./test-results",
    reporter: [["html", { outputFolder: "./playwright-report", open: "on-failure" }]],
    use: {
        baseURL: `http://localhost:${port}`,
        trace: "on-first-retry",
        viewport: { width: 1280, height: 720 },
    },
    expect: {
        toHaveScreenshot: {
            maxDiffPixelRatio: 0.001,
            animations: "disabled",
        },
    },
    projects: [
        {
            name: "setup",
            testMatch: /auth\.setup\.ts/,
        },
        {
            name: "chromium",
            use: {
                ...devices["Desktop Chrome"],
                viewport: { width: 1280, height: 720 },
                storageState: ".auth/user.json",
                colorScheme: "dark",
            },
            dependencies: ["setup"],
        },
    ],
    webServer: {
        command: `cd .. && go build -o dockge . && rm -rf test-data/e2e-stacks && cp -a test-data/stacks test-data/e2e-stacks && ./dockge --dev --mock --port ${port} --data-dir test-data/e2e-data-${port} --stacks-dir test-data/e2e-stacks`,
        port,
        reuseExistingServer: !process.env.CI,
    },
});
