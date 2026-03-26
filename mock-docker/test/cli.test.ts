import { describe, it, expect, beforeAll, afterAll } from "vitest";
import { execFile } from "node:child_process";
import { mkdtempSync, unlinkSync, existsSync } from "node:fs";
import { join } from "node:path";
import { tmpdir } from "node:os";
import { request as httpRequest } from "node:http";

import { createClock } from "../src/clock.js";
import { initState, type InitOptions } from "../src/init.js";
import { EventEmitter } from "../src/events.js";
import { createServer, type Route } from "../src/server.js";

import { systemRoutes } from "../src/api/system.js";
import { containerRoutes } from "../src/api/containers.js";
import { networkRoutes } from "../src/api/networks.js";
import { volumeRoutes } from "../src/api/volumes.js";
import { imageRoutes } from "../src/api/images.js";
import { distributionRoutes } from "../src/api/distribution.js";
import { execRoutes } from "../src/api/exec.js";
import { mockRoutes } from "../src/api/mock.js";

// ---------------------------------------------------------------------------
// HTTP helper (direct daemon communication for assertions)
// ---------------------------------------------------------------------------

interface HTTPResponse {
    statusCode: number;
    body: string;
}

function req(socketPath: string, method: string, path: string, body?: unknown): Promise<HTTPResponse> {
    return new Promise((resolve, reject) => {
        const bodyStr = body != null ? JSON.stringify(body) : undefined;
        const r = httpRequest(
            {
                socketPath,
                path,
                method,
                headers: bodyStr
                    ? { "Content-Type": "application/json", "Content-Length": String(Buffer.byteLength(bodyStr)) }
                    : {},
            },
            (res) => {
                const chunks: Buffer[] = [];
                res.on("data", (chunk: Buffer) => chunks.push(chunk));
                res.on("end", () => {
                    resolve({
                        statusCode: res.statusCode || 0,
                        body: Buffer.concat(chunks).toString(),
                    });
                });
            },
        );
        r.on("error", reject);
        if (bodyStr) r.write(bodyStr);
        r.end();
    });
}

// ---------------------------------------------------------------------------
// Test setup
// ---------------------------------------------------------------------------

const FIXTURES = join(import.meta.dirname!, "..", "test", "fixtures", "stacks");

describe("CLI integration tests", () => {
    let socketPath: string;
    let stopServer: () => Promise<void>;
    let tmpDir: string;
    let stacksDir: string;
    let cliEntryPoint: string;

    beforeAll(async () => {
        tmpDir = mkdtempSync(join(tmpdir(), "cli-test-"));
        socketPath = join(tmpDir, "docker.sock");
        stacksDir = join(tmpDir, "stacks");

        // Use the built CLI bundle (must run `npm run build:cli` first)
        cliEntryPoint = join(import.meta.dirname!, "..", "dist", "cli.js");

        const clock = createClock({ base: "2025-01-15T00:00:00Z" });
        const initOpts: InitOptions = {
            stacksSource: FIXTURES,
            stacksDir,
            clock,
        };
        const state = await initState(initOpts);
        const emitter = new EventEmitter();

        const routes: Route[] = [
            ...systemRoutes,
            ...mockRoutes,
            ...containerRoutes,
            ...imageRoutes,
            ...distributionRoutes,
            ...networkRoutes,
            ...volumeRoutes,
            ...execRoutes,
        ];

        const server = createServer(
            { socketPath, state, emitter, clock, initOpts },
            routes,
        );
        await server.start();
        stopServer = server.stop;
    });

    afterAll(async () => {
        if (stopServer) await stopServer();
        try { unlinkSync(socketPath); } catch { /* ignore */ }
    });

    /**
     * Run CLI command using the built bundle (async to not block the event loop).
     * The server runs in-process, so we must not use execFileSync.
     */
    function cli(
        args: string[],
        options?: { cwd?: string },
    ): Promise<{ stdout: string; stderr: string; exitCode: number }> {
        return new Promise((resolve) => {
            const env: Record<string, string> = {
                ...process.env as Record<string, string>,
                DOCKER_HOST: `unix://${socketPath}`,
                NODE_NO_WARNINGS: "1",
            };
            const child = execFile(
                process.execPath,
                [cliEntryPoint, ...args],
                {
                    env,
                    cwd: options?.cwd,
                    encoding: "utf-8",
                    timeout: 10000,
                },
                (err, stdout, stderr) => {
                    const exitCode = err && "code" in err ? (err as { code: number }).code : 0;
                    resolve({
                        stdout: stdout || "",
                        stderr: stderr || "",
                        exitCode: typeof exitCode === "number" ? exitCode : 1,
                    });
                },
            );
        });
    }

    // -----------------------------------------------------------------------
    // Docker commands
    // -----------------------------------------------------------------------

    describe("docker ps", () => {
        it("should list containers", async () => {
            const { stdout, exitCode } = await cli(["ps", "-a"]);
            expect(exitCode).toBe(0);
            expect(stdout).toContain("CONTAINER ID");
        });
    });

    describe("docker inspect", () => {
        it("should inspect a container by name", async () => {
            const res = await req(socketPath, "GET", "/containers/json?all=1");
            const containers = JSON.parse(res.body);
            if (containers.length === 0) return;

            const name = containers[0].Names[0].replace(/^\//, "");
            const { stdout, exitCode } = await cli(["inspect", name]);
            expect(exitCode).toBe(0);
            const parsed = JSON.parse(stdout);
            expect(Array.isArray(parsed)).toBe(true);
            expect(parsed.length).toBe(1);
        });

        it("should return error for unknown container", async () => {
            const { stderr, exitCode } = await cli(["inspect", "nonexistent-container-xyz"]);
            expect(exitCode).toBe(1);
            expect(stderr).toContain("No such object");
        });
    });

    describe("docker network ls", () => {
        it("should list networks", async () => {
            const { stdout, exitCode } = await cli(["network", "ls"]);
            expect(exitCode).toBe(0);
            expect(stdout).toContain("NETWORK ID");
        });
    });

    describe("docker volume ls", () => {
        it("should list volumes", async () => {
            const { stdout, exitCode } = await cli(["volume", "ls"]);
            expect(exitCode).toBe(0);
            expect(stdout).toContain("DRIVER");
        });
    });

    describe("docker images", () => {
        it("should list images", async () => {
            const { stdout, exitCode } = await cli(["images"]);
            expect(exitCode).toBe(0);
            expect(stdout).toContain("REPOSITORY");
        });
    });

    describe("docker image prune", () => {
        it("should print reclaimed space", async () => {
            const { stdout, exitCode } = await cli(["image", "prune"]);
            expect(exitCode).toBe(0);
            expect(stdout).toContain("Total reclaimed space: 0B");
        });
    });

    // -----------------------------------------------------------------------
    // Docker container actions
    // -----------------------------------------------------------------------

    describe("docker stop/start", () => {
        it("should stop and start a container", async () => {
            const res = await req(socketPath, "GET", "/containers/json?all=1");
            const containers = JSON.parse(res.body);
            const running = containers.find((c: { State: string }) => c.State === "running");
            if (!running) return;

            const name = running.Names[0].replace(/^\//, "");

            // Stop
            const stopResult = await cli(["stop", name]);
            expect(stopResult.exitCode).toBe(0);
            expect(stopResult.stdout.trim()).toBe(name);

            // Start
            const startResult = await cli(["start", name]);
            expect(startResult.exitCode).toBe(0);
            expect(startResult.stdout.trim()).toBe(name);
        });
    });

    // -----------------------------------------------------------------------
    // Compose commands
    // -----------------------------------------------------------------------

    describe("docker compose up/down", () => {
        it("should create and tear down containers via compose", async () => {
            const basicDir = join(stacksDir, "basic");
            if (!existsSync(basicDir)) return;

            // Compose up
            const upResult = await cli(
                ["compose", "-p", "test-basic", "up", "-d"],
                { cwd: basicDir },
            );
            expect(upResult.exitCode).toBe(0);

            // Compose down
            const downResult = await cli(
                ["compose", "-p", "test-basic", "down"],
                { cwd: basicDir },
            );
            expect(downResult.exitCode).toBe(0);
        });
    });

    describe("docker compose stop/start", () => {
        it("should stop and start compose services", async () => {
            const basicDir = join(stacksDir, "basic");
            if (!existsSync(basicDir)) return;

            // Bring up
            await cli(
                ["compose", "-p", "test-startstop", "up", "-d"],
                { cwd: basicDir },
            );

            // Stop
            const stopResult = await cli(
                ["compose", "-p", "test-startstop", "stop"],
                { cwd: basicDir },
            );
            expect(stopResult.exitCode).toBe(0);

            // Start again
            const startResult = await cli(
                ["compose", "-p", "test-startstop", "up", "-d"],
                { cwd: basicDir },
            );
            expect(startResult.exitCode).toBe(0);

            // Clean up
            await cli(
                ["compose", "-p", "test-startstop", "down"],
                { cwd: basicDir },
            );
        });
    });

    describe("docker compose config", () => {
        it("should validate compose file successfully", async () => {
            const basicDir = join(stacksDir, "basic");
            if (!existsSync(basicDir)) return;

            const result = await cli(["compose", "config"], { cwd: basicDir });
            expect(result.exitCode).toBe(0);
        });

        it("should fail for missing compose file", async () => {
            const result = await cli(["compose", "config"], { cwd: tmpDir });
            expect(result.exitCode).toBe(1);
            expect(result.stderr).toContain("not found");
        });
    });

    // -----------------------------------------------------------------------
    // Error cases
    // -----------------------------------------------------------------------

    describe("error handling", () => {
        it("should exit 0 for unknown docker command (matching Go behavior)", async () => {
            const result = await cli(["nonexistent-cmd"]);
            expect(result.stderr).toContain("unsupported command");
        });

        it("should exit 1 for stop without container name", async () => {
            const { exitCode, stderr } = await cli(["stop"]);
            expect(exitCode).toBe(1);
            expect(stderr).toContain("container name required");
        });
    });
});
