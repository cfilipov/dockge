import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { mkdtempSync, rmSync, cpSync, writeFileSync, mkdirSync } from "node:fs";
import { join } from "node:path";
import { tmpdir } from "node:os";
import { initState } from "../src/init.js";
import { FixedClock } from "../src/clock.js";

const fixturesDir = join(import.meta.dirname, "fixtures/stacks");
const clock = new FixedClock(new Date("2025-01-01T00:00:00Z"));

let tempDir: string;

beforeEach(() => {
    tempDir = mkdtempSync(join(tmpdir(), "mock-docker-init-"));
});

afterEach(() => {
    rmSync(tempDir, { recursive: true, force: true });
});

describe("initState", () => {
    it("processes fixture stacks without error", async () => {
        const state = await initState({
            stacksSource: fixturesDir,
            stacksDir: tempDir,
            clock,
        });

        // Should have containers from deployed stacks (basic, multi-net, host-mode, with-volumes, with-overrides, env-file)
        // not-deployed should be skipped
        expect(state.containers.size).toBeGreaterThan(0);
        expect(state.networks.size).toBeGreaterThan(0);
        expect(state.images.size).toBeGreaterThan(0);
    });

    it("skips deployed: false stacks", async () => {
        const state = await initState({
            stacksSource: fixturesDir,
            stacksDir: tempDir,
            clock,
        });

        // No container should reference "not-deployed" project
        for (const c of state.containers.values()) {
            expect(c.Config.Labels!["com.docker.compose.project"]).not.toBe("not-deployed");
        }
    });

    it("populates NetworkInspect.Containers in post-processing", async () => {
        const state = await initState({
            stacksSource: fixturesDir,
            stacksDir: tempDir,
            clock,
        });

        // Find a network that should have containers
        let foundPopulated = false;
        for (const net of state.networks.values()) {
            if (net.Containers && Object.keys(net.Containers).length > 0) {
                foundPopulated = true;
                const entry = Object.values(net.Containers)[0];
                expect(entry.Name).toBeTruthy();
                expect(entry.EndpointID).toBeTruthy();
                break;
            }
        }
        expect(foundPopulated).toBe(true);
    });

    it("produces identical state on repeated calls (determinism)", async () => {
        const state1 = await initState({
            stacksSource: fixturesDir,
            stacksDir: tempDir,
            clock,
        });

        // Reset temp dir for second run
        rmSync(tempDir, { recursive: true, force: true });
        const tempDir2 = mkdtempSync(join(tmpdir(), "mock-docker-init-"));

        const state2 = await initState({
            stacksSource: fixturesDir,
            stacksDir: tempDir2,
            clock,
        });

        // Same container IDs
        const ids1 = [...state1.containers.keys()].sort();
        const ids2 = [...state2.containers.keys()].sort();
        expect(ids1).toEqual(ids2);

        // Same network IDs
        const netIds1 = [...state1.networks.keys()].sort();
        const netIds2 = [...state2.networks.keys()].sort();
        expect(netIds1).toEqual(netIds2);

        rmSync(tempDir2, { recursive: true, force: true });
    });

    it("creates global networks from global .mock.yaml", async () => {
        // Create a minimal stacks dir with a global .mock.yaml
        const src = mkdtempSync(join(tmpdir(), "mock-docker-global-"));
        writeFileSync(join(src, ".mock.yaml"), `
networks:
  proxy:
    driver: bridge
    subnet: "172.30.0.0/16"
    gateway: "172.30.0.1"
volumes:
  shared-data:
    driver: local
`);

        const state = await initState({
            stacksSource: src,
            stacksDir: tempDir,
            clock,
        });

        // Should have the global network
        let foundProxy = false;
        for (const net of state.networks.values()) {
            if (net.Name === "proxy") {
                foundProxy = true;
                expect(net.Driver).toBe("bridge");
                expect(net.IPAM.Config![0].Subnet).toBe("172.30.0.0/16");
            }
        }
        expect(foundProxy).toBe(true);

        // Should have the global volume
        expect(state.volumes.has("shared-data")).toBe(true);

        rmSync(src, { recursive: true, force: true });
    });

    it("silently skips directories without compose files", async () => {
        const src = mkdtempSync(join(tmpdir(), "mock-docker-nocompose-"));
        mkdirSync(join(src, "random-dir"));
        writeFileSync(join(src, "random-dir", "README.md"), "not a compose file");

        const state = await initState({
            stacksSource: src,
            stacksDir: tempDir,
            clock,
        });

        expect(state.containers.size).toBe(0);
        rmSync(src, { recursive: true, force: true });
    });

    it("reads env_file and merges into environment", async () => {
        const state = await initState({
            stacksSource: fixturesDir,
            stacksDir: tempDir,
            clock,
        });

        // Find the env-file stack's app container
        let envContainer = null;
        for (const c of state.containers.values()) {
            if (c.Config.Labels!["com.docker.compose.project"] === "env-file" &&
                c.Config.Labels!["com.docker.compose.service"] === "app") {
                envContainer = c;
                break;
            }
        }
        expect(envContainer).not.toBeNull();

        // Should have both env_file vars and compose environment vars
        const env = envContainer!.Config.Env || [];
        const envMap: Record<string, string> = {};
        for (const e of env) {
            const [k, ...v] = e.split("=");
            envMap[k] = v.join("=");
        }
        expect(envMap.NODE_ENV).toBe("production"); // from compose
        expect(envMap.DB_HOST).toBe("localhost");    // from .env file
        expect(envMap.DB_PORT).toBe("5432");         // from .env file
    });

    it("applies mock overrides from .mock.yaml", async () => {
        const state = await initState({
            stacksSource: fixturesDir,
            stacksDir: tempDir,
            clock,
        });

        // Find with-overrides containers
        const overrideContainers = [...state.containers.values()].filter(
            (c) => c.Config.Labels!["com.docker.compose.project"] === "with-overrides",
        );

        const worker = overrideContainers.find(
            (c) => c.Config.Labels!["com.docker.compose.service"] === "worker",
        );
        expect(worker!.State.Status).toBe("exited");
        expect(worker!.State.ExitCode).toBe(137);

        const db = overrideContainers.find(
            (c) => c.Config.Labels!["com.docker.compose.service"] === "db",
        );
        expect(db!.State.Health!.Status).toBe("unhealthy");
    });

    it("uses pre-captured images when images.json is provided", async () => {
        const src = mkdtempSync(join(tmpdir(), "mock-docker-precap-"));
        mkdirSync(join(src, "mystack"));
        writeFileSync(join(src, "mystack", "compose.yaml"), `
services:
  web:
    image: nginx:latest
`);
        // Create images.json with pre-captured data
        const imagesJson = join(src, "images.json");
        writeFileSync(imagesJson, JSON.stringify({
            "nginx:latest": {
                Id: "sha256:precaptured123",
                RepoTags: ["nginx:latest"],
                RepoDigests: ["nginx@sha256:abc"],
                Created: "2025-01-01T00:00:00Z",
                Architecture: "amd64",
                Os: "linux",
                Size: 50000000,
                RootFS: { Type: "layers", Layers: ["sha256:real1", "sha256:real2"] },
                Config: {
                    Cmd: ["nginx", "-g", "daemon off;"],
                    Env: ["PATH=/usr/local/bin:/usr/bin", "NGINX_VERSION=1.25"],
                    ExposedPorts: { "80/tcp": {} },
                },
            },
        }));

        const state = await initState({
            stacksSource: src,
            stacksDir: tempDir,
            clock,
            imagesJsonPath: imagesJson,
        });

        // Image should be precaptured
        const img = [...state.images.values()].find((i) => i.RepoTags.includes("nginx:latest"));
        expect(img).toBeDefined();
        expect(img!.Id).toBe("sha256:precaptured123");
        expect(img!.RootFS.Layers).toHaveLength(2);

        // Container should reference the real image ID
        const container = [...state.containers.values()][0];
        expect(container.Image).toBe("sha256:precaptured123");

        // Container should inherit image env
        const env = container.Config.Env || [];
        const envMap: Record<string, string> = {};
        for (const e of env) {
            const [k, ...v] = e.split("=");
            envMap[k] = v.join("=");
        }
        expect(envMap.NGINX_VERSION).toBe("1.25");

        rmSync(src, { recursive: true, force: true });
    });

    it("falls back to synthetic for images not in images.json", async () => {
        const src = mkdtempSync(join(tmpdir(), "mock-docker-fallback-"));
        mkdirSync(join(src, "mystack"));
        writeFileSync(join(src, "mystack", "compose.yaml"), `
services:
  web:
    image: nginx:latest
  cache:
    image: redis:7
`);
        const imagesJson = join(src, "images.json");
        writeFileSync(imagesJson, JSON.stringify({
            "nginx:latest": {
                Id: "sha256:precaptured123",
                RepoTags: ["nginx:latest"],
                RepoDigests: [],
                Created: "2025-01-01T00:00:00Z",
                Architecture: "amd64",
                Os: "linux",
                Size: 50000000,
                RootFS: { Type: "layers", Layers: ["sha256:real1"] },
                Config: {},
            },
        }));

        const state = await initState({
            stacksSource: src,
            stacksDir: tempDir,
            clock,
            imagesJsonPath: imagesJson,
        });

        const nginx = [...state.images.values()].find((i) => i.RepoTags.includes("nginx:latest"));
        const redis = [...state.images.values()].find((i) => i.RepoTags.includes("redis:7"));
        expect(nginx!.Id).toBe("sha256:precaptured123");
        // redis should be synthetic (deterministic hash)
        expect(redis!.Id).toMatch(/^sha256:/);
        expect(redis!.Id).not.toBe("sha256:precaptured123");

        rmSync(src, { recursive: true, force: true });
    });

    it("compose overrides take precedence over image defaults", async () => {
        const src = mkdtempSync(join(tmpdir(), "mock-docker-override-"));
        mkdirSync(join(src, "mystack"));
        writeFileSync(join(src, "mystack", "compose.yaml"), `
services:
  app:
    image: myapp:latest
    command: /custom
    environment:
      FOO: override
`);
        const imagesJson = join(src, "images.json");
        writeFileSync(imagesJson, JSON.stringify({
            "myapp:latest": {
                Id: "sha256:myappdigest",
                RepoTags: ["myapp:latest"],
                RepoDigests: [],
                Created: "2025-01-01T00:00:00Z",
                Architecture: "amd64",
                Os: "linux",
                Size: 10000000,
                RootFS: { Type: "layers", Layers: ["sha256:l1"] },
                Config: {
                    Cmd: ["/default"],
                    Env: ["PATH=/usr/local/bin", "FOO=bar"],
                },
            },
        }));

        const state = await initState({
            stacksSource: src,
            stacksDir: tempDir,
            clock,
            imagesJsonPath: imagesJson,
        });

        const container = [...state.containers.values()][0];
        // Compose command overrides image Cmd
        expect(container.Config.Cmd).toEqual(["/custom"]);
        // Compose env overrides by key, image env inherited
        const env = container.Config.Env || [];
        const envMap: Record<string, string> = {};
        for (const e of env) {
            const [k, ...v] = e.split("=");
            envMap[k] = v.join("=");
        }
        expect(envMap.FOO).toBe("override");
        expect(envMap.PATH).toBe("/usr/local/bin");

        rmSync(src, { recursive: true, force: true });
    });

    it("handles untracked stacks by deleting their dirs", async () => {
        const src = mkdtempSync(join(tmpdir(), "mock-docker-untracked-"));
        mkdirSync(join(src, "my-stack"));
        writeFileSync(join(src, "my-stack", "compose.yaml"), `
services:
  app:
    image: alpine
`);
        writeFileSync(join(src, "my-stack", ".mock.yaml"), "untracked: true");

        const state = await initState({
            stacksSource: src,
            stacksDir: tempDir,
            clock,
        });

        // Container should exist (it was processed)
        expect(state.containers.size).toBe(1);

        // But the directory should be deleted
        const { existsSync } = await import("node:fs");
        expect(existsSync(join(tempDir, "my-stack"))).toBe(false);

        rmSync(src, { recursive: true, force: true });
    });
});
