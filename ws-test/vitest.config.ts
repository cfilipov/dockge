import { defineConfig } from "vitest/config";

export default defineConfig({
    test: {
        globalSetup: "./src/setup.ts",
        testTimeout: 15_000,
        hookTimeout: 30_000,
        fileParallelism: false,
        sequence: { concurrent: false },
        include: ["tests/**/*.test.ts"],
    },
});
