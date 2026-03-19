import { describe, it, expect, beforeAll, afterAll, beforeEach } from "vitest";
import { request as httpRequest, type IncomingMessage } from "node:http";
import { mkdtempSync, mkdirSync, cpSync, existsSync } from "node:fs";
import { join } from "node:path";
import { tmpdir } from "node:os";

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
// Test helper: HTTP client over Unix socket
// ---------------------------------------------------------------------------

interface HTTPResponse {
    statusCode: number;
    headers: IncomingMessage["headers"];
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
                    ? { "Content-Type": "application/json", "Content-Length": Buffer.byteLength(bodyStr) }
                    : {},
            },
            (res) => {
                const chunks: Buffer[] = [];
                res.on("data", (chunk: Buffer) => chunks.push(chunk));
                res.on("end", () => {
                    resolve({
                        statusCode: res.statusCode || 0,
                        headers: res.headers,
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

function json(r: HTTPResponse): unknown {
    return JSON.parse(r.body);
}

// ---------------------------------------------------------------------------
// Test setup
// ---------------------------------------------------------------------------

const FIXTURES = join(import.meta.dirname, "fixtures", "stacks");

let socketPath: string;
let stopServer: () => Promise<void>;
let initOpts: InitOptions;

// Build full route list
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

beforeAll(async () => {
    const tmpDir = mkdtempSync(join(tmpdir(), "mock-docker-test-"));
    socketPath = join(tmpDir, "docker.sock");
    const stacksDir = join(tmpDir, "stacks");
    mkdirSync(stacksDir, { recursive: true });

    const clock = createClock({ fixed: true, base: "2025-01-15T00:00:00Z" });
    initOpts = { stacksSource: FIXTURES, stacksDir, clock };
    const state = await initState(initOpts);
    const emitter = new EventEmitter();

    const srv = createServer({ socketPath, state, emitter, clock, initOpts }, routes);
    await srv.start();
    stopServer = srv.stop;
});

afterAll(async () => {
    await stopServer();
});

// ---------------------------------------------------------------------------
// System endpoints
// ---------------------------------------------------------------------------

describe("system", () => {
    it("GET /_ping returns OK with API-Version header", async () => {
        const r = await req(socketPath, "GET", "/_ping");
        expect(r.statusCode).toBe(200);
        expect(r.body).toBe("OK");
        expect(r.headers["api-version"]).toBe("1.47");
        expect(r.headers["docker-experimental"]).toBe("false");
    });

    it("HEAD /_ping returns headers only, no body", async () => {
        const r = await req(socketPath, "HEAD", "/_ping");
        expect(r.statusCode).toBe(200);
        expect(r.body).toBe("");
        expect(r.headers["api-version"]).toBe("1.47");
    });

    it("GET /version returns version JSON", async () => {
        const r = await req(socketPath, "GET", "/version");
        expect(r.statusCode).toBe(200);
        const body = json(r) as Record<string, unknown>;
        expect(body.ApiVersion).toBe("1.47");
        expect(body.Version).toBe("27.5.1");
        expect(body.Os).toBe("linux");
    });

    it("GET /info returns container/image counts", async () => {
        const r = await req(socketPath, "GET", "/info");
        expect(r.statusCode).toBe(200);
        const body = json(r) as Record<string, unknown>;
        expect(typeof body.Containers).toBe("number");
        expect(typeof body.Images).toBe("number");
        expect(typeof body.ContainersRunning).toBe("number");
    });

    it("GET /events streams and receives emitted events", async () => {
        const received: string[] = [];
        const request = httpRequest(
            { socketPath, path: "/events", method: "GET" },
            (res) => {
                expect(res.statusCode).toBe(200);
                expect(res.headers["transfer-encoding"]).toBe("chunked");
                res.on("data", (chunk: Buffer) => {
                    received.push(chunk.toString());
                });
            },
        );
        request.end();

        // Give time for the connection to establish
        await new Promise((r) => setTimeout(r, 50));

        // Emit an event by stopping/starting a container
        const listR = await req(socketPath, "GET", "/containers/json");
        const list = json(listR) as Array<{ Id: string }>;
        if (list.length > 0) {
            await req(socketPath, "POST", `/containers/${list[0].Id}/stop`);
            await new Promise((r) => setTimeout(r, 50));
            await req(socketPath, "POST", `/containers/${list[0].Id}/start`);
            await new Promise((r) => setTimeout(r, 50));
        }

        request.destroy();

        // Should have received some events
        if (list.length > 0) {
            expect(received.length).toBeGreaterThan(0);
            // Events are newline-delimited JSON
            const firstEvent = JSON.parse(received[0].split("\n")[0]);
            expect(firstEvent.Type).toBe("container");
        }
    });
});

// ---------------------------------------------------------------------------
// Version prefix stripping
// ---------------------------------------------------------------------------

describe("version prefix", () => {
    it("GET /v1.47/containers/json works with version prefix", async () => {
        const r = await req(socketPath, "GET", "/v1.47/containers/json");
        expect(r.statusCode).toBe(200);
        const body = json(r) as unknown[];
        expect(Array.isArray(body)).toBe(true);
    });

    it("GET /v1.45/_ping works with older version prefix", async () => {
        const r = await req(socketPath, "GET", "/v1.45/_ping");
        expect(r.statusCode).toBe(200);
        expect(r.body).toBe("OK");
    });
});

// ---------------------------------------------------------------------------
// Container endpoints
// ---------------------------------------------------------------------------

describe("containers", () => {
    it("GET /containers/json returns running containers by default", async () => {
        const r = await req(socketPath, "GET", "/containers/json");
        expect(r.statusCode).toBe(200);
        const body = json(r) as Array<Record<string, unknown>>;
        expect(Array.isArray(body)).toBe(true);
        // All returned containers should be running
        for (const c of body) {
            expect(c.State).toBe("running");
        }
    });

    it("GET /containers/json?all=1 returns all containers", async () => {
        const r = await req(socketPath, "GET", "/containers/json?all=1");
        expect(r.statusCode).toBe(200);
        const body = json(r) as unknown[];
        expect(Array.isArray(body)).toBe(true);
        // Should include more or equal containers than running-only
        const runningR = await req(socketPath, "GET", "/containers/json");
        const running = json(runningR) as unknown[];
        expect(body.length).toBeGreaterThanOrEqual(running.length);
    });

    it("GET /containers/json with label filter", async () => {
        const filters = JSON.stringify({ label: ["com.docker.compose.project=basic"] });
        const r = await req(socketPath, "GET", `/containers/json?all=1&filters=${encodeURIComponent(filters)}`);
        expect(r.statusCode).toBe(200);
        const body = json(r) as Array<Record<string, unknown>>;
        for (const c of body) {
            const labels = c.Labels as Record<string, string>;
            expect(labels["com.docker.compose.project"]).toBe("basic");
        }
    });

    it("GET /containers/:id/json returns full inspect for valid container", async () => {
        // First get a container ID
        const listR = await req(socketPath, "GET", "/containers/json?all=1");
        const list = json(listR) as Array<{ Id: string }>;
        expect(list.length).toBeGreaterThan(0);

        const id = list[0].Id;
        const r = await req(socketPath, "GET", `/containers/${id}/json`);
        expect(r.statusCode).toBe(200);
        const body = json(r) as Record<string, unknown>;
        expect(body.Id).toBe(id);
    });

    it("GET /containers/:id/json returns 404 for unknown container", async () => {
        const r = await req(socketPath, "GET", "/containers/nonexistent/json");
        expect(r.statusCode).toBe(404);
        const body = json(r) as Record<string, unknown>;
        expect(body.message).toBeTruthy();
    });

    it("POST /containers/:id/stop changes state", async () => {
        // Find a running container
        const listR = await req(socketPath, "GET", "/containers/json");
        const list = json(listR) as Array<{ Id: string }>;
        if (list.length === 0) return; // Skip if no running containers

        const id = list[0].Id;
        const stopR = await req(socketPath, "POST", `/containers/${id}/stop`);
        expect(stopR.statusCode).toBe(204);

        // Verify it's stopped
        const inspR = await req(socketPath, "GET", `/containers/${id}/json`);
        const insp = json(inspR) as { State: { Running: boolean } };
        expect(insp.State.Running).toBe(false);

        // Restart it for other tests
        await req(socketPath, "POST", `/containers/${id}/start`);
    });

    it("POST /containers/create returns 201 with Id", async () => {
        const r = await req(socketPath, "POST", "/containers/create?name=test-new", {
            Image: "alpine:latest",
            Cmd: ["echo", "hello"],
        });
        expect(r.statusCode).toBe(201);
        const body = json(r) as { Id: string; Warnings: unknown[] };
        expect(body.Id).toBeTruthy();
        expect(body.Warnings).toEqual([]);

        // Clean up
        await req(socketPath, "DELETE", `/containers/${body.Id}?force=1`);
    });

    it("DELETE /containers/:id returns 204", async () => {
        // Create a container to delete
        const createR = await req(socketPath, "POST", "/containers/create?name=delete-me", {
            Image: "alpine:latest",
        });
        const { Id } = json(createR) as { Id: string };

        const r = await req(socketPath, "DELETE", `/containers/${Id}`);
        expect(r.statusCode).toBe(204);

        // Verify it's gone
        const inspR = await req(socketPath, "GET", `/containers/${Id}/json`);
        expect(inspR.statusCode).toBe(404);
    });

    it("DELETE running container without force returns 409", async () => {
        // Create and start a container
        const createR = await req(socketPath, "POST", "/containers/create?name=cant-delete", {
            Image: "alpine:latest",
        });
        const { Id } = json(createR) as { Id: string };
        await req(socketPath, "POST", `/containers/${Id}/start`);

        const r = await req(socketPath, "DELETE", `/containers/${Id}`);
        expect(r.statusCode).toBe(409);

        // Clean up
        await req(socketPath, "DELETE", `/containers/${Id}?force=1`);
    });

    it("POST /containers/:id/update stub returns 200", async () => {
        const listR = await req(socketPath, "GET", "/containers/json?all=1");
        const list = json(listR) as Array<{ Id: string }>;
        if (list.length === 0) return;
        const r = await req(socketPath, "POST", `/containers/${list[0].Id}/update`);
        expect(r.statusCode).toBe(200);
    });

    it("POST /containers/:id/exec creates exec session", async () => {
        const listR = await req(socketPath, "GET", "/containers/json");
        const list = json(listR) as Array<{ Id: string }>;
        if (list.length === 0) return;
        const id = list[0].Id;

        const execR = await req(socketPath, "POST", `/containers/${id}/exec`, {
            Cmd: ["echo", "hello"],
            AttachStdout: true,
            AttachStderr: true,
        });
        expect(execR.statusCode).toBe(201);
        const body = json(execR) as { Id: string };
        expect(body.Id).toBeTruthy();
        expect(body.Id.length).toBe(64);
    });

    it("GET /exec/:id/json returns exec inspect", async () => {
        const listR = await req(socketPath, "GET", "/containers/json");
        const list = json(listR) as Array<{ Id: string }>;
        if (list.length === 0) return;
        const id = list[0].Id;

        const execR = await req(socketPath, "POST", `/containers/${id}/exec`, {
            Cmd: ["echo", "test"],
            AttachStdout: true,
        });
        const { Id: execId } = json(execR) as { Id: string };

        const inspectR = await req(socketPath, "GET", `/exec/${execId}/json`);
        expect(inspectR.statusCode).toBe(200);
        const inspect = json(inspectR) as { ID: string; ContainerID: string; Running: boolean };
        expect(inspect.ID).toBe(execId);
        expect(inspect.ContainerID).toBe(id);
        expect(inspect.Running).toBe(false);
    });

    it("GET /exec/:id/json returns 404 for unknown ID", async () => {
        const r = await req(socketPath, "GET", "/exec/nonexistent/json");
        expect(r.statusCode).toBe(404);
    });

    it("POST /exec/:id/start with one-shot command returns output", async () => {
        const listR = await req(socketPath, "GET", "/containers/json");
        const list = json(listR) as Array<{ Id: string }>;
        if (list.length === 0) return;
        const id = list[0].Id;

        // Create exec with echo command
        const execR = await req(socketPath, "POST", `/containers/${id}/exec`, {
            Cmd: ["echo", "hello", "world"],
            AttachStdout: true,
            AttachStderr: true,
        });
        const { Id: execId } = json(execR) as { Id: string };

        // Start exec
        const startR = await req(socketPath, "POST", `/exec/${execId}/start`, {
            Detach: false,
            Tty: false,
        });
        expect(startR.statusCode).toBe(200);
        // The output should contain "hello world" in multiplexed stream format
        expect(startR.body).toContain("hello world");
    });

    it("GET /containers/:id/logs returns log data", async () => {
        const listR = await req(socketPath, "GET", "/containers/json");
        const list = json(listR) as Array<{ Id: string }>;
        if (list.length === 0) return;
        const id = list[0].Id;

        const r = await req(socketPath, "GET", `/containers/${id}/logs?tail=5`);
        expect(r.statusCode).toBe(200);
        // Should have content (multiplexed framing makes body non-empty)
        expect(r.body.length).toBeGreaterThan(0);
    });

    it("GET /containers/:id/logs returns 404 for unknown", async () => {
        const r = await req(socketPath, "GET", "/containers/nonexistent/logs");
        expect(r.statusCode).toBe(404);
    });

    it("GET /containers/:id/stats?stream=false returns single JSON", async () => {
        const listR = await req(socketPath, "GET", "/containers/json");
        const list = json(listR) as Array<{ Id: string }>;
        if (list.length === 0) return;
        const id = list[0].Id;

        const r = await req(socketPath, "GET", `/containers/${id}/stats?stream=false`);
        expect(r.statusCode).toBe(200);
        const body = json(r) as Record<string, unknown>;
        expect(body.cpu_stats).toBeDefined();
        expect(body.memory_stats).toBeDefined();
        expect(body.networks).toBeDefined();
    });

    it("GET /containers/:id/stats returns 409 for stopped container", async () => {
        // Find a stopped container or create one
        const listR = await req(socketPath, "GET", "/containers/json?all=1");
        const all = json(listR) as Array<{ Id: string; State: string }>;
        const stopped = all.find((c) => c.State !== "running");
        if (!stopped) return; // skip if all running

        const r = await req(socketPath, "GET", `/containers/${stopped.Id}/stats?stream=false`);
        expect(r.statusCode).toBe(409);
    });

    it("GET /containers/:id/top returns process list", async () => {
        const listR = await req(socketPath, "GET", "/containers/json");
        const list = json(listR) as Array<{ Id: string }>;
        if (list.length === 0) return;
        const id = list[0].Id;

        const r = await req(socketPath, "GET", `/containers/${id}/top`);
        expect(r.statusCode).toBe(200);
        const body = json(r) as { Titles: string[]; Processes: string[][] };
        expect(body.Titles).toContain("PID");
        expect(body.Titles).toContain("CMD");
        expect(body.Processes.length).toBeGreaterThan(0);
        // PID 1 should be first
        expect(body.Processes[0][1]).toBe("1");
    });

    it("GET /containers/:id/top returns 500 for stopped container", async () => {
        const listR = await req(socketPath, "GET", "/containers/json?all=1");
        const all = json(listR) as Array<{ Id: string; State: string }>;
        const stopped = all.find((c) => c.State !== "running");
        if (!stopped) return;

        const r = await req(socketPath, "GET", `/containers/${stopped.Id}/top`);
        expect(r.statusCode).toBe(500);
    });
});

// ---------------------------------------------------------------------------
// Network endpoints
// ---------------------------------------------------------------------------

describe("networks", () => {
    it("GET /networks returns array", async () => {
        const r = await req(socketPath, "GET", "/networks");
        expect(r.statusCode).toBe(200);
        const body = json(r) as unknown[];
        expect(Array.isArray(body)).toBe(true);
    });

    it("POST /networks/create returns 201 with Id", async () => {
        const r = await req(socketPath, "POST", "/networks/create", {
            Name: "test-network",
            Driver: "bridge",
        });
        expect(r.statusCode).toBe(201);
        const body = json(r) as { Id: string };
        expect(body.Id).toBeTruthy();

        // Clean up
        await req(socketPath, "DELETE", `/networks/${body.Id}`);
    });

    it("GET /networks/:id returns network by ID", async () => {
        const createR = await req(socketPath, "POST", "/networks/create", { Name: "inspect-net" });
        const { Id } = json(createR) as { Id: string };

        const r = await req(socketPath, "GET", `/networks/${Id}`);
        expect(r.statusCode).toBe(200);
        const body = json(r) as { Id: string; Name: string };
        expect(body.Name).toBe("inspect-net");

        await req(socketPath, "DELETE", `/networks/${Id}`);
    });

    it("DELETE /networks/:id returns 204", async () => {
        const createR = await req(socketPath, "POST", "/networks/create", { Name: "delete-net" });
        const { Id } = json(createR) as { Id: string };

        const r = await req(socketPath, "DELETE", `/networks/${Id}`);
        expect(r.statusCode).toBe(204);
    });

    it("GET /networks/:id returns 404 for unknown", async () => {
        const r = await req(socketPath, "GET", "/networks/nonexistent");
        expect(r.statusCode).toBe(404);
    });
});

// ---------------------------------------------------------------------------
// Volume endpoints
// ---------------------------------------------------------------------------

describe("volumes", () => {
    it("GET /volumes returns wrapped response", async () => {
        const r = await req(socketPath, "GET", "/volumes");
        expect(r.statusCode).toBe(200);
        const body = json(r) as { Volumes: unknown[]; Warnings: unknown[] };
        expect(Array.isArray(body.Volumes)).toBe(true);
        expect(body.Warnings).toEqual([]);
    });

    it("POST /volumes/create returns 201", async () => {
        const r = await req(socketPath, "POST", "/volumes/create", {
            Name: "test-vol",
            Driver: "local",
        });
        expect(r.statusCode).toBe(201);
        const body = json(r) as { Name: string };
        expect(body.Name).toBe("test-vol");

        // Clean up
        await req(socketPath, "DELETE", "/volumes/test-vol");
    });

    it("GET /volumes/:name returns volume", async () => {
        await req(socketPath, "POST", "/volumes/create", { Name: "get-vol" });
        const r = await req(socketPath, "GET", "/volumes/get-vol");
        expect(r.statusCode).toBe(200);
        const body = json(r) as { Name: string };
        expect(body.Name).toBe("get-vol");

        await req(socketPath, "DELETE", "/volumes/get-vol");
    });

    it("GET /volumes/:name returns 404 for unknown", async () => {
        const r = await req(socketPath, "GET", "/volumes/nonexistent");
        expect(r.statusCode).toBe(404);
    });

    it("DELETE /volumes/:name returns 204", async () => {
        await req(socketPath, "POST", "/volumes/create", { Name: "del-vol" });
        const r = await req(socketPath, "DELETE", "/volumes/del-vol");
        expect(r.statusCode).toBe(204);
    });
});

// ---------------------------------------------------------------------------
// Image endpoints
// ---------------------------------------------------------------------------

describe("images", () => {
    it("GET /images/json returns projected entries", async () => {
        const r = await req(socketPath, "GET", "/images/json");
        expect(r.statusCode).toBe(200);
        const body = json(r) as Array<Record<string, unknown>>;
        expect(Array.isArray(body)).toBe(true);
        if (body.length > 0) {
            expect(body[0]).toHaveProperty("Id");
            expect(body[0]).toHaveProperty("RepoTags");
        }
    });

    it("GET /images/<name>/json strips /json suffix from greedy capture", async () => {
        // nginx:latest is in the basic fixture
        const r = await req(socketPath, "GET", "/images/nginx:latest/json");
        expect(r.statusCode).toBe(200);
        const body = json(r) as { RepoTags: string[] };
        expect(body.RepoTags).toContain("nginx:latest");
    });

    it("GET /images/<name>/json returns 404 for unknown image", async () => {
        const r = await req(socketPath, "GET", "/images/no-such-image:v999/json");
        expect(r.statusCode).toBe(404);
    });
});

// ---------------------------------------------------------------------------
// Distribution endpoint
// ---------------------------------------------------------------------------

describe("distribution", () => {
    it("GET /distribution/<name>/json returns descriptor for known image", async () => {
        const r = await req(socketPath, "GET", "/distribution/nginx:latest/json");
        expect(r.statusCode).toBe(200);
        const body = json(r) as { Descriptor: { digest: string; size: number }; Platforms: unknown[] };
        expect(body.Descriptor.digest).toBeTruthy();
        expect(body.Platforms.length).toBeGreaterThan(0);
    });

    it("GET /distribution/<name>/json returns 404 for unknown image", async () => {
        const r = await req(socketPath, "GET", "/distribution/no-such-image:v999/json");
        expect(r.statusCode).toBe(404);
    });

    it("returns altered digest for image with update_available in global config", async () => {
        // postgres:16 has update_available: true in global .mock.yaml
        const r = await req(socketPath, "GET", "/distribution/postgres:16/json");
        expect(r.statusCode).toBe(200);
        const body = json(r) as { Descriptor: { digest: string } };

        // Also get nginx:latest which does NOT have update_available
        const r2 = await req(socketPath, "GET", "/distribution/nginx:latest/json");
        const body2 = json(r2) as { Descriptor: { digest: string } };

        // Both should have digests
        expect(body.Descriptor.digest).toBeTruthy();
        expect(body2.Descriptor.digest).toBeTruthy();

        // The postgres digest should differ from its image's own digest
        // (because update_available alters it)
        const imgR = await req(socketPath, "GET", "/images/postgres:16/json");
        const img = json(imgR) as { RepoDigests: string[] };
        const originalDigest = img.RepoDigests[0]?.split("@")[1];
        expect(body.Descriptor.digest).not.toBe(originalDigest);
    });
});

// ---------------------------------------------------------------------------
// Mock reset
// ---------------------------------------------------------------------------

describe("mock reset", () => {
    it("POST /_mock/reset re-initializes state", async () => {
        // Get initial container count
        const beforeR = await req(socketPath, "GET", "/containers/json?all=1");
        const beforeCount = (json(beforeR) as unknown[]).length;

        // Create extra container
        await req(socketPath, "POST", "/containers/create?name=temp-reset-test", {
            Image: "alpine:latest",
        });
        const afterCreateR = await req(socketPath, "GET", "/containers/json?all=1");
        const afterCreateCount = (json(afterCreateR) as unknown[]).length;
        expect(afterCreateCount).toBe(beforeCount + 1);

        // Reset
        const resetR = await req(socketPath, "POST", "/_mock/reset");
        expect(resetR.statusCode).toBe(200);
        const body = json(resetR) as { status: string };
        expect(body.status).toBe("ok");

        // Count should be back to initial
        const afterResetR = await req(socketPath, "GET", "/containers/json?all=1");
        const afterResetCount = (json(afterResetR) as unknown[]).length;
        expect(afterResetCount).toBe(beforeCount);
    });
});

// ---------------------------------------------------------------------------
// E2E mode
// ---------------------------------------------------------------------------

describe("e2e mode", () => {
    let e2eSocketPath: string;
    let stopE2eServer: () => Promise<void>;

    beforeAll(async () => {
        const tmpDir = mkdtempSync(join(tmpdir(), "mock-docker-e2e-"));
        e2eSocketPath = join(tmpDir, "docker.sock");
        const stacksDir = join(tmpDir, "stacks");
        mkdirSync(stacksDir, { recursive: true });

        const clock = createClock({ fixed: true, base: "2025-01-15T00:00:00Z" });
        const e2eInitOpts: InitOptions = { stacksSource: FIXTURES, stacksDir, clock };
        const state = await initState(e2eInitOpts);
        const emitter = new EventEmitter();

        const srv = createServer(
            { socketPath: e2eSocketPath, state, emitter, clock, initOpts: e2eInitOpts, e2eMode: true },
            routes,
        );
        await srv.start();
        stopE2eServer = srv.stop;
    });

    afterAll(async () => {
        await stopE2eServer();
    });

    it("GET /containers/:id/logs?follow=true returns a finite response", async () => {
        const listR = await req(e2eSocketPath, "GET", "/containers/json");
        const list = json(listR) as Array<{ Id: string }>;
        expect(list.length).toBeGreaterThan(0);

        const id = list[0].Id;
        // This should resolve without hanging because E2E mode ends follow streams
        const r = await req(e2eSocketPath, "GET", `/containers/${id}/logs?follow=true`);
        expect(r.statusCode).toBe(200);
        expect(r.body.length).toBeGreaterThan(0);
    });

    it("GET /containers/:id/stats returns single JSON snapshot (no stream=false needed)", async () => {
        const listR = await req(e2eSocketPath, "GET", "/containers/json");
        const list = json(listR) as Array<{ Id: string }>;
        expect(list.length).toBeGreaterThan(0);

        const id = list[0].Id;
        // Without stream=false, in E2E mode it should still return a single snapshot
        const r = await req(e2eSocketPath, "GET", `/containers/${id}/stats`);
        expect(r.statusCode).toBe(200);
        const body = json(r) as Record<string, unknown>;
        expect(body.cpu_stats).toBeDefined();
        expect(body.memory_stats).toBeDefined();
    });
});

// ---------------------------------------------------------------------------
// 404 for unknown routes
// ---------------------------------------------------------------------------

describe("routing", () => {
    it("returns 404 for unknown routes", async () => {
        const r = await req(socketPath, "GET", "/totally/unknown/route");
        expect(r.statusCode).toBe(404);
        const body = json(r) as { message: string };
        expect(body.message).toBe("page not found");
    });
});
