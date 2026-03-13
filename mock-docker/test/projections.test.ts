import { describe, it, expect } from "vitest";
import { projectToContainerListEntry, projectToImageListEntry, computeStatusString } from "../src/projections.js";
import { FixedClock } from "../src/clock.js";
import type { ContainerInspect, ContainerState, ImageInspect } from "../src/types.js";

// Helper to build ContainerState with Error defaulted.
function st(s: Omit<ContainerState, "Error"> & { Error?: string }): ContainerState {
    return { Error: "", ...s };
}

// Minimal container builder — only the fields projections actually read.
function makeContainer(overrides: Partial<ContainerInspect> & { State: ContainerState }): ContainerInspect {
    return {
        Id: "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
        Name: "/test-project-web-1",
        Created: "2025-01-01T00:00:00Z",
        Path: "/docker-entrypoint.sh",
        Args: ["nginx", "-g", "daemon off;"],
        Image: "sha256:aabbccdd",
        Config: {
            Image: "nginx:latest",
            Labels: { "com.docker.compose.project": "test-project" },
            Hostname: "web",
            Domainname: "",
            User: "",
            ExposedPorts: {},
            Env: [],
            Cmd: null,
            Entrypoint: null,
            WorkingDir: "",
            Volumes: null,
            StopSignal: "SIGTERM",
            OnBuild: null,
        },
        HostConfig: {
            NetworkMode: "test-project_default",
            RestartPolicy: { Name: "no", MaximumRetryCount: 0 },
            LogConfig: { Type: "json-file", Config: {} },
        },
        NetworkSettings: {
            Bridge: "",
            SandboxID: "",
            SandboxKey: "",
            Ports: {},
            Networks: {
                test_default: {
                    NetworkID: "net123",
                    EndpointID: "ep123",
                    Gateway: "172.18.0.1",
                    IPAddress: "172.18.0.2",
                    IPPrefixLen: 16,
                    MacAddress: "02:42:ac:12:00:02",
                    GlobalIPv6Address: "",
                    GlobalIPv6PrefixLen: 0,
                    IPv6Gateway: "",
                    Links: null,
                    Aliases: null,
                    DriverOpts: null,
                    DNSNames: null,
                },
            },
        },
        Mounts: [],
        Driver: "overlay2",
        Platform: "linux",
        ...overrides,
        State: overrides.State,
    } as ContainerInspect;
}

function makeImage(overrides: Partial<ImageInspect> = {}): ImageInspect {
    return {
        Id: "sha256:aabbccdd",
        Created: "2025-01-01T00:00:00Z",
        Size: 187654321,
        RepoTags: ["nginx:latest"],
        RepoDigests: ["nginx@sha256:abcdef"],
        Architecture: "amd64",
        Os: "linux",
        Config: {
            Image: "",
            Labels: { maintainer: "NGINX Docker Maintainers" },
            Hostname: "",
            Domainname: "",
            User: "",
            ExposedPorts: {},
            Env: [],
            Cmd: null,
            Entrypoint: null,
            WorkingDir: "",
            Volumes: null,
            StopSignal: "",
            OnBuild: null,
        },
        RootFS: { Type: "layers", Layers: [] },
        GraphDriver: { Name: "overlay2", Data: {} },
        ...overrides,
    } as ImageInspect;
}

const BASE_TIME = "2025-01-01T00:00:00Z";
const ZERO_TIME = "0001-01-01T00:00:00Z";

// Shorthand for a running state starting at BASE_TIME.
const RUNNING = st({ Status: "running", Running: true, Paused: false, Restarting: false, OOMKilled: false, Dead: false, Pid: 1, ExitCode: 0, StartedAt: BASE_TIME, FinishedAt: ZERO_TIME });

describe("computeStatusString", () => {
    it("running container shows Up duration", () => {
        const clock = new FixedClock(new Date(BASE_TIME));
        clock.advance(2 * 60 * 60 * 1000);
        expect(computeStatusString(RUNNING, clock)).toBe("Up 2 hours");
    });

    it("running healthy container shows (healthy)", () => {
        const clock = new FixedClock(new Date(BASE_TIME));
        clock.advance(5 * 60 * 1000);
        const s = st({ ...RUNNING, Health: { Status: "healthy", FailingStreak: 0, Log: [] } });
        expect(computeStatusString(s, clock)).toBe("Up 5 minutes (healthy)");
    });

    it("running unhealthy container shows (unhealthy)", () => {
        const clock = new FixedClock(new Date(BASE_TIME));
        clock.advance(30 * 1000);
        const s = st({ ...RUNNING, Health: { Status: "unhealthy", FailingStreak: 3, Log: [] } });
        expect(computeStatusString(s, clock)).toBe("Up 30 seconds (unhealthy)");
    });

    it("exited container shows exit code and duration ago", () => {
        const clock = new FixedClock(new Date(BASE_TIME));
        clock.advance(3 * 24 * 60 * 60 * 1000);
        const s = st({ Status: "exited", Running: false, Paused: false, Restarting: false, OOMKilled: false, Dead: false, Pid: 0, ExitCode: 137, StartedAt: "2024-12-30T00:00:00Z", FinishedAt: BASE_TIME });
        expect(computeStatusString(s, clock)).toBe("Exited (137) 3 days ago");
    });

    it("paused container shows (Paused)", () => {
        const clock = new FixedClock(new Date(BASE_TIME));
        clock.advance(10 * 60 * 1000);
        const s = st({ Status: "paused", Running: true, Paused: true, Restarting: false, OOMKilled: false, Dead: false, Pid: 1234, ExitCode: 0, StartedAt: BASE_TIME, FinishedAt: ZERO_TIME });
        expect(computeStatusString(s, clock)).toBe("Up 10 minutes (Paused)");
    });

    it("created container shows Created", () => {
        const s = st({ Status: "created", Running: false, Paused: false, Restarting: false, OOMKilled: false, Dead: false, Pid: 0, ExitCode: 0, StartedAt: ZERO_TIME, FinishedAt: ZERO_TIME });
        expect(computeStatusString(s)).toBe("Created");
    });

    it("formats 1 second singular", () => {
        const clock = new FixedClock(new Date(BASE_TIME));
        clock.advance(1000);
        expect(computeStatusString(RUNNING, clock)).toBe("Up 1 second");
    });

    it("formats 1 day singular", () => {
        const clock = new FixedClock(new Date(BASE_TIME));
        clock.advance(24 * 60 * 60 * 1000);
        expect(computeStatusString(RUNNING, clock)).toBe("Up 1 day");
    });
});

describe("projectToContainerListEntry", () => {
    it("maps all fields with correct types", () => {
        const clock = new FixedClock(new Date(BASE_TIME));
        clock.advance(60 * 1000);
        const c = makeContainer({ State: RUNNING });
        const entry = projectToContainerListEntry(c, clock);

        expect(entry.Id).toBe(c.Id);
        expect(entry.Names).toEqual(["/test-project-web-1"]);
        expect(entry.Image).toBe("nginx:latest");
        expect(entry.ImageID).toBe("sha256:aabbccdd");
        expect(entry.State).toBe("running");
        expect(entry.Status).toBe("Up 1 minute");
        expect(entry.HostConfig).toEqual({ NetworkMode: "test-project_default" });
        expect(typeof entry.Created).toBe("number");
        expect(entry.Labels).toEqual({ "com.docker.compose.project": "test-project" });
        expect(entry.Mounts).toEqual([]);
        expect(entry.NetworkSettings.Networks).toHaveProperty("test_default");
    });

    it("concatenates Path and Args into Command", () => {
        const c = makeContainer({ State: RUNNING, Path: "/bin/sh", Args: ["-c", "echo hello"] });
        const entry = projectToContainerListEntry(c);
        expect(entry.Command).toBe("/bin/sh -c echo hello");
    });

    it("handles empty Args in Command", () => {
        const c = makeContainer({ State: RUNNING, Path: "/bin/sh", Args: [] });
        const entry = projectToContainerListEntry(c);
        expect(entry.Command).toBe("/bin/sh");
    });

    it("converts Created ISO to unix epoch", () => {
        const c = makeContainer({ State: RUNNING, Created: "2025-06-15T12:30:00Z" });
        const entry = projectToContainerListEntry(c);
        expect(entry.Created).toBe(Math.floor(new Date("2025-06-15T12:30:00Z").getTime() / 1000));
    });

    it("flattens published ports", () => {
        const c = makeContainer({
            State: RUNNING,
            NetworkSettings: {
                Bridge: "", SandboxID: "", SandboxKey: "",
                Ports: {
                    "80/tcp": [{ HostIp: "0.0.0.0", HostPort: "8080" }],
                    "443/tcp": [{ HostIp: "0.0.0.0", HostPort: "8443" }, { HostIp: "::", HostPort: "8443" }],
                },
                Networks: {},
            },
        });
        const entry = projectToContainerListEntry(c);
        expect(entry.Ports).toHaveLength(3);
        expect(entry.Ports[0]).toEqual({ PrivatePort: 80, PublicPort: 8080, Type: "tcp", IP: "0.0.0.0" });
        expect(entry.Ports[1]).toEqual({ PrivatePort: 443, PublicPort: 8443, Type: "tcp", IP: "0.0.0.0" });
        expect(entry.Ports[2]).toEqual({ PrivatePort: 443, PublicPort: 8443, Type: "tcp", IP: "::" });
    });

    it("flattens exposed-only ports (null bindings)", () => {
        const c = makeContainer({
            State: RUNNING,
            NetworkSettings: {
                Bridge: "", SandboxID: "", SandboxKey: "",
                Ports: { "3306/tcp": null },
                Networks: {},
            },
        });
        const entry = projectToContainerListEntry(c);
        expect(entry.Ports).toEqual([{ PrivatePort: 3306, Type: "tcp" }]);
    });

    it("includes Health when healthcheck exists", () => {
        const s = st({ ...RUNNING, Health: { Status: "healthy", FailingStreak: 0, Log: [] } });
        const c = makeContainer({ State: s });
        const entry = projectToContainerListEntry(c);
        expect(entry.Health).toEqual({ Status: "healthy", FailingStreak: 0 });
    });

    it("omits Health when no healthcheck", () => {
        const c = makeContainer({ State: RUNNING });
        const entry = projectToContainerListEntry(c);
        expect(entry.Health).toBeUndefined();
    });
});

describe("projectToImageListEntry", () => {
    it("maps all fields correctly", () => {
        const img = makeImage();
        const containers = new Map<string, ContainerInspect>();
        const entry = projectToImageListEntry(img, containers);

        expect(entry.Id).toBe("sha256:aabbccdd");
        expect(entry.ParentId).toBe("");
        expect(entry.RepoTags).toEqual(["nginx:latest"]);
        expect(entry.RepoDigests).toEqual(["nginx@sha256:abcdef"]);
        expect(entry.Created).toBe(Math.floor(new Date("2025-01-01T00:00:00Z").getTime() / 1000));
        expect(entry.Size).toBe(187654321);
        expect(entry.SharedSize).toBe(-1);
        expect(entry.Labels).toEqual({ maintainer: "NGINX Docker Maintainers" });
        expect(entry.Containers).toBe(0);
    });

    it("counts containers using the image", () => {
        const img = makeImage();
        const c1 = makeContainer({ State: RUNNING, Id: "c1".padEnd(64, "0"), Image: "sha256:aabbccdd" });
        const c2 = makeContainer({ State: RUNNING, Id: "c2".padEnd(64, "0"), Image: "sha256:aabbccdd" });
        const c3 = makeContainer({ State: RUNNING, Id: "c3".padEnd(64, "0"), Image: "sha256:different" });

        const containers = new Map([
            [c1.Id, c1],
            [c2.Id, c2],
            [c3.Id, c3],
        ]);
        const entry = projectToImageListEntry(img, containers);
        expect(entry.Containers).toBe(2);
    });

    it("uses empty object when Config.Labels is undefined", () => {
        const img = makeImage({ Config: undefined as any });
        const entry = projectToImageListEntry(img, new Map());
        expect(entry.Labels).toEqual({});
    });
});
