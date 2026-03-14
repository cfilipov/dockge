import type { MockState } from "./state.js";
import type { ContainerInspect, NetworkInspect, VolumeInspect, ImageInspect, EndpointSettings, ExecInspect } from "./types.js";
import type { Clock } from "./clock.js";
import type { EventEmitter } from "./events.js";
import { makeEvent } from "./events.js";
import { resolveByIdOrName } from "./name-resolution.js";
import { deterministicId, deterministicInt, deterministicIp, deterministicMac, serviceSeed, networkSeed, imageSeed } from "./deterministic.js";

// ---------------------------------------------------------------------------
// Result type
// ---------------------------------------------------------------------------

export type MutationResult<T = void> = { ok: T } | { error: string; statusCode: number };

function ok(): MutationResult<void>;
function ok<T>(value: T): MutationResult<T>;
function ok<T>(value?: T): MutationResult<T> {
    return { ok: value as T };
}

function fail(statusCode: number, error: string): MutationResult<never> {
    return { error, statusCode };
}

// ---------------------------------------------------------------------------
// Private helpers — resolution
// ---------------------------------------------------------------------------

function resolveContainer(
    state: MockState,
    id: string,
): MutationResult<ContainerInspect> {
    const result = resolveByIdOrName(state.containers, id, (c) => c.Name, (c) => c.Id);
    if ("error" in result) return fail(404, `No such container: ${id}`);
    return ok(result.found);
}

function resolveNetwork(
    state: MockState,
    id: string,
): MutationResult<NetworkInspect> {
    const result = resolveByIdOrName(state.networks, id, (n) => n.Name, (n) => n.Id);
    if ("error" in result) return fail(404, `network ${id} not found`);
    return ok(result.found);
}

function resolveImage(
    state: MockState,
    nameOrId: string,
): MutationResult<ImageInspect> {
    // Try by ID first (exact or prefix)
    const byId = resolveByIdOrName(state.images, nameOrId, (i) => i.RepoTags[0] || "", (i) => i.Id);
    if ("found" in byId) return ok(byId.found);

    // Try with sha256: prefix
    if (!nameOrId.startsWith("sha256:")) {
        const withPrefix = resolveByIdOrName(state.images, "sha256:" + nameOrId, (i) => i.RepoTags[0] || "", (i) => i.Id);
        if ("found" in withPrefix) return ok(withPrefix.found);
    }

    // Scan RepoTags for tag match
    for (const img of state.images.values()) {
        for (const tag of img.RepoTags) {
            if (tag === nameOrId) return ok(img);
        }
    }

    return fail(404, `No such image: ${nameOrId}`);
}

// ---------------------------------------------------------------------------
// Private helpers — attributes & network bookkeeping
// ---------------------------------------------------------------------------

function containerAttrs(c: ContainerInspect): Record<string, string> {
    const name = c.Name.replace(/^\//, "");
    const attrs: Record<string, string> = { name, image: c.Config.Image || "" };
    if (c.Config.Labels) {
        for (const [k, v] of Object.entries(c.Config.Labels)) {
            attrs[k] = v;
        }
    }
    return attrs;
}

function addContainerToNetworks(state: MockState, container: ContainerInspect): void {
    const networks = container.NetworkSettings.Networks;
    if (!networks) return;

    for (const [, endpoint] of Object.entries(networks)) {
        const network = state.networks.get(endpoint.NetworkID);
        if (network && network.Containers) {
            network.Containers[container.Id] = {
                Name: container.Name.replace(/^\//, ""),
                EndpointID: endpoint.EndpointID,
                MacAddress: endpoint.MacAddress || "",
                IPv4Address: endpoint.IPAddress ? `${endpoint.IPAddress}/${endpoint.IPPrefixLen}` : "",
            };
        }
    }
}

function removeContainerFromNetworks(state: MockState, container: ContainerInspect): void {
    const networks = container.NetworkSettings.Networks;
    if (!networks) return;

    for (const [, endpoint] of Object.entries(networks)) {
        const network = state.networks.get(endpoint.NetworkID);
        if (network && network.Containers) {
            delete network.Containers[container.Id];
        }
    }
}

// ---------------------------------------------------------------------------
// Container mutations
// ---------------------------------------------------------------------------

export function containerStart(
    state: MockState,
    id: string,
    emitter: EventEmitter,
    clock: Clock,
    opts?: { e2eMode?: boolean },
): MutationResult {
    const r = resolveContainer(state, id);
    if ("error" in r) return r;
    const c = r.ok;

    if (c.State.Running) return fail(304, "container already started");

    // Default e2eMode to true — the mock is primarily for testing and
    // "healthy" immediately is the safer default.
    const e2e = opts?.e2eMode ?? true;

    const now = clock.now().toISOString();
    c.State.Status = "running";
    c.State.Running = true;
    c.State.Paused = false;
    c.State.Pid = deterministicInt(c.Id + now, 1000, 65535);
    c.State.StartedAt = now;
    c.State.FinishedAt = "0001-01-01T00:00:00Z";
    c.State.ExitCode = 0;
    c.State.Error = "";

    // Restore health status if container has a healthcheck configured
    const hcTest = c.Config.Healthcheck?.Test;
    if (hcTest && hcTest.length > 0) {
        c.State.Health = {
            Status: e2e ? "healthy" : "starting",
            FailingStreak: 0,
            Log: [],
        };
    } else {
        delete c.State.Health;
    }

    addContainerToNetworks(state, c);
    emitter.emit(makeEvent(clock, "container", "start", c.Id, containerAttrs(c)));

    return ok();
}

export function containerStop(
    state: MockState,
    id: string,
    emitter: EventEmitter,
    clock: Clock,
): MutationResult {
    const r = resolveContainer(state, id);
    if ("error" in r) return r;
    const c = r.ok;

    if (!c.State.Running && !c.State.Paused) return fail(304, "container already stopped");

    const now = clock.now().toISOString();
    const attrs = containerAttrs(c);

    c.State.Status = "exited";
    c.State.Running = false;
    c.State.Paused = false;
    c.State.Pid = 0;
    c.State.FinishedAt = now;
    c.State.ExitCode = 0;
    delete c.State.Health;

    removeContainerFromNetworks(state, c);

    emitter.emit(makeEvent(clock, "container", "kill", c.Id, { ...attrs, signal: "SIGTERM" }));
    emitter.emit(makeEvent(clock, "container", "die", c.Id, { ...attrs, exitCode: "0" }));
    emitter.emit(makeEvent(clock, "container", "stop", c.Id, attrs));

    return ok();
}

export function containerRestart(
    state: MockState,
    id: string,
    emitter: EventEmitter,
    clock: Clock,
): MutationResult {
    const r = resolveContainer(state, id);
    if ("error" in r) return r;
    const c = r.ok;

    // Stop if running
    if (c.State.Running || c.State.Paused) {
        const stopResult = containerStop(state, id, emitter, clock);
        if ("error" in stopResult && stopResult.statusCode !== 304) return stopResult;
    }

    // Start
    const startResult = containerStart(state, id, emitter, clock);
    if ("error" in startResult) return startResult;

    emitter.emit(makeEvent(clock, "container", "restart", c.Id, containerAttrs(c)));

    return ok();
}

export function containerPause(
    state: MockState,
    id: string,
    emitter: EventEmitter,
    clock: Clock,
): MutationResult {
    const r = resolveContainer(state, id);
    if ("error" in r) return r;
    const c = r.ok;

    if (!c.State.Running) return fail(409, "container is not running");
    if (c.State.Paused) return fail(409, "container is already paused");

    c.State.Status = "paused";
    c.State.Paused = true;

    emitter.emit(makeEvent(clock, "container", "pause", c.Id, containerAttrs(c)));

    return ok();
}

export function containerUnpause(
    state: MockState,
    id: string,
    emitter: EventEmitter,
    clock: Clock,
): MutationResult {
    const r = resolveContainer(state, id);
    if ("error" in r) return r;
    const c = r.ok;

    if (!c.State.Paused) return fail(409, "container is not paused");

    c.State.Status = "running";
    c.State.Paused = false;

    emitter.emit(makeEvent(clock, "container", "unpause", c.Id, containerAttrs(c)));

    return ok();
}

export function containerRemove(
    state: MockState,
    id: string,
    emitter: EventEmitter,
    clock: Clock,
    opts: { force?: boolean } = {},
): MutationResult {
    const r = resolveContainer(state, id);
    if ("error" in r) return r;
    const c = r.ok;

    if (c.State.Running) {
        if (!opts.force) return fail(409, `You cannot remove a running container ${c.Id}. Stop the container before attempting removal or force remove`);
        // Force stop first (also removes from networks)
        containerStop(state, c.Id, emitter, clock);
    }

    state.containers.delete(c.Id);

    emitter.emit(makeEvent(clock, "container", "destroy", c.Id, containerAttrs(c)));

    return ok();
}

export interface ContainerCreateConfig {
    name?: string;
    Image: string;
    Cmd?: string[];
    Entrypoint?: string[];
    Env?: string[];
    Labels?: Record<string, string>;
    ExposedPorts?: Record<string, Record<string, never>>;
    HostConfig?: {
        NetworkMode?: string;
        PortBindings?: Record<string, Array<{ HostIp: string; HostPort: string }>>;
        Binds?: string[];
        RestartPolicy?: { Name: string; MaximumRetryCount: number };
    };
}

export function containerCreate(
    state: MockState,
    config: ContainerCreateConfig,
    emitter: EventEmitter,
    clock: Clock,
): MutationResult<{ Id: string }> {
    if (!config.Image) {
        return fail(400, "image is required");
    }
    const now = clock.now();
    const name = config.name || uniquifyName(config);
    const nameWithSlash = name.startsWith("/") ? name : `/${name}`;
    const seed = nameWithSlash;
    const containerId = deterministicId(seed, "container-id");

    const imageRef = config.Image;
    const imgResult = resolveImage(state, imageRef);
    const imageId = "found" in imgResult ? (imgResult as { ok: ImageInspect }).ok.Id : `sha256:${deterministicId(imageSeed(imageRef), "image-id")}`;

    const container: ContainerInspect = {
        Id: containerId,
        Created: now.toISOString(),
        Path: "",
        Args: [],
        State: {
            Status: "created",
            Running: false,
            Paused: false,
            Restarting: false,
            OOMKilled: false,
            Dead: false,
            Pid: 0,
            ExitCode: 0,
            Error: "",
            StartedAt: "0001-01-01T00:00:00Z",
            FinishedAt: "0001-01-01T00:00:00Z",
        },
        Image: imageId,
        Name: nameWithSlash,
        HostConfig: {
            NetworkMode: config.HostConfig?.NetworkMode || "default",
            RestartPolicy: config.HostConfig?.RestartPolicy || { Name: "", MaximumRetryCount: 0 },
            PortBindings: config.HostConfig?.PortBindings,
            Binds: config.HostConfig?.Binds,
        },
        Mounts: [],
        Config: {
            Image: imageRef,
            Cmd: config.Cmd,
            Entrypoint: config.Entrypoint,
            Env: config.Env,
            Labels: config.Labels || {},
            ExposedPorts: config.ExposedPorts,
        },
        NetworkSettings: {
            Networks: {},
        },
    };

    state.containers.set(containerId, container);

    // Connect to default network (matches real Docker behavior)
    const networkMode = container.HostConfig.NetworkMode || "default";
    if (networkMode !== "none" && networkMode !== "host") {
        // Find the network by name
        for (const net of state.networks.values()) {
            if (net.Name === networkMode) {
                const epSeed = containerId + net.Id;
                const subnet = net.IPAM.Config?.[0]?.Subnet || "172.18.0.0/16";
                const gateway = net.IPAM.Config?.[0]?.Gateway || "";
                const endpoint: EndpointSettings = {
                    NetworkID: net.Id,
                    EndpointID: deterministicId(epSeed, "endpoint-id"),
                    Gateway: gateway,
                    IPAddress: deterministicIp(epSeed, subnet),
                    IPPrefixLen: parseInt(subnet.split("/")[1] || "16", 10),
                    MacAddress: deterministicMac(epSeed),
                };
                container.NetworkSettings.Networks![net.Name] = endpoint;
                break;
            }
        }
    }

    emitter.emit(makeEvent(clock, "container", "create", containerId, containerAttrs(container)));

    return ok({ Id: containerId });
}

function uniquifyName(config: ContainerCreateConfig): string {
    const parts = [
        config.Image,
        ...(config.Cmd || []),
        ...(config.Env || []).sort(),
        ...Object.entries(config.Labels || {}).sort().map(([k, v]) => `${k}=${v}`),
    ];
    const hash = deterministicId(parts.join("\0"), "container-name");
    return `mock-${hash.slice(0, 12)}`;
}

export function containerRename(
    state: MockState,
    id: string,
    newName: string,
    emitter: EventEmitter,
    clock: Clock,
): MutationResult {
    const r = resolveContainer(state, id);
    if ("error" in r) return r;
    const c = r.ok;

    const oldName = c.Name.replace(/^\//, "");
    const newNameWithSlash = newName.startsWith("/") ? newName : `/${newName}`;
    c.Name = newNameWithSlash;

    // Update network container entries
    const networks = c.NetworkSettings.Networks;
    if (networks) {
        for (const [, endpoint] of Object.entries(networks)) {
            const network = state.networks.get(endpoint.NetworkID);
            if (network?.Containers?.[c.Id]) {
                network.Containers[c.Id].Name = newName.replace(/^\//, "");
            }
        }
    }

    emitter.emit(makeEvent(clock, "container", "rename", c.Id, { ...containerAttrs(c), oldName }));

    return ok();
}

export function containerKill(
    state: MockState,
    id: string,
    signal: string,
    emitter: EventEmitter,
    clock: Clock,
): MutationResult {
    const r = resolveContainer(state, id);
    if ("error" in r) return r;
    const c = r.ok;

    if (!c.State.Running && !c.State.Paused) return fail(409, "container is not running");

    const sig = signal.toUpperCase();
    const exitCode = sig === "SIGKILL" || sig === "9" ? 137 : 143;
    const now = clock.now().toISOString();
    const attrs = containerAttrs(c);

    c.State.Status = "exited";
    c.State.Running = false;
    c.State.Paused = false;
    c.State.Pid = 0;
    c.State.FinishedAt = now;
    c.State.ExitCode = exitCode;
    delete c.State.Health;

    removeContainerFromNetworks(state, c);

    emitter.emit(makeEvent(clock, "container", "kill", c.Id, { ...attrs, signal: sig }));
    emitter.emit(makeEvent(clock, "container", "die", c.Id, { ...attrs, exitCode: String(exitCode) }));

    return ok();
}

// ---------------------------------------------------------------------------
// Network mutations
// ---------------------------------------------------------------------------

export interface NetworkCreateConfig {
    Name: string;
    Driver?: string;
    Internal?: boolean;
    Attachable?: boolean;
    Labels?: Record<string, string>;
    IPAM?: {
        Driver?: string;
        Config?: Array<{ Subnet?: string; Gateway?: string }>;
    };
    Options?: Record<string, string>;
}

export function networkCreate(
    state: MockState,
    config: NetworkCreateConfig,
    emitter: EventEmitter,
    clock: Clock,
): MutationResult<{ Id: string }> {
    const seed = networkSeed(config.Name);
    const id = deterministicId(seed, "network-id");
    const driver = config.Driver || "bridge";

    const subnet = config.IPAM?.Config?.[0]?.Subnet || `172.${deterministicInt(seed + "subnet-b", 18, 31)}.0.0/16`;
    const gateway = config.IPAM?.Config?.[0]?.Gateway || subnet.replace(/\.0\.0\/\d+$/, ".0.1");

    const network: NetworkInspect = {
        Name: config.Name,
        Id: id,
        Created: clock.now().toISOString(),
        Scope: "local",
        Driver: driver,
        EnableIPv4: true,
        EnableIPv6: false,
        IPAM: {
            Driver: config.IPAM?.Driver || "default",
            Config: [{ Subnet: subnet, Gateway: gateway }],
            Options: {},
        },
        Internal: config.Internal || false,
        Attachable: config.Attachable || false,
        Ingress: false,
        ConfigFrom: { Network: "" },
        ConfigOnly: false,
        Containers: {},
        Options: config.Options || {},
        Labels: config.Labels || {},
    };

    state.networks.set(id, network);

    emitter.emit(makeEvent(clock, "network", "create", id, { name: config.Name, type: driver }));

    return ok({ Id: id });
}

export function networkRemove(
    state: MockState,
    id: string,
    emitter: EventEmitter,
    clock: Clock,
): MutationResult {
    const r = resolveNetwork(state, id);
    if ("error" in r) return r;
    const net = r.ok;

    state.networks.delete(net.Id);

    emitter.emit(makeEvent(clock, "network", "destroy", net.Id, { name: net.Name, type: net.Driver }));

    return ok();
}

export function networkConnect(
    state: MockState,
    netId: string,
    ctrId: string,
    config: { IPv4Address?: string; IPv6Address?: string } = {},
    emitter: EventEmitter,
    clock: Clock,
): MutationResult {
    const nr = resolveNetwork(state, netId);
    if ("error" in nr) return nr;
    const net = nr.ok;

    const cr = resolveContainer(state, ctrId);
    if ("error" in cr) return cr;
    const ctr = cr.ok;

    // Build endpoint
    const epSeed = ctr.Id + net.Id;
    const subnet = net.IPAM.Config?.[0]?.Subnet || "172.18.0.0/16";
    const gateway = net.IPAM.Config?.[0]?.Gateway || "";
    const endpoint: EndpointSettings = {
        NetworkID: net.Id,
        EndpointID: deterministicId(epSeed, "endpoint-id"),
        Gateway: gateway,
        IPAddress: config.IPv4Address || deterministicIp(epSeed, subnet),
        IPPrefixLen: parseInt(subnet.split("/")[1] || "16", 10),
        MacAddress: deterministicMac(epSeed),
    };

    // Add to container's NetworkSettings
    if (!ctr.NetworkSettings.Networks) ctr.NetworkSettings.Networks = {};
    ctr.NetworkSettings.Networks[net.Name] = endpoint;

    // Add to network's Containers
    if (!net.Containers) net.Containers = {};
    net.Containers[ctr.Id] = {
        Name: ctr.Name.replace(/^\//, ""),
        EndpointID: endpoint.EndpointID,
        MacAddress: endpoint.MacAddress || "",
        IPv4Address: endpoint.IPAddress ? `${endpoint.IPAddress}/${endpoint.IPPrefixLen}` : "",
    };

    emitter.emit(makeEvent(clock, "network", "connect", net.Id, {
        name: net.Name,
        container: ctr.Id,
        type: net.Driver,
    }));

    return ok();
}

export function networkDisconnect(
    state: MockState,
    netId: string,
    ctrId: string,
    emitter: EventEmitter,
    clock: Clock,
): MutationResult {
    const nr = resolveNetwork(state, netId);
    if ("error" in nr) return nr;
    const net = nr.ok;

    const cr = resolveContainer(state, ctrId);
    if ("error" in cr) return cr;
    const ctr = cr.ok;

    // Remove from container's NetworkSettings
    if (ctr.NetworkSettings.Networks) {
        delete ctr.NetworkSettings.Networks[net.Name];
    }

    // Remove from network's Containers
    if (net.Containers) {
        delete net.Containers[ctr.Id];
    }

    emitter.emit(makeEvent(clock, "network", "disconnect", net.Id, {
        name: net.Name,
        container: ctr.Id,
        type: net.Driver,
    }));

    return ok();
}

// ---------------------------------------------------------------------------
// Volume mutations
// ---------------------------------------------------------------------------

export interface VolumeCreateConfig {
    Name: string;
    Driver?: string;
    Labels?: Record<string, string>;
    DriverOpts?: Record<string, string>;
}

export function volumeCreate(
    state: MockState,
    config: VolumeCreateConfig,
    emitter: EventEmitter,
    clock: Clock,
): MutationResult<VolumeInspect> {
    const name = config.Name;
    const driver = config.Driver || "local";

    const vol: VolumeInspect = {
        Name: name,
        Driver: driver,
        Mountpoint: `/var/lib/docker/volumes/${name}/_data`,
        CreatedAt: clock.now().toISOString(),
        Labels: config.Labels || {},
        Scope: "local",
        Options: config.DriverOpts,
    };

    state.volumes.set(name, vol);

    emitter.emit(makeEvent(clock, "volume", "create", name, { driver }));

    return ok(vol);
}

export function volumeRemove(
    state: MockState,
    name: string,
    emitter: EventEmitter,
    clock: Clock,
): MutationResult {
    const vol = state.volumes.get(name);
    if (!vol) return fail(404, `get ${name}: no such volume`);

    state.volumes.delete(name);

    emitter.emit(makeEvent(clock, "volume", "destroy", name, { driver: vol.Driver }));

    return ok();
}

// ---------------------------------------------------------------------------
// Image mutations
// ---------------------------------------------------------------------------

export function imageRemove(
    state: MockState,
    nameOrId: string,
    emitter: EventEmitter,
    clock: Clock,
    opts: { force?: boolean } = {},
): MutationResult<Array<{ Untagged?: string; Deleted?: string }>> {
    const r = resolveImage(state, nameOrId);
    if ("error" in r) return r;
    const img = r.ok;

    // Check if any running container uses this image (unless force)
    if (!opts.force) {
        for (const c of state.containers.values()) {
            if (c.Image === img.Id && c.State.Running) {
                return fail(409, `unable to remove ${img.Id}: image is being used by running container ${c.Id}`);
            }
        }
    }

    const result: Array<{ Untagged?: string; Deleted?: string }> = [];

    // Untag events
    for (const tag of img.RepoTags) {
        result.push({ Untagged: tag });
        emitter.emit(makeEvent(clock, "image", "untag", img.Id, { name: tag }));
    }

    // Delete
    result.push({ Deleted: img.Id });
    state.images.delete(img.Id);
    emitter.emit(makeEvent(clock, "image", "delete", img.Id, { name: img.Id }));

    return ok(result);
}

export function imagePrune(
    state: MockState,
    all: boolean,
    emitter: EventEmitter,
    clock: Clock,
): MutationResult<{ ImagesDeleted: Array<{ Untagged?: string; Deleted?: string }>; SpaceReclaimed: number }> {
    // Find which images are referenced by containers
    const usedImageIds = new Set<string>();
    for (const c of state.containers.values()) {
        usedImageIds.add(c.Image);
    }

    const deleted: Array<{ Untagged?: string; Deleted?: string }> = [];
    let spaceReclaimed = 0;

    for (const img of [...state.images.values()]) {
        if (usedImageIds.has(img.Id)) continue;

        // Without `all`, only prune dangling (no tags)
        if (!all && img.RepoTags.length > 0 && img.RepoTags[0] !== "<none>:<none>") continue;

        for (const tag of img.RepoTags) {
            deleted.push({ Untagged: tag });
            emitter.emit(makeEvent(clock, "image", "untag", img.Id, { name: tag }));
        }

        deleted.push({ Deleted: img.Id });
        spaceReclaimed += img.Size;
        state.images.delete(img.Id);
        emitter.emit(makeEvent(clock, "image", "delete", img.Id, { name: img.Id }));
    }

    return ok({ ImagesDeleted: deleted, SpaceReclaimed: spaceReclaimed });
}

// ---------------------------------------------------------------------------
// Exec mutations
// ---------------------------------------------------------------------------

export interface ExecCreateConfig {
    Cmd: string[];
    AttachStdin: boolean;
    AttachStdout: boolean;
    AttachStderr: boolean;
    Tty: boolean;
    User?: string;
}

export function execCreate(
    state: MockState,
    containerId: string,
    config: ExecCreateConfig,
    clock: Clock,
): MutationResult<{ Id: string }> {
    const r = resolveContainer(state, containerId);
    if ("error" in r) return r;
    const c = r.ok;

    if (!c.State.Running) return fail(409, `Container ${c.Id} is not running`);

    const execId = deterministicId(
        c.Id + JSON.stringify(config.Cmd) + clock.now().toISOString(),
        "exec-id",
    );

    const user = config.User || c.Config.User || "root";
    const exec: ExecInspect = {
        ID: execId,
        Running: false,
        ExitCode: 0,
        ProcessConfig: {
            tty: config.Tty,
            entrypoint: config.Cmd[0] || "/bin/sh",
            arguments: config.Cmd.slice(1),
            user,
        },
        OpenStdin: config.AttachStdin,
        OpenStdout: config.AttachStdout,
        OpenStderr: config.AttachStderr,
        ContainerID: c.Id,
        Pid: 0,
    };

    // Track exec ID on the container
    if (!c.ExecIDs) c.ExecIDs = [];
    c.ExecIDs.push(execId);

    state.execSessions.set(execId, exec);

    return ok({ Id: execId });
}

export function execInspect(
    state: MockState,
    execId: string,
): MutationResult<ExecInspect> {
    const exec = state.execSessions.get(execId);
    if (!exec) return fail(404, `No such exec instance: ${execId}`);
    return ok(exec);
}
