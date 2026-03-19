import { describe, it, expect } from "vitest";
import { generateStack, resolveNetworkName, resolveNetworkNameFromConfig } from "../src/generator.js";
import { parseCompose } from "../src/compose-parser.js";
import { parseStackMockConfig } from "../src/mock-config.js";
import { FixedClock } from "../src/clock.js";
import type { GeneratorInput } from "../src/generator.js";
import type { ImageInspect } from "../src/types.js";

function makeInput(yaml: string, mockYaml: string | null = null): GeneratorInput {
    return {
        project: "test-project",
        stackDir: "/opt/stacks/test-project",
        composeFilePath: "/opt/stacks/test-project/compose.yaml",
        parsed: parseCompose(yaml),
        mockConfig: parseStackMockConfig(mockYaml),
        clock: new FixedClock(new Date("2025-01-01T00:00:00Z")),
    };
}

describe("generateStack", () => {
    it("generates a single service with correct fields", () => {
        const input = makeInput(`
services:
  web:
    image: nginx:latest
    ports:
      - "8080:80"
`);
        const result = generateStack(input);

        expect(result.containers).toHaveLength(1);
        const c = result.containers[0];
        expect(c.Id).toHaveLength(64);
        expect(c.Name).toBe("/test-project-web-1");
        expect(c.State.Status).toBe("running");
        expect(c.State.Running).toBe(true);
        expect(c.Config.Image).toBe("nginx:latest");
        expect(c.Driver).toBe("overlay2");
        expect(c.Platform).toBe("linux");
        expect(c.Image).toMatch(/^sha256:/);
    });

    it("applies compose labels to all containers", () => {
        const input = makeInput(`
services:
  web:
    image: nginx:latest
  api:
    image: node:20
`);
        const result = generateStack(input);

        for (const c of result.containers) {
            expect(c.Config.Labels!["com.docker.compose.project"]).toBe("test-project");
            expect(c.Config.Labels!["com.docker.compose.version"]).toBe("2.30.0");
            expect(c.Config.Labels!["com.docker.compose.oneoff"]).toBe("False");
            expect(c.Config.Labels!["com.docker.compose.container-number"]).toBe("1");
        }

        const web = result.containers.find((c) => c.Name === "/test-project-web-1");
        expect(web!.Config.Labels!["com.docker.compose.service"]).toBe("web");
        const api = result.containers.find((c) => c.Name === "/test-project-api-1");
        expect(api!.Config.Labels!["com.docker.compose.service"]).toBe("api");
    });

    it("handles host network mode", () => {
        const input = makeInput(`
services:
  app:
    image: alpine
    network_mode: host
`);
        const result = generateStack(input);
        const c = result.containers[0];
        expect(c.HostConfig.NetworkMode).toBe("host");
        expect(c.NetworkSettings.Networks!["host"]).toBeDefined();
    });

    it("handles none network mode", () => {
        const input = makeInput(`
services:
  app:
    image: alpine
    network_mode: none
`);
        const result = generateStack(input);
        const c = result.containers[0];
        expect(c.HostConfig.NetworkMode).toBe("none");
        expect(Object.keys(c.NetworkSettings.Networks!)).toHaveLength(0);
    });

    it("creates implicit default network", () => {
        const input = makeInput(`
services:
  app:
    image: alpine
`);
        const result = generateStack(input);
        expect(result.networks.some((n) => n.Name === "test-project_default")).toBe(true);
        const c = result.containers[0];
        expect(c.HostConfig.NetworkMode).toBe("test-project_default");
        expect(c.NetworkSettings.Networks!["test-project_default"]).toBeDefined();
    });

    it("creates explicit networks", () => {
        const input = makeInput(`
services:
  web:
    image: nginx
    networks:
      - frontend
      - backend
networks:
  frontend:
  backend:
`);
        const result = generateStack(input);
        expect(result.networks).toHaveLength(2);
        expect(result.networks.map((n) => n.Name).sort()).toEqual([
            "test-project_backend",
            "test-project_frontend",
        ]);

        const c = result.containers[0];
        expect(c.NetworkSettings.Networks!["test-project_frontend"]).toBeDefined();
        expect(c.NetworkSettings.Networks!["test-project_backend"]).toBeDefined();
    });

    it("applies mock override: state", () => {
        const input = makeInput(
            `
services:
  web:
    image: nginx
`,
            `
services:
  web:
    state: exited
    exit_code: 137
`,
        );
        const result = generateStack(input);
        const c = result.containers[0];
        expect(c.State.Status).toBe("exited");
        expect(c.State.Running).toBe(false);
        expect(c.State.ExitCode).toBe(137);
        expect(c.State.Pid).toBe(0);
    });

    it("applies mock override: health", () => {
        const input = makeInput(
            `
services:
  web:
    image: nginx
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost"]
      interval: 30s
`,
            `
services:
  web:
    health: unhealthy
`,
        );
        const result = generateStack(input);
        const c = result.containers[0];
        expect(c.State.Health).toBeDefined();
        expect(c.State.Health!.Status).toBe("unhealthy");
        expect(c.State.Health!.FailingStreak).toBe(3);
    });

    it("defaults health to healthy when healthcheck defined", () => {
        const input = makeInput(`
services:
  web:
    image: nginx
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost"]
`);
        const result = generateStack(input);
        expect(result.containers[0].State.Health!.Status).toBe("healthy");
    });

    it("no health when no healthcheck and no override", () => {
        const input = makeInput(`
services:
  web:
    image: nginx
`);
        const result = generateStack(input);
        expect(result.containers[0].State.Health).toBeUndefined();
    });

    it("generates volumes from compose", () => {
        const input = makeInput(`
services:
  db:
    image: postgres
    volumes:
      - pgdata:/var/lib/postgresql/data
volumes:
  pgdata:
`);
        const result = generateStack(input);
        expect(result.volumes).toHaveLength(1);
        expect(result.volumes[0].Name).toBe("test-project_pgdata");
        expect(result.volumes[0].Mountpoint).toBe("/var/lib/docker/volumes/test-project_pgdata/_data");
        expect(result.volumes[0].Labels!["com.docker.compose.volume"]).toBe("pgdata");

        const c = result.containers[0];
        const mount = c.Mounts.find((m) => m.Destination === "/var/lib/postgresql/data");
        expect(mount).toBeDefined();
        expect(mount!.Type).toBe("volume");
        expect(mount!.Name).toBe("test-project_pgdata");

        // Named volumes also appear in HostConfig.Binds (matching real Docker)
        expect(c.HostConfig.Binds).toContain("pgdata:/var/lib/postgresql/data:rw");
    });

    it("puts both bind mounts and named volumes in HostConfig.Binds", () => {
        const input = makeInput(`
services:
  app:
    image: nginx
    volumes:
      - appdata:/data
      - ./config:/etc/config:ro
volumes:
  appdata:
`);
        const result = generateStack(input);
        const c = result.containers[0];

        // Both in Binds
        expect(c.HostConfig.Binds).toContain("appdata:/data:rw");
        expect(c.HostConfig.Binds).toContain("./config:/etc/config:ro");

        // Both in Mounts array
        expect(c.Mounts).toHaveLength(2);
        expect(c.Mounts.find((m) => m.Type === "volume" && m.Destination === "/data")).toBeDefined();
        expect(c.Mounts.find((m) => m.Type === "bind" && m.Destination === "/etc/config")).toBeDefined();
    });

    it("generates synthetic images", () => {
        const input = makeInput(`
services:
  web:
    image: nginx:latest
  api:
    image: node:20-alpine
`);
        const result = generateStack(input);
        expect(result.images).toHaveLength(2);
        for (const img of result.images) {
            expect(img.Id).toMatch(/^sha256:/);
            expect(img.RepoTags).toHaveLength(1);
            expect(img.Architecture).toBe("amd64");
            expect(img.Os).toBe("linux");
            expect(img.RootFS.Layers).toHaveLength(1);
        }
    });

    it("deduplicates images when services share the same ref", () => {
        const input = makeInput(`
services:
  web1:
    image: nginx:latest
  web2:
    image: nginx:latest
`);
        const result = generateStack(input);
        expect(result.images).toHaveLength(1);
    });

    it("generates image for build-only service", () => {
        const input = makeInput(`
services:
  app:
    build: .
`);
        const result = generateStack(input);
        expect(result.images).toHaveLength(1);
        expect(result.images[0].RepoTags[0]).toBe("test-project-app:latest");
    });

    it("resolves service: network mode to container ID", () => {
        const input = makeInput(`
services:
  app:
    image: alpine
  sidecar:
    image: alpine
    network_mode: "service:app"
`);
        const result = generateStack(input);
        const sidecar = result.containers.find((c) => c.Name === "/test-project-sidecar-1");
        const app = result.containers.find((c) => c.Name === "/test-project-app-1");
        expect(sidecar!.HostConfig.NetworkMode).toBe(`container:${app!.Id}`);
    });

    it("produces deterministic output (same input → same IDs)", () => {
        const input1 = makeInput(`
services:
  web:
    image: nginx:latest
`);
        const input2 = makeInput(`
services:
  web:
    image: nginx:latest
`);
        const r1 = generateStack(input1);
        const r2 = generateStack(input2);
        expect(r1.containers[0].Id).toBe(r2.containers[0].Id);
        expect(r1.networks[0].Id).toBe(r2.networks[0].Id);
        expect(r1.images[0].Id).toBe(r2.images[0].Id);
    });

    it("generates port bindings in HostConfig and NetworkSettings", () => {
        const input = makeInput(`
services:
  web:
    image: nginx
    ports:
      - "8080:80"
      - "443:443/tcp"
`);
        const result = generateStack(input);
        const c = result.containers[0];
        expect(c.HostConfig.PortBindings!["80/tcp"]).toEqual([
            { HostIp: "", HostPort: "8080" },
        ]);
        expect(c.NetworkSettings.Ports!["80/tcp"]).toEqual([
            { HostIp: "0.0.0.0", HostPort: "8080" },
        ]);
    });

    it("uses container_name when specified", () => {
        const input = makeInput(`
services:
  web:
    image: nginx
    container_name: my-custom-name
`);
        const result = generateStack(input);
        expect(result.containers[0].Name).toBe("/my-custom-name");
    });

    it("includes restart policy", () => {
        const input = makeInput(`
services:
  web:
    image: nginx
    restart: unless-stopped
`);
        const result = generateStack(input);
        expect(result.containers[0].HostConfig.RestartPolicy).toEqual({
            Name: "unless-stopped",
            MaximumRetryCount: 0,
        });
    });

    it("uses external network name (not project-prefixed) for service references", () => {
        const input = makeInput(`
services:
  web:
    image: nginx
    networks:
      - proxy
networks:
  proxy:
    external: true
`);
        const result = generateStack(input);
        const c = result.containers[0];

        // NetworkMode should be the bare external name, not test-project_proxy
        expect(c.HostConfig.NetworkMode).toBe("proxy");

        // NetworkSettings.Networks key should match
        expect(c.NetworkSettings.Networks!["proxy"]).toBeDefined();
        expect(c.NetworkSettings.Networks!["test-project_proxy"]).toBeUndefined();
    });

    it("uses external network with custom name", () => {
        const input = makeInput(`
services:
  web:
    image: nginx
    networks:
      - shared
networks:
  shared:
    name: my-shared-net
    external: true
`);
        const result = generateStack(input);
        const c = result.containers[0];

        expect(c.HostConfig.NetworkMode).toBe("my-shared-net");
        expect(c.NetworkSettings.Networks!["my-shared-net"]).toBeDefined();
    });

    it("uses precaptured image when available", () => {
        const precaptured = new Map<string, ImageInspect>();
        precaptured.set("nginx:latest", {
            Id: "sha256:realdigest123",
            RepoTags: ["nginx:latest"],
            RepoDigests: ["nginx@sha256:abc"],
            Created: "2025-01-01T00:00:00Z",
            Architecture: "amd64",
            Os: "linux",
            Size: 50_000_000,
            RootFS: { Type: "layers", Layers: ["sha256:layer1", "sha256:layer2"] },
            Config: {
                Cmd: ["/docker-entrypoint.sh", "nginx", "-g", "daemon off;"],
                Env: ["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin", "NGINX_VERSION=1.25.0"],
                ExposedPorts: { "80/tcp": {} },
                StopSignal: "SIGQUIT",
            },
        });

        const input: GeneratorInput = {
            ...makeInput(`
services:
  web:
    image: nginx:latest
    ports:
      - "8080:80"
`),
            precapturedImages: precaptured,
        };
        const result = generateStack(input);

        // Image should be the precaptured one
        expect(result.images).toHaveLength(1);
        expect(result.images[0].Id).toBe("sha256:realdigest123");
        expect(result.images[0].RootFS.Layers).toHaveLength(2);

        // Container should reference the real image ID
        const c = result.containers[0];
        expect(c.Image).toBe("sha256:realdigest123");
    });

    it("container inherits image entrypoint when compose has none", () => {
        const precaptured = new Map<string, ImageInspect>();
        precaptured.set("nginx:latest", {
            Id: "sha256:realdigest",
            RepoTags: ["nginx:latest"],
            RepoDigests: [],
            Created: "2025-01-01T00:00:00Z",
            Architecture: "amd64",
            Os: "linux",
            Size: 50_000_000,
            RootFS: { Type: "layers", Layers: ["sha256:layer1"] },
            Config: {
                Entrypoint: ["/docker-entrypoint.sh"],
                Cmd: ["nginx", "-g", "daemon off;"],
            },
        });

        const input: GeneratorInput = {
            ...makeInput(`
services:
  web:
    image: nginx:latest
`),
            precapturedImages: precaptured,
        };
        const result = generateStack(input);
        const c = result.containers[0];

        expect(c.Path).toBe("/docker-entrypoint.sh");
        expect(c.Config.Entrypoint).toEqual(["/docker-entrypoint.sh"]);
        expect(c.Config.Cmd).toEqual(["nginx", "-g", "daemon off;"]);
    });

    it("compose command overrides image cmd", () => {
        const precaptured = new Map<string, ImageInspect>();
        precaptured.set("nginx:latest", {
            Id: "sha256:realdigest",
            RepoTags: ["nginx:latest"],
            RepoDigests: [],
            Created: "2025-01-01T00:00:00Z",
            Architecture: "amd64",
            Os: "linux",
            Size: 50_000_000,
            RootFS: { Type: "layers", Layers: ["sha256:layer1"] },
            Config: {
                Entrypoint: ["/docker-entrypoint.sh"],
                Cmd: ["nginx", "-g", "daemon off;"],
            },
        });

        const input: GeneratorInput = {
            ...makeInput(`
services:
  web:
    image: nginx:latest
    command: nginx-debug -g "daemon off;"
`),
            precapturedImages: precaptured,
        };
        const result = generateStack(input);
        const c = result.containers[0];

        // Compose command overrides image Cmd
        expect(c.Config.Cmd).toEqual(["nginx-debug", "-g", '"daemon', 'off;"']);
        // Image entrypoint is preserved
        expect(c.Config.Entrypoint).toEqual(["/docker-entrypoint.sh"]);
    });

    it("container inherits image env as base, compose overrides by key", () => {
        const precaptured = new Map<string, ImageInspect>();
        precaptured.set("postgres:16", {
            Id: "sha256:pgdigest",
            RepoTags: ["postgres:16"],
            RepoDigests: [],
            Created: "2025-01-01T00:00:00Z",
            Architecture: "amd64",
            Os: "linux",
            Size: 100_000_000,
            RootFS: { Type: "layers", Layers: ["sha256:layer1"] },
            Config: {
                Env: ["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin", "PGDATA=/var/lib/postgresql/data", "LANG=en_US.utf8"],
            },
        });

        const input: GeneratorInput = {
            ...makeInput(`
services:
  db:
    image: postgres:16
    environment:
      POSTGRES_PASSWORD: secret
      LANG: C.UTF-8
`),
            precapturedImages: precaptured,
        };
        const result = generateStack(input);
        const c = result.containers[0];
        const env = c.Config.Env || [];
        const envMap: Record<string, string> = {};
        for (const e of env) {
            const [k, ...v] = e.split("=");
            envMap[k] = v.join("=");
        }

        // Image env inherited
        expect(envMap.PATH).toBe("/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin");
        expect(envMap.PGDATA).toBe("/var/lib/postgresql/data");
        // Compose overrides by key
        expect(envMap.LANG).toBe("C.UTF-8");
        // Compose additions
        expect(envMap.POSTGRES_PASSWORD).toBe("secret");
    });

    it("merges image ExposedPorts with compose ports", () => {
        const precaptured = new Map<string, ImageInspect>();
        precaptured.set("nginx:latest", {
            Id: "sha256:realdigest",
            RepoTags: ["nginx:latest"],
            RepoDigests: [],
            Created: "2025-01-01T00:00:00Z",
            Architecture: "amd64",
            Os: "linux",
            Size: 50_000_000,
            RootFS: { Type: "layers", Layers: ["sha256:layer1"] },
            Config: {
                ExposedPorts: { "80/tcp": {} },
            },
        });

        const input: GeneratorInput = {
            ...makeInput(`
services:
  web:
    image: nginx:latest
    ports:
      - "8443:443"
`),
            precapturedImages: precaptured,
        };
        const result = generateStack(input);
        const c = result.containers[0];

        // Both image's 80/tcp and compose's 443/tcp should be exposed
        expect(c.Config.ExposedPorts!["80/tcp"]).toBeDefined();
        expect(c.Config.ExposedPorts!["443/tcp"]).toBeDefined();
    });

    it("falls back to synthetic for unknown images", () => {
        const precaptured = new Map<string, ImageInspect>();
        precaptured.set("nginx:latest", {
            Id: "sha256:realdigest",
            RepoTags: ["nginx:latest"],
            RepoDigests: [],
            Created: "2025-01-01T00:00:00Z",
            Architecture: "amd64",
            Os: "linux",
            Size: 50_000_000,
            RootFS: { Type: "layers", Layers: ["sha256:layer1"] },
            Config: {},
        });

        const input: GeneratorInput = {
            ...makeInput(`
services:
  web:
    image: nginx:latest
  cache:
    image: redis:7
`),
            precapturedImages: precaptured,
        };
        const result = generateStack(input);

        const nginxImg = result.images.find((i) => i.RepoTags.includes("nginx:latest"));
        const redisImg = result.images.find((i) => i.RepoTags.includes("redis:7"));

        // nginx uses precaptured
        expect(nginxImg!.Id).toBe("sha256:realdigest");
        // redis falls back to synthetic
        expect(redisImg!.Id).toMatch(/^sha256:/);
        expect(redisImg!.RootFS.Layers).toHaveLength(1); // synthetic has 1 layer
    });
});

describe("resolveNetworkNameFromConfig", () => {
    it("external network uses name or key", () => {
        expect(resolveNetworkNameFromConfig("proj", "proxy", { name: "shared-proxy", external: true })).toBe("shared-proxy");
        expect(resolveNetworkNameFromConfig("proj", "proxy", { external: true })).toBe("proxy");
    });

    it("non-external uses project_key or name", () => {
        expect(resolveNetworkNameFromConfig("proj", "frontend", { external: false })).toBe("proj_frontend");
        expect(resolveNetworkNameFromConfig("proj", "frontend", { name: "custom", external: false })).toBe("custom");
    });
});
