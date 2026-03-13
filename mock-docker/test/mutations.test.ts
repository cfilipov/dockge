import { describe, it, expect, beforeEach } from "vitest";
import { MockState } from "../src/state.js";
import { FixedClock } from "../src/clock.js";
import { EventEmitter } from "../src/events.js";
import type { DockerEvent } from "../src/list-types.js";
import type { ContainerInspect, NetworkInspect, ImageInspect } from "../src/types.js";
import {
    containerStart,
    containerStop,
    containerRestart,
    containerPause,
    containerUnpause,
    containerRemove,
    containerCreate,
    containerRename,
    containerKill,
    networkCreate,
    networkRemove,
    networkConnect,
    networkDisconnect,
    volumeCreate,
    volumeRemove,
    imageRemove,
    imagePrune,
} from "../src/mutations.js";

// ---------------------------------------------------------------------------
// Test fixture
// ---------------------------------------------------------------------------

interface TestEnv {
    state: MockState;
    clock: FixedClock;
    emitter: EventEmitter;
    events: DockerEvent[];
    stoppedContainer: ContainerInspect;
    runningContainer: ContainerInspect;
    network: NetworkInspect;
    image: ImageInspect;
}

function makeTestState(): TestEnv {
    const state = new MockState();
    const clock = new FixedClock(new Date("2025-01-01T00:00:00Z"));
    const emitter = new EventEmitter();
    const events: DockerEvent[] = [];
    emitter.subscribe((e) => events.push(e));

    // Image
    const image: ImageInspect = {
        Id: "sha256:aabbccddee00112233445566778899aabbccddee00112233445566778899aabb",
        RepoTags: ["myapp:latest"],
        RepoDigests: ["myapp@sha256:digest000"],
        Created: "2025-01-01T00:00:00Z",
        Architecture: "amd64",
        Os: "linux",
        Size: 50_000_000,
        RootFS: { Type: "layers", Layers: ["sha256:layer1"] },
        Config: { Labels: {} },
    };
    state.images.set(image.Id, image);

    // Network
    const network: NetworkInspect = {
        Name: "myproject_default",
        Id: "net0001aabbccddee00112233445566778899aabbccddee00112233445566",
        Created: "2025-01-01T00:00:00Z",
        Scope: "local",
        Driver: "bridge",
        EnableIPv4: true,
        EnableIPv6: false,
        IPAM: {
            Driver: "default",
            Config: [{ Subnet: "172.20.0.0/16", Gateway: "172.20.0.1" }],
            Options: {},
        },
        Internal: false,
        Attachable: false,
        Ingress: false,
        Containers: {},
        Options: {},
        Labels: {
            "com.docker.compose.network": "default",
            "com.docker.compose.project": "myproject",
        },
    };
    state.networks.set(network.Id, network);

    // Stopped container
    const stoppedContainer: ContainerInspect = {
        Id: "ctr_stopped_001aabbccddee00112233445566778899aabbccddee001122334455",
        Created: "2025-01-01T00:00:00Z",
        Path: "/bin/sh",
        Args: [],
        State: {
            Status: "exited",
            Running: false,
            Paused: false,
            Restarting: false,
            OOMKilled: false,
            Dead: false,
            Pid: 0,
            ExitCode: 0,
            Error: "",
            StartedAt: "2024-12-31T23:00:00Z",
            FinishedAt: "2024-12-31T23:30:00Z",
        },
        Image: image.Id,
        Name: "/myproject-web-1",
        HostConfig: {
            NetworkMode: "myproject_default",
            RestartPolicy: { Name: "always", MaximumRetryCount: 0 },
        },
        Mounts: [],
        Config: {
            Image: "myapp:latest",
            Labels: {
                "com.docker.compose.project": "myproject",
                "com.docker.compose.service": "web",
            },
        },
        NetworkSettings: {
            Networks: {
                myproject_default: {
                    NetworkID: network.Id,
                    EndpointID: "ep_stopped_001",
                    Gateway: "172.20.0.1",
                    IPAddress: "172.20.0.2",
                    IPPrefixLen: 16,
                    MacAddress: "02:42:ac:14:00:02",
                },
            },
        },
    };
    state.containers.set(stoppedContainer.Id, stoppedContainer);

    // Running container
    const runningContainer: ContainerInspect = {
        Id: "ctr_running_001aabbccddee00112233445566778899aabbccddee0011223344",
        Created: "2025-01-01T00:00:00Z",
        Path: "/bin/sh",
        Args: [],
        State: {
            Status: "running",
            Running: true,
            Paused: false,
            Restarting: false,
            OOMKilled: false,
            Dead: false,
            Pid: 12345,
            ExitCode: 0,
            Error: "",
            StartedAt: "2025-01-01T00:00:00Z",
            FinishedAt: "0001-01-01T00:00:00Z",
        },
        Image: image.Id,
        Name: "/myproject-api-1",
        HostConfig: {
            NetworkMode: "myproject_default",
            RestartPolicy: { Name: "always", MaximumRetryCount: 0 },
        },
        Mounts: [],
        Config: {
            Image: "myapp:latest",
            Labels: {
                "com.docker.compose.project": "myproject",
                "com.docker.compose.service": "api",
            },
        },
        NetworkSettings: {
            Networks: {
                myproject_default: {
                    NetworkID: network.Id,
                    EndpointID: "ep_running_001",
                    Gateway: "172.20.0.1",
                    IPAddress: "172.20.0.3",
                    IPPrefixLen: 16,
                    MacAddress: "02:42:ac:14:00:03",
                },
            },
        },
    };
    state.containers.set(runningContainer.Id, runningContainer);

    // Populate network.Containers for the running container
    network.Containers![runningContainer.Id] = {
        Name: "myproject-api-1",
        EndpointID: "ep_running_001",
        MacAddress: "02:42:ac:14:00:03",
        IPv4Address: "172.20.0.3/16",
    };

    return { state, clock, emitter, events, stoppedContainer, runningContainer, network, image };
}

// ---------------------------------------------------------------------------
// Container: start
// ---------------------------------------------------------------------------

describe("containerStart", () => {
    let env: TestEnv;
    beforeEach(() => { env = makeTestState(); });

    it("starts a stopped container and updates state", () => {
        const { state, clock, emitter, events, stoppedContainer, network } = env;
        const result = containerStart(state, stoppedContainer.Id, emitter, clock);
        expect("ok" in result).toBe(true);

        const c = state.containers.get(stoppedContainer.Id)!;
        expect(c.State.Status).toBe("running");
        expect(c.State.Running).toBe(true);
        expect(c.State.Paused).toBe(false);
        expect(c.State.Pid).toBeGreaterThan(0);
        expect(c.State.StartedAt).toBe("2025-01-01T00:00:00.000Z");
        expect(c.State.FinishedAt).toBe("0001-01-01T00:00:00Z");
        expect(c.State.ExitCode).toBe(0);
    });

    it("adds container to network Containers map", () => {
        const { state, clock, emitter, stoppedContainer, network } = env;
        containerStart(state, stoppedContainer.Id, emitter, clock);

        expect(network.Containers![stoppedContainer.Id]).toBeDefined();
        expect(network.Containers![stoppedContainer.Id].Name).toBe("myproject-web-1");
    });

    it("emits start event with compose labels", () => {
        const { state, clock, emitter, events, stoppedContainer } = env;
        containerStart(state, stoppedContainer.Id, emitter, clock);

        expect(events).toHaveLength(1);
        expect(events[0].Action).toBe("start");
        expect(events[0].Type).toBe("container");
        expect(events[0].Actor.ID).toBe(stoppedContainer.Id);
        expect(events[0].Actor.Attributes["com.docker.compose.project"]).toBe("myproject");
        expect(events[0].Actor.Attributes["com.docker.compose.service"]).toBe("web");
    });

    it("returns 304 if already running", () => {
        const { state, clock, emitter, runningContainer } = env;
        const result = containerStart(state, runningContainer.Id, emitter, clock);
        expect("error" in result).toBe(true);
        if ("error" in result) expect(result.statusCode).toBe(304);
    });

    it("sets Health to healthy when container has a healthcheck", () => {
        const { state, clock, emitter, stoppedContainer } = env;
        stoppedContainer.Config.Healthcheck = {
            Test: ["CMD", "curl", "-f", "http://localhost/"],
            Interval: 30_000_000_000,
            Timeout: 10_000_000_000,
            Retries: 3,
        };
        containerStart(state, stoppedContainer.Id, emitter, clock);

        const c = state.containers.get(stoppedContainer.Id)!;
        expect(c.State.Health).toEqual({ Status: "healthy", FailingStreak: 0, Log: [] });
    });

    it("does not set Health when container has no healthcheck", () => {
        const { state, clock, emitter, stoppedContainer } = env;
        delete stoppedContainer.Config.Healthcheck;
        containerStart(state, stoppedContainer.Id, emitter, clock);

        const c = state.containers.get(stoppedContainer.Id)!;
        expect(c.State.Health).toBeUndefined();
    });

    it("defaults to e2eMode=true: health is 'healthy' immediately", () => {
        const { state, clock, emitter, stoppedContainer } = env;
        stoppedContainer.Config.Healthcheck = {
            Test: ["CMD", "curl", "-f", "http://localhost/"],
            Interval: 30_000_000_000,
            Timeout: 10_000_000_000,
            Retries: 3,
        };
        // No opts passed — default e2eMode is true
        containerStart(state, stoppedContainer.Id, emitter, clock);

        const c = state.containers.get(stoppedContainer.Id)!;
        expect(c.State.Health!.Status).toBe("healthy");
    });

    it("e2eMode=false sets health to 'starting'", () => {
        const { state, clock, emitter, stoppedContainer } = env;
        stoppedContainer.Config.Healthcheck = {
            Test: ["CMD", "curl", "-f", "http://localhost/"],
            Interval: 30_000_000_000,
            Timeout: 10_000_000_000,
            Retries: 3,
        };
        containerStart(state, stoppedContainer.Id, emitter, clock, { e2eMode: false });

        const c = state.containers.get(stoppedContainer.Id)!;
        expect(c.State.Health!.Status).toBe("starting");
    });
});

// ---------------------------------------------------------------------------
// Container: stop
// ---------------------------------------------------------------------------

describe("containerStop", () => {
    let env: TestEnv;
    beforeEach(() => { env = makeTestState(); });

    it("stops a running container and updates state", () => {
        const { state, clock, emitter, runningContainer } = env;
        const result = containerStop(state, runningContainer.Id, emitter, clock);
        expect("ok" in result).toBe(true);

        const c = state.containers.get(runningContainer.Id)!;
        expect(c.State.Status).toBe("exited");
        expect(c.State.Running).toBe(false);
        expect(c.State.Pid).toBe(0);
        expect(c.State.ExitCode).toBe(0);
        expect(c.State.FinishedAt).toBe("2025-01-01T00:00:00.000Z");
    });

    it("removes container from network Containers map", () => {
        const { state, clock, emitter, runningContainer, network } = env;
        expect(network.Containers![runningContainer.Id]).toBeDefined();

        containerStop(state, runningContainer.Id, emitter, clock);
        expect(network.Containers![runningContainer.Id]).toBeUndefined();
    });

    it("emits kill, die, stop events in order", () => {
        const { state, clock, emitter, events, runningContainer } = env;
        containerStop(state, runningContainer.Id, emitter, clock);

        expect(events.map((e) => e.Action)).toEqual(["kill", "die", "stop"]);
        expect(events[0].Actor.Attributes.signal).toBe("SIGTERM");
        expect(events[1].Actor.Attributes.exitCode).toBe("0");
    });

    it("returns 304 if already stopped", () => {
        const { state, clock, emitter, stoppedContainer } = env;
        const result = containerStop(state, stoppedContainer.Id, emitter, clock);
        expect("error" in result).toBe(true);
        if ("error" in result) expect(result.statusCode).toBe(304);
    });
});

// ---------------------------------------------------------------------------
// Container: restart
// ---------------------------------------------------------------------------

describe("containerRestart", () => {
    let env: TestEnv;
    beforeEach(() => { env = makeTestState(); });

    it("restarts a running container with combined event sequence ending in restart", () => {
        const { state, clock, emitter, events, runningContainer } = env;
        const result = containerRestart(state, runningContainer.Id, emitter, clock);
        expect("ok" in result).toBe(true);

        const actions = events.map((e) => e.Action);
        // stop events: kill, die, stop; then start, then restart
        expect(actions).toEqual(["kill", "die", "stop", "start", "restart"]);

        const c = state.containers.get(runningContainer.Id)!;
        expect(c.State.Running).toBe(true);
    });

    it("restarts a stopped container (just start + restart)", () => {
        const { state, clock, emitter, events, stoppedContainer } = env;
        const result = containerRestart(state, stoppedContainer.Id, emitter, clock);
        expect("ok" in result).toBe(true);

        const actions = events.map((e) => e.Action);
        expect(actions).toEqual(["start", "restart"]);
    });
});

// ---------------------------------------------------------------------------
// Container: pause / unpause
// ---------------------------------------------------------------------------

describe("containerPause", () => {
    let env: TestEnv;
    beforeEach(() => { env = makeTestState(); });

    it("pauses a running container", () => {
        const { state, clock, emitter, events, runningContainer } = env;
        const result = containerPause(state, runningContainer.Id, emitter, clock);
        expect("ok" in result).toBe(true);

        const c = state.containers.get(runningContainer.Id)!;
        expect(c.State.Status).toBe("paused");
        expect(c.State.Paused).toBe(true);

        expect(events).toHaveLength(1);
        expect(events[0].Action).toBe("pause");
    });

    it("returns 409 if container not running", () => {
        const { state, clock, emitter, stoppedContainer } = env;
        const result = containerPause(state, stoppedContainer.Id, emitter, clock);
        expect("error" in result).toBe(true);
        if ("error" in result) expect(result.statusCode).toBe(409);
    });

    it("returns 409 if already paused", () => {
        const { state, clock, emitter, runningContainer } = env;
        containerPause(state, runningContainer.Id, emitter, clock);
        const result = containerPause(state, runningContainer.Id, emitter, clock);
        expect("error" in result).toBe(true);
        if ("error" in result) expect(result.statusCode).toBe(409);
    });
});

describe("containerUnpause", () => {
    let env: TestEnv;
    beforeEach(() => { env = makeTestState(); });

    it("unpauses a paused container", () => {
        const { state, clock, emitter, events, runningContainer } = env;
        containerPause(state, runningContainer.Id, emitter, clock);
        events.length = 0; // clear pause event

        const result = containerUnpause(state, runningContainer.Id, emitter, clock);
        expect("ok" in result).toBe(true);

        const c = state.containers.get(runningContainer.Id)!;
        expect(c.State.Status).toBe("running");
        expect(c.State.Paused).toBe(false);

        expect(events).toHaveLength(1);
        expect(events[0].Action).toBe("unpause");
    });

    it("returns 409 if not paused", () => {
        const { state, clock, emitter, runningContainer } = env;
        const result = containerUnpause(state, runningContainer.Id, emitter, clock);
        expect("error" in result).toBe(true);
        if ("error" in result) expect(result.statusCode).toBe(409);
    });
});

// ---------------------------------------------------------------------------
// Container: remove
// ---------------------------------------------------------------------------

describe("containerRemove", () => {
    let env: TestEnv;
    beforeEach(() => { env = makeTestState(); });

    it("removes a stopped container", () => {
        const { state, clock, emitter, events, stoppedContainer } = env;
        const result = containerRemove(state, stoppedContainer.Id, emitter, clock);
        expect("ok" in result).toBe(true);
        expect(state.containers.has(stoppedContainer.Id)).toBe(false);

        expect(events).toHaveLength(1);
        expect(events[0].Action).toBe("destroy");
    });

    it("returns 409 for running container without force", () => {
        const { state, clock, emitter, runningContainer } = env;
        const result = containerRemove(state, runningContainer.Id, emitter, clock);
        expect("error" in result).toBe(true);
        if ("error" in result) expect(result.statusCode).toBe(409);
    });

    it("force removes a running container (stop events then destroy)", () => {
        const { state, clock, emitter, events, runningContainer } = env;
        const result = containerRemove(state, runningContainer.Id, emitter, clock, { force: true });
        expect("ok" in result).toBe(true);
        expect(state.containers.has(runningContainer.Id)).toBe(false);

        const actions = events.map((e) => e.Action);
        // stop: kill, die, stop; then destroy
        expect(actions).toEqual(["kill", "die", "stop", "destroy"]);
    });
});

// ---------------------------------------------------------------------------
// Container: kill
// ---------------------------------------------------------------------------

describe("containerKill", () => {
    let env: TestEnv;
    beforeEach(() => { env = makeTestState(); });

    it("SIGKILL sets exitCode 137", () => {
        const { state, clock, emitter, events, runningContainer } = env;
        const result = containerKill(state, runningContainer.Id, "SIGKILL", emitter, clock);
        expect("ok" in result).toBe(true);

        const c = state.containers.get(runningContainer.Id)!;
        expect(c.State.ExitCode).toBe(137);
        expect(c.State.Running).toBe(false);

        expect(events[0].Action).toBe("kill");
        expect(events[0].Actor.Attributes.signal).toBe("SIGKILL");
        expect(events[1].Action).toBe("die");
        expect(events[1].Actor.Attributes.exitCode).toBe("137");
    });

    it("SIGTERM sets exitCode 143", () => {
        const { state, clock, emitter, events, runningContainer } = env;
        const result = containerKill(state, runningContainer.Id, "SIGTERM", emitter, clock);
        expect("ok" in result).toBe(true);

        const c = state.containers.get(runningContainer.Id)!;
        expect(c.State.ExitCode).toBe(143);

        expect(events[1].Actor.Attributes.exitCode).toBe("143");
    });

    it("returns 409 if container not running", () => {
        const { state, clock, emitter, stoppedContainer } = env;
        const result = containerKill(state, stoppedContainer.Id, "SIGKILL", emitter, clock);
        expect("error" in result).toBe(true);
        if ("error" in result) expect(result.statusCode).toBe(409);
    });
});

// ---------------------------------------------------------------------------
// Container: rename
// ---------------------------------------------------------------------------

describe("containerRename", () => {
    let env: TestEnv;
    beforeEach(() => { env = makeTestState(); });

    it("renames container and emits rename event with oldName", () => {
        const { state, clock, emitter, events, runningContainer } = env;
        const result = containerRename(state, runningContainer.Id, "new-name", emitter, clock);
        expect("ok" in result).toBe(true);

        const c = state.containers.get(runningContainer.Id)!;
        expect(c.Name).toBe("/new-name");

        expect(events).toHaveLength(1);
        expect(events[0].Action).toBe("rename");
        expect(events[0].Actor.Attributes.oldName).toBe("myproject-api-1");
    });

    it("updates network container entries", () => {
        const { state, clock, emitter, network, runningContainer } = env;
        containerRename(state, runningContainer.Id, "new-name", emitter, clock);

        expect(network.Containers![runningContainer.Id].Name).toBe("new-name");
    });
});

// ---------------------------------------------------------------------------
// Container: create
// ---------------------------------------------------------------------------

describe("containerCreate", () => {
    let env: TestEnv;
    beforeEach(() => { env = makeTestState(); });

    it("fails with 400 when Image is missing", () => {
        const { state, clock, emitter, events } = env;
        const result = containerCreate(state, { } as any, emitter, clock);
        expect("error" in result).toBe(true);
        if ("error" in result) {
            expect(result.statusCode).toBe(400);
            expect(result.error).toBe("image is required");
        }
        expect(events).toHaveLength(0);
    });

    it("fails with 400 when Image is empty string", () => {
        const { state, clock, emitter, events } = env;
        const result = containerCreate(state, { Image: "" } as any, emitter, clock);
        expect("error" in result).toBe(true);
        if ("error" in result) {
            expect(result.statusCode).toBe(400);
        }
        expect(events).toHaveLength(0);
    });

    it("creates a container in created state", () => {
        const { state, clock, emitter, events } = env;
        const result = containerCreate(state, { Image: "myapp:latest", name: "test-ctr" }, emitter, clock);
        expect("ok" in result).toBe(true);
        if ("ok" in result) {
            const c = state.containers.get(result.ok.Id)!;
            expect(c.State.Status).toBe("created");
            expect(c.State.Running).toBe(false);
            expect(c.Name).toBe("/test-ctr");
            expect(c.Config.Image).toBe("myapp:latest");
        }

        expect(events).toHaveLength(1);
        expect(events[0].Action).toBe("create");
    });

    it("connects to default network at create time", () => {
        const { state, clock, emitter, network } = env;
        const result = containerCreate(state, {
            Image: "myapp:latest",
            name: "net-ctr",
            HostConfig: { NetworkMode: "myproject_default" },
        }, emitter, clock);
        expect("ok" in result).toBe(true);
        if ("ok" in result) {
            const c = state.containers.get(result.ok.Id)!;
            const ep = c.NetworkSettings.Networks!["myproject_default"];
            expect(ep).toBeDefined();
            expect(ep.NetworkID).toBe(network.Id);
            expect(ep.IPAddress).toBeTruthy();
            expect(ep.MacAddress).toBeTruthy();
        }
    });

    it("does not connect to network when mode is none", () => {
        const { state, clock, emitter } = env;
        const result = containerCreate(state, {
            Image: "myapp:latest",
            name: "no-net-ctr",
            HostConfig: { NetworkMode: "none" },
        }, emitter, clock);
        expect("ok" in result).toBe(true);
        if ("ok" in result) {
            const c = state.containers.get(result.ok.Id)!;
            expect(Object.keys(c.NetworkSettings.Networks!)).toHaveLength(0);
        }
    });

    it("containerStart adds created container to network Containers map", () => {
        const { state, clock, emitter, events, network } = env;
        const cr = containerCreate(state, {
            Image: "myapp:latest",
            name: "start-net-ctr",
            HostConfig: { NetworkMode: "myproject_default" },
        }, emitter, clock);
        expect("ok" in cr).toBe(true);
        if ("ok" in cr) {
            containerStart(state, cr.ok.Id, emitter, clock);
            expect(network.Containers![cr.ok.Id]).toBeDefined();
            expect(network.Containers![cr.ok.Id].Name).toBe("start-net-ctr");
        }
    });
});

// ---------------------------------------------------------------------------
// Network: create / remove
// ---------------------------------------------------------------------------

describe("networkCreate", () => {
    let env: TestEnv;
    beforeEach(() => { env = makeTestState(); });

    it("creates a network and emits create event", () => {
        const { state, clock, emitter, events } = env;
        const result = networkCreate(state, { Name: "test-net" }, emitter, clock);
        expect("ok" in result).toBe(true);
        if ("ok" in result) {
            const net = state.networks.get(result.ok.Id)!;
            expect(net.Name).toBe("test-net");
            expect(net.Driver).toBe("bridge");
        }

        expect(events).toHaveLength(1);
        expect(events[0].Type).toBe("network");
        expect(events[0].Action).toBe("create");
    });
});

describe("networkRemove", () => {
    let env: TestEnv;
    beforeEach(() => { env = makeTestState(); });

    it("removes a network and emits destroy event", () => {
        const { state, clock, emitter, events, network } = env;
        const result = networkRemove(state, network.Id, emitter, clock);
        expect("ok" in result).toBe(true);
        expect(state.networks.has(network.Id)).toBe(false);

        expect(events).toHaveLength(1);
        expect(events[0].Type).toBe("network");
        expect(events[0].Action).toBe("destroy");
    });

    it("returns 404 for unknown network", () => {
        const { state, clock, emitter } = env;
        const result = networkRemove(state, "nonexistent", emitter, clock);
        expect("error" in result).toBe(true);
        if ("error" in result) expect(result.statusCode).toBe(404);
    });
});

// ---------------------------------------------------------------------------
// Network: connect / disconnect
// ---------------------------------------------------------------------------

describe("networkConnect", () => {
    let env: TestEnv;
    beforeEach(() => { env = makeTestState(); });

    it("connects container to network, updates both sides", () => {
        const { state, clock, emitter, events, stoppedContainer } = env;
        // Create a second network
        const nr = networkCreate(state, { Name: "other-net" }, emitter, clock);
        expect("ok" in nr).toBe(true);
        events.length = 0; // clear create event

        if ("ok" in nr) {
            const netId = nr.ok.Id;
            const result = networkConnect(state, netId, stoppedContainer.Id, {}, emitter, clock);
            expect("ok" in result).toBe(true);

            // Container side
            const c = state.containers.get(stoppedContainer.Id)!;
            expect(c.NetworkSettings.Networks!["other-net"]).toBeDefined();
            expect(c.NetworkSettings.Networks!["other-net"].NetworkID).toBe(netId);

            // Network side
            const net = state.networks.get(netId)!;
            expect(net.Containers![stoppedContainer.Id]).toBeDefined();
            expect(net.Containers![stoppedContainer.Id].Name).toBe("myproject-web-1");

            expect(events).toHaveLength(1);
            expect(events[0].Type).toBe("network");
            expect(events[0].Action).toBe("connect");
            expect(events[0].Actor.Attributes.container).toBe(stoppedContainer.Id);
        }
    });
});

describe("networkDisconnect", () => {
    let env: TestEnv;
    beforeEach(() => { env = makeTestState(); });

    it("disconnects container from network, updates both sides", () => {
        const { state, clock, emitter, events, runningContainer, network } = env;
        const result = networkDisconnect(state, network.Id, runningContainer.Id, emitter, clock);
        expect("ok" in result).toBe(true);

        // Container side — no longer has the network
        const c = state.containers.get(runningContainer.Id)!;
        expect(c.NetworkSettings.Networks!["myproject_default"]).toBeUndefined();

        // Network side — container removed
        expect(network.Containers![runningContainer.Id]).toBeUndefined();

        expect(events).toHaveLength(1);
        expect(events[0].Type).toBe("network");
        expect(events[0].Action).toBe("disconnect");
    });
});

// ---------------------------------------------------------------------------
// Volume: create / remove
// ---------------------------------------------------------------------------

describe("volumeCreate", () => {
    let env: TestEnv;
    beforeEach(() => { env = makeTestState(); });

    it("creates a volume and emits create event", () => {
        const { state, clock, emitter, events } = env;
        const result = volumeCreate(state, { Name: "test-vol" }, emitter, clock);
        expect("ok" in result).toBe(true);
        if ("ok" in result) {
            expect(result.ok.Name).toBe("test-vol");
            expect(result.ok.Driver).toBe("local");
        }
        expect(state.volumes.has("test-vol")).toBe(true);

        expect(events).toHaveLength(1);
        expect(events[0].Type).toBe("volume");
        expect(events[0].Action).toBe("create");
    });
});

describe("volumeRemove", () => {
    let env: TestEnv;
    beforeEach(() => { env = makeTestState(); });

    it("removes a volume and emits destroy event", () => {
        const { state, clock, emitter, events } = env;
        volumeCreate(state, { Name: "test-vol" }, emitter, clock);
        events.length = 0;

        const result = volumeRemove(state, "test-vol", emitter, clock);
        expect("ok" in result).toBe(true);
        expect(state.volumes.has("test-vol")).toBe(false);

        expect(events).toHaveLength(1);
        expect(events[0].Type).toBe("volume");
        expect(events[0].Action).toBe("destroy");
    });

    it("returns 404 for unknown volume", () => {
        const { state, clock, emitter } = env;
        const result = volumeRemove(state, "nonexistent", emitter, clock);
        expect("error" in result).toBe(true);
        if ("error" in result) expect(result.statusCode).toBe(404);
    });
});

// ---------------------------------------------------------------------------
// Image: remove
// ---------------------------------------------------------------------------

describe("imageRemove", () => {
    let env: TestEnv;
    beforeEach(() => { env = makeTestState(); });

    it("removes an image by tag with untag + delete events", () => {
        const { state, clock, emitter, events, image } = env;
        // Stop all containers so image can be removed
        containerStop(state, env.runningContainer.Id, emitter, clock);
        events.length = 0;

        const result = imageRemove(state, "myapp:latest", emitter, clock);
        expect("ok" in result).toBe(true);
        expect(state.images.has(image.Id)).toBe(false);

        if ("ok" in result) {
            const entries = result.ok;
            expect(entries.some((e) => e.Untagged === "myapp:latest")).toBe(true);
            expect(entries.some((e) => e.Deleted === image.Id)).toBe(true);
        }

        const actions = events.map((e) => e.Action);
        expect(actions).toContain("untag");
        expect(actions).toContain("delete");
    });

    it("returns 404 for unknown image", () => {
        const { state, clock, emitter } = env;
        const result = imageRemove(state, "nonexistent:v1", emitter, clock);
        expect("error" in result).toBe(true);
        if ("error" in result) expect(result.statusCode).toBe(404);
    });
});

// ---------------------------------------------------------------------------
// Image: prune
// ---------------------------------------------------------------------------

describe("imagePrune", () => {
    let env: TestEnv;
    beforeEach(() => { env = makeTestState(); });

    it("prunes only unreferenced images when all=true", () => {
        const { state, clock, emitter, events } = env;

        // Add an unreferenced image
        const unusedImage: ImageInspect = {
            Id: "sha256:unused00112233445566778899aabbccddee00112233445566778899aabbccdd",
            RepoTags: ["old:v1"],
            RepoDigests: [],
            Created: "2025-01-01T00:00:00Z",
            Architecture: "amd64",
            Os: "linux",
            Size: 30_000_000,
            RootFS: { Type: "layers" },
        };
        state.images.set(unusedImage.Id, unusedImage);

        const result = imagePrune(state, true, emitter, clock);
        expect("ok" in result).toBe(true);
        if ("ok" in result) {
            // Only unused image should be pruned (myapp:latest is used by containers)
            expect(result.ok.ImagesDeleted.some((e) => e.Deleted === unusedImage.Id)).toBe(true);
            expect(result.ok.SpaceReclaimed).toBe(30_000_000);
        }

        // myapp:latest should still exist
        expect(state.images.has(env.image.Id)).toBe(true);
        // unused should be gone
        expect(state.images.has(unusedImage.Id)).toBe(false);
    });
});

// ---------------------------------------------------------------------------
// Resolution: mutations accept short ID prefix and name
// ---------------------------------------------------------------------------

describe("resolution", () => {
    let env: TestEnv;
    beforeEach(() => { env = makeTestState(); });

    it("containerStart accepts container name", () => {
        const { state, clock, emitter, stoppedContainer } = env;
        const result = containerStart(state, "myproject-web-1", emitter, clock);
        expect("ok" in result).toBe(true);
    });

    it("containerStop accepts short ID prefix", () => {
        const { state, clock, emitter, runningContainer } = env;
        const prefix = runningContainer.Id.slice(0, 12);
        const result = containerStop(state, prefix, emitter, clock);
        expect("ok" in result).toBe(true);
    });

    it("networkRemove accepts network name", () => {
        const { state, clock, emitter, network } = env;
        const result = networkRemove(state, "myproject_default", emitter, clock);
        expect("ok" in result).toBe(true);
        expect(state.networks.has(network.Id)).toBe(false);
    });
});
