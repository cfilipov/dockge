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
const authPort = port + 1;

// Tests that mutate mock state or test auth — must run sequentially.
const SERIAL_FILES = [
    "login.spec.ts",
    "logout.spec.ts",
    "session-invalidation.spec.ts",
    "change-password.spec.ts",
    "z-change-password-success.spec.ts",
    "compose-operations.spec.ts",
    "compose-deploy.spec.ts",
    "compose-delete.spec.ts",
    "compose-save-discard.spec.ts",
    "compose-deploy-invalid.spec.ts",
    "compose-save-env.spec.ts",
    "container-card-delete.spec.ts",
    "container-details.spec.ts",
    "unmanaged-actions.spec.ts",
    "settings.spec.ts",
    "check-image-updates.spec.ts",
    "zz-perf-benchmarks.spec.ts",
];

const sharedDeviceUse = {
    ...devices["Desktop Chrome"],
    viewport: { width: 1280, height: 720 } as const,
    colorScheme: "dark" as const,
};

export default defineConfig({
    testDir: "./tests",
    snapshotDir: "./__screenshots__",
    snapshotPathTemplate: "{snapshotDir}/{arg}{ext}",
    fullyParallel: false,
    forbidOnly: !!process.env.CI,
    retries: process.env.CI ? 2 : 0,
    workers: 1,
    outputDir: "../.e2e-output/test-results",
    reporter: [["html", { outputFolder: "../.e2e-output/playwright-report", open: "on-failure" }]],
    use: {
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
            use: { baseURL: `http://localhost:${authPort}` },
        },
        {
            name: "parallel",
            use: {
                ...sharedDeviceUse,
                baseURL: `http://localhost:${port}`,
                storageState: {
                    cookies: [],
                    origins: [{
                        origin: `http://localhost:${port}`,
                        localStorage: [{ name: "theme", value: "auto" }],
                    }],
                },
            },
            fullyParallel: true,
            testIgnore: SERIAL_FILES,
        },
        {
            name: "serial",
            use: {
                ...sharedDeviceUse,
                baseURL: `http://localhost:${authPort}`,
                storageState: "../.e2e-output/auth/user.json",
            },
            fullyParallel: false,
            dependencies: ["setup"],
            testMatch: SERIAL_FILES.map(f => new RegExp(f.replace(/\./g, "\\."))),
        },
    ],
    webServer: [
        {
            command: `cd .. && rm -rf .run/e2e-${port} && mkdir -p .run/e2e-${port}/data && task run:mock-docker-daemon PORT=${port} STACKS_DIR=.run/e2e-${port}/stacks DATA_DIR=.run/e2e-${port}/data EXTRA_FLAGS=--no-auth`,
            port,
            reuseExistingServer: !process.env.CI,
        },
        {
            command: `cd .. && rm -rf .run/e2e-${authPort} && mkdir -p .run/e2e-${authPort}/data && task run:mock-docker-daemon PORT=${authPort} STACKS_DIR=.run/e2e-${authPort}/stacks DATA_DIR=.run/e2e-${authPort}/data`,
            port: authPort,
            reuseExistingServer: !process.env.CI,
        },
    ],
});
