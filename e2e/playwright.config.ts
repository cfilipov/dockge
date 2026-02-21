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

export default defineConfig({
    testDir: "./tests",
    snapshotDir: "./__screenshots__",
    snapshotPathTemplate: "{snapshotDir}/{arg}{ext}",
    fullyParallel: false,
    forbidOnly: !!process.env.CI,
    retries: process.env.CI ? 2 : 0,
    workers: 1,
    outputDir: "./test-results",
    reporter: [["html", { outputFolder: "./playwright-report" }]],
    use: {
        baseURL: "http://localhost:5001",
        trace: "on-first-retry",
        viewport: { width: 1280, height: 720 },
    },
    expect: {
        toHaveScreenshot: {
            maxDiffPixelRatio: 0.005,
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
                storageState: "e2e/.auth/user.json",
            },
            dependencies: ["setup"],
        },
    ],
    webServer: {
        command: "cd ../backend-go && go build -o dockge-backend . && ./dockge-backend --dev --mock --port 5001 --stacks-dir test-data/stacks",
        port: 5001,
        reuseExistingServer: !process.env.CI,
    },
});
