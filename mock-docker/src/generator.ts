import type {
    ContainerInspect,
    ContainerState,
    ContainerConfig,
    HostConfig,
    NetworkSettings,
    EndpointSettings,
    MountPoint,
    NetworkInspect,
    VolumeInspect,
    ImageInspect,
    PortBinding,
    RestartPolicy,
    HealthState,
} from "./types.js";
import type { ParsedCompose, ParsedService, ParsedNetwork, ParsedPort, ParsedVolumeMount, ParsedVolume } from "./compose-parser.js";
import type { MockStackConfig, MockServiceOverride } from "./mock-config.js";
import type { Clock } from "./clock.js";
import {
    deterministicId,
    deterministicMac,
    deterministicIp,
    deterministicTimestamp,
    deterministicInt,
    serviceSeed,
    networkSeed,
    imageSeed,
} from "./deterministic.js";

export interface GeneratorInput {
    project: string;
    stackDir: string;
    composeFilePath: string;
    parsed: ParsedCompose;
    mockConfig: MockStackConfig;
    clock: Clock;
    existingNetworks?: Map<string, NetworkInspect>;
    precapturedImages?: Map<string, ImageInspect>;
}

export interface GeneratedStack {
    containers: ContainerInspect[];
    networks: NetworkInspect[];
    volumes: VolumeInspect[];
    images: ImageInspect[];
}

const COMPOSE_VERSION = "2.30.0";

export function generateStack(input: GeneratorInput): GeneratedStack {
    const { project, parsed, mockConfig, clock } = input;
    const baseTime = clock.now().toISOString();

    // Generate networks first (needed for container endpoint resolution)
    const networks = generateNetworks(input, baseTime);
    const networkMap = new Map<string, NetworkInspect>();
    for (const net of networks) {
        networkMap.set(net.Name, net);
    }

    // Merge with existing networks for external resolution
    if (input.existingNetworks) {
        for (const [id, net] of input.existingNetworks) {
            if (!networkMap.has(net.Name)) {
                networkMap.set(net.Name, net);
            }
        }
    }

    // Generate volumes
    const volumes = generateVolumes(input, baseTime);

    // Generate images (deduplicated by ref)
    // Use precaptured images when available, fall back to synthetic
    const imageRefs = new Set<string>();
    for (const [name, svc] of Object.entries(parsed.services)) {
        const ref = resolveImageRef(project, name, svc);
        imageRefs.add(ref);
    }
    const imageMap = new Map<string, ImageInspect>();
    const images: ImageInspect[] = [];
    for (const ref of imageRefs) {
        const precaptured = lookupPrecaptured(ref, input.precapturedImages);
        const img = precaptured ?? generateImage(ref, baseTime);
        images.push(img);
        imageMap.set(ref, img);
    }

    // Generate containers
    const containers: ContainerInspect[] = [];
    // Pre-generate container IDs for service: network mode resolution
    const containerIds = new Map<string, string>();
    for (const name of Object.keys(parsed.services)) {
        containerIds.set(name, deterministicId(serviceSeed(project, name), "container-id"));
    }

    for (const [name, svc] of Object.entries(parsed.services)) {
        const override = mockConfig.services[name] || {};
        const container = generateContainer(input, name, svc, override, baseTime, networkMap, containerIds, imageMap);
        containers.push(container);
    }

    return { containers, networks, volumes, images };
}

// --- Image ref resolution ---

function resolveImageRef(project: string, serviceName: string, svc: ParsedService): string {
    if (svc.image) {
        const normalized = normalizeImageRef(svc.image);
        return sanitizeUnresolvedVars(normalized, project, serviceName);
    }
    return `${project}-${serviceName}:latest`;
}

function normalizeImageRef(ref: string): string {
    if (!ref.includes(":")) return ref + ":latest";
    return ref;
}

/** Replace unresolved env vars (e.g. ${TAG}, $VERSION) with safe fallbacks. */
function sanitizeUnresolvedVars(ref: string, project: string, serviceName: string): string {
    const hasUnresolved = /\$\{|(\$[A-Z_])/.test(ref);
    if (!hasUnresolved) return ref;

    const colonIdx = ref.indexOf(":");
    if (colonIdx === -1) {
        // Entire ref has unresolved vars — use fallback
        return `${project}-${serviceName}:latest`;
    }

    const repo = ref.slice(0, colonIdx);
    const tag = ref.slice(colonIdx + 1);

    // If repo contains unresolved vars, whole ref is unusable
    if (/\$\{|(\$[A-Z_])/.test(repo)) {
        return `${project}-${serviceName}:latest`;
    }

    // Only tag has unresolved vars — replace with latest
    if (/\$\{|(\$[A-Z_])/.test(tag)) {
        return `${repo}:latest`;
    }

    return ref;
}

// --- Container generation ---

function generateContainer(
    input: GeneratorInput,
    serviceName: string,
    svc: ParsedService,
    override: MockServiceOverride,
    baseTime: string,
    networkMap: Map<string, NetworkInspect>,
    containerIds: Map<string, string>,
    imageMap: Map<string, ImageInspect>,
): ContainerInspect {
    const { project, stackDir, composeFilePath } = input;
    const seed = serviceSeed(project, serviceName);
    const containerId = containerIds.get(serviceName)!;
    const imageRef = resolveImageRef(project, serviceName, svc);

    // Resolve image defaults (Layer 1)
    const resolvedImage = imageMap.get(imageRef);
    const imageConfig = resolvedImage?.Config;

    // State
    const stateStr = override.state || "running";
    const exitCode = override.exitCode ?? (stateStr === "exited" ? 0 : 0);
    const state = buildContainerState(stateStr, exitCode, seed, baseTime, svc, override);

    // Container name
    const containerName = svc.containerName || `/${project}-${serviceName}-1`;
    const nameWithSlash = containerName.startsWith("/") ? containerName : `/${containerName}`;

    // Command / entrypoint (Layer 2: compose overrides image defaults)
    const composeCmd = resolveCommand(svc.command);
    const composeEntrypoint = resolveCommand(svc.entrypoint);
    const imageCmd = imageConfig?.Cmd ?? [];
    const imageEntrypoint = imageConfig?.Entrypoint ?? [];

    const cmd = composeCmd.length > 0 ? composeCmd : imageCmd;
    const entrypoint = composeEntrypoint.length > 0 ? composeEntrypoint : imageEntrypoint;
    const path = entrypoint.length > 0 ? entrypoint[0] : (cmd.length > 0 ? cmd[0] : "");
    const args = entrypoint.length > 0 ? [...entrypoint.slice(1), ...cmd] : cmd.slice(1);

    // Environment (Layer 2: image env as base, compose env overrides by key)
    const mergedEnv = mergeEnvironment(imageConfig?.Env, svc.environment);

    // Labels
    const labels = buildContainerLabels(project, serviceName, composeFilePath, stackDir, imageRef, svc);

    // Mock override labels
    if (override.updateAvailable) {
        labels["com.portge.mock.update_available"] = "true";
    }
    if (override.needsRecreation) {
        labels["com.portge.mock.needs_recreation"] = "true";
    }

    // ExposedPorts (Layer 2: merge image ExposedPorts with compose ports/expose)
    const exposedPorts: Record<string, Record<string, never>> = {};
    if (imageConfig?.ExposedPorts) {
        for (const key of Object.keys(imageConfig.ExposedPorts)) {
            exposedPorts[key] = {};
        }
    }
    for (const p of svc.ports) {
        exposedPorts[`${p.target}/${p.protocol}`] = {};
    }
    for (const e of svc.expose) {
        exposedPorts[`${e}/tcp`] = {};
    }

    // Volumes (Layer 2: merge image Volumes with compose volume targets)
    const mergedVolumes: Record<string, Record<string, never>> | undefined =
        mergeVolumes(imageConfig?.Volumes, svc.volumes);

    // Config
    const config: ContainerConfig = {
        Hostname: svc.hostname || containerId.slice(0, 12),
        Domainname: svc.domainname || "",
        User: svc.user || imageConfig?.User || "",
        AttachStdin: false,
        AttachStdout: true,
        AttachStderr: true,
        ExposedPorts: Object.keys(exposedPorts).length > 0 ? exposedPorts : undefined,
        Tty: svc.tty || false,
        OpenStdin: svc.stdinOpen || false,
        StdinOnce: false,
        Env: mergedEnv.length > 0 ? mergedEnv : undefined,
        Cmd: cmd.length > 0 ? cmd : undefined,
        Image: override.needsRecreation ? imageRef + "-old" : imageRef,
        WorkingDir: svc.workingDir || imageConfig?.WorkingDir || "",
        Entrypoint: entrypoint.length > 0 ? entrypoint : undefined,
        Labels: labels,
        StopSignal: svc.stopSignal || imageConfig?.StopSignal || "SIGTERM",
        StopTimeout: svc.stopGracePeriod,
        Volumes: mergedVolumes,
        Shell: imageConfig?.Shell,
    };

    if (svc.healthcheck && !svc.healthcheck.disable) {
        config.Healthcheck = {
            Test: svc.healthcheck.test,
            Interval: svc.healthcheck.interval,
            Timeout: svc.healthcheck.timeout,
            Retries: svc.healthcheck.retries,
            StartPeriod: svc.healthcheck.startPeriod,
            StartInterval: svc.healthcheck.startInterval,
        };
    }

    // HostConfig
    const hostConfig = buildHostConfig(project, svc, networkMap, containerIds, input.parsed.networks);

    // Mounts
    const mounts = buildMounts(svc, project, stackDir, input.parsed.volumes);

    // NetworkSettings
    const networkSettings = buildNetworkSettings(project, serviceName, svc, networkMap, containerIds, input.parsed.networks);

    // Image ID — use real ID from resolved image when available
    const imageId = resolvedImage?.Id ?? ("sha256:" + deterministicId(imageSeed(imageRef), "image-id"));

    const container: ContainerInspect = {
        Id: containerId,
        Created: deterministicTimestamp(seed + "created", baseTime),
        Path: path,
        Args: args,
        State: state,
        Image: imageId,
        ResolvConfPath: `/var/lib/docker/containers/${containerId}/resolv.conf`,
        HostnamePath: `/var/lib/docker/containers/${containerId}/hostname`,
        HostsPath: `/var/lib/docker/containers/${containerId}/hosts`,
        LogPath: `/var/lib/docker/containers/${containerId}/${containerId}-json.log`,
        Name: nameWithSlash,
        RestartCount: 0,
        Driver: "overlay2",
        Platform: "linux",
        MountLabel: "",
        ProcessLabel: "",
        AppArmorProfile: "",
        ExecIDs: null,
        HostConfig: hostConfig,
        Mounts: mounts,
        Config: config,
        NetworkSettings: networkSettings,
    };

    return container;
}

function resolveCommand(cmd: string | string[] | undefined): string[] {
    if (!cmd) return [];
    if (Array.isArray(cmd)) return cmd;
    // Split string command by spaces (simple split, good enough for mock)
    return cmd.split(/\s+/).filter(Boolean);
}

function buildContainerState(
    status: string,
    exitCode: number,
    seed: string,
    baseTime: string,
    svc: ParsedService,
    override: MockServiceOverride,
): ContainerState {
    const running = status === "running";
    const paused = status === "paused";
    const startedAt = deterministicTimestamp(seed + "started", baseTime);
    const finishedAt = running || paused
        ? "0001-01-01T00:00:00Z"
        : deterministicTimestamp(seed + "finished", baseTime);

    const state: ContainerState = {
        Status: status,
        Running: running,
        Paused: paused,
        Restarting: false,
        OOMKilled: false,
        Dead: false,
        Pid: running || paused ? deterministicInt(seed + "pid", 1000, 65535) : 0,
        ExitCode: exitCode,
        Error: "",
        StartedAt: startedAt,
        FinishedAt: finishedAt,
    };

    // Health
    const healthStatus = resolveHealth(svc, override);
    if (healthStatus) {
        state.Health = buildHealthState(healthStatus, seed, baseTime);
    }

    return state;
}

function resolveHealth(svc: ParsedService, override: MockServiceOverride): string | undefined {
    // .mock.yaml override takes priority
    if (override.health) {
        return override.health === "none" ? undefined : override.health;
    }
    // If healthcheck defined and not disabled, default to "healthy"
    if (svc.healthcheck && !svc.healthcheck.disable) {
        return "healthy";
    }
    return undefined;
}

function buildHealthState(status: string, seed: string, baseTime: string): HealthState {
    return {
        Status: status,
        FailingStreak: status === "unhealthy" ? 3 : 0,
        Log: [
            {
                Start: deterministicTimestamp(seed + "health-start", baseTime),
                End: deterministicTimestamp(seed + "health-end", baseTime),
                ExitCode: status === "healthy" ? 0 : 1,
                Output: "",
            },
        ],
    };
}

function buildContainerLabels(
    project: string,
    serviceName: string,
    composeFilePath: string,
    stackDir: string,
    imageRef: string,
    svc: ParsedService,
): Record<string, string> {
    const labels: Record<string, string> = {
        "com.docker.compose.config-hash": deterministicId(serviceSeed(project, serviceName), "config-hash"),
        "com.docker.compose.container-number": "1",
        "com.docker.compose.image": imageRef,
        "com.docker.compose.oneoff": "False",
        "com.docker.compose.project": project,
        "com.docker.compose.project.config_files": composeFilePath,
        "com.docker.compose.project.working_dir": stackDir,
        "com.docker.compose.service": serviceName,
        "com.docker.compose.version": COMPOSE_VERSION,
        ...svc.labels,
    };
    return labels;
}

// --- HostConfig ---

function buildHostConfig(
    project: string,
    svc: ParsedService,
    networkMap: Map<string, NetworkInspect>,
    containerIds: Map<string, string>,
    parsedNetworks: Record<string, ParsedNetwork>,
): HostConfig {
    const hc: HostConfig = {
        NetworkMode: resolveNetworkMode(project, svc, containerIds, parsedNetworks),
        RestartPolicy: parseRestartPolicy(svc.restart),
        AutoRemove: false,
        PublishAllPorts: false,
        ReadonlyRootfs: svc.readOnly || false,
        Privileged: svc.privileged || false,
        ConsoleSize: [0, 0],
        Isolation: "",
    };

    // Port bindings
    const portBindings: Record<string, PortBinding[]> = {};
    for (const p of svc.ports) {
        if (p.published !== undefined) {
            const key = `${p.target}/${p.protocol}`;
            if (!portBindings[key]) portBindings[key] = [];
            portBindings[key].push({
                HostIp: p.hostIp || "",
                HostPort: String(p.published),
            });
        }
    }
    if (Object.keys(portBindings).length > 0) {
        hc.PortBindings = portBindings;
    }

    // Binds — real Docker puts both bind mounts and named volumes here
    const binds: string[] = [];
    for (const vol of svc.volumes) {
        if (vol.type === "bind") {
            const mode = vol.readOnly ? "ro" : "rw";
            binds.push(`${vol.source}:${vol.target}:${mode}`);
        } else if (vol.type === "volume" && vol.source) {
            const mode = vol.readOnly ? "ro" : "rw";
            binds.push(`${vol.source}:${vol.target}:${mode}`);
        }
    }
    if (binds.length > 0) hc.Binds = binds;

    // Capabilities
    if (svc.capAdd) hc.CapAdd = svc.capAdd;
    if (svc.capDrop) hc.CapDrop = svc.capDrop;

    // DNS
    if (svc.dns) hc.Dns = svc.dns;
    if (svc.dnsSearch) hc.DnsSearch = svc.dnsSearch;
    if (svc.dnsOpt) hc.DnsOptions = svc.dnsOpt;

    // Extra hosts
    if (svc.extraHosts) hc.ExtraHosts = svc.extraHosts;

    // Security
    if (svc.securityOpt) hc.SecurityOpt = svc.securityOpt;

    // PID/IPC mode
    if (svc.pid) hc.PidMode = svc.pid;
    if (svc.ipc) hc.IpcMode = svc.ipc;

    // Init
    if (svc.init !== undefined) hc.Init = svc.init;

    // Runtime
    if (svc.runtime) hc.Runtime = svc.runtime;

    // Resources
    if (svc.memLimit) hc.Memory = svc.memLimit;
    if (svc.memReservation) hc.MemoryReservation = svc.memReservation;
    if (svc.cpus) hc.NanoCpus = Math.floor(svc.cpus * 1_000_000_000);
    if (svc.cpuShares) hc.CpuShares = svc.cpuShares;
    if (svc.pidsLimit) hc.PidsLimit = svc.pidsLimit;
    if (svc.shmSize) hc.ShmSize = svc.shmSize;

    // Ulimits
    if (svc.ulimits) {
        hc.Ulimits = svc.ulimits.map((u) => ({
            Name: u.name,
            Soft: u.soft,
            Hard: u.hard,
        }));
    }

    // Devices
    if (svc.devices) {
        hc.Devices = svc.devices.map((d) => ({
            PathOnHost: d.host,
            PathInContainer: d.container,
            CgroupPermissions: d.permissions,
        }));
    }

    // Sysctls
    if (svc.sysctls) hc.Sysctls = svc.sysctls;

    // Tmpfs
    if (svc.tmpfs) {
        if (Array.isArray(svc.tmpfs)) {
            const tmpfs: Record<string, string> = {};
            for (const t of svc.tmpfs) {
                tmpfs[t] = "";
            }
            hc.Tmpfs = tmpfs;
        } else {
            hc.Tmpfs = svc.tmpfs;
        }
    }

    // LogConfig
    if (svc.logging) {
        hc.LogConfig = {
            Type: svc.logging.driver,
            Config: svc.logging.options,
        };
    }

    return hc;
}

function resolveNetworkMode(
    project: string,
    svc: ParsedService,
    containerIds: Map<string, string>,
    parsedNetworks: Record<string, ParsedNetwork>,
): string {
    if (svc.networkMode) {
        if (svc.networkMode.startsWith("service:")) {
            const targetService = svc.networkMode.slice("service:".length);
            const targetId = containerIds.get(targetService);
            return targetId ? `container:${targetId}` : svc.networkMode;
        }
        return svc.networkMode;
    }

    // Use first explicit network, or default
    if (svc.networks.length > 0) {
        return resolveNetworkName(project, svc.networks[0].name, parsedNetworks);
    }

    return `${project}_default`;
}

function parseRestartPolicy(restart: string): RestartPolicy {
    if (restart === "always") return { Name: "always", MaximumRetryCount: 0 };
    if (restart === "unless-stopped") return { Name: "unless-stopped", MaximumRetryCount: 0 };
    if (restart.startsWith("on-failure")) {
        const match = restart.match(/on-failure:(\d+)/);
        return { Name: "on-failure", MaximumRetryCount: match ? parseInt(match[1], 10) : 0 };
    }
    return { Name: "", MaximumRetryCount: 0 };
}

// --- Mounts ---

function buildMounts(svc: ParsedService, project: string, stackDir: string, parsedVolumes: Record<string, ParsedVolume>): MountPoint[] {
    return svc.volumes.map((vol): MountPoint => {
        if (vol.type === "bind") {
            // Resolve relative bind paths against stack dir
            const source = vol.source.startsWith("/") ? vol.source : `${stackDir}/${vol.source}`;
            return {
                Type: "bind",
                Source: source,
                Destination: vol.target,
                Mode: vol.readOnly ? "ro" : "",
                RW: !vol.readOnly,
                Propagation: vol.bindPropagation || "rprivate",
            };
        }

        // Named or anonymous volume — resolve through the same helper used for volume generation
        let volumeName: string;
        if (vol.source && parsedVolumes[vol.source]) {
            volumeName = resolveVolumeName(project, vol.source, parsedVolumes[vol.source]);
        } else {
            volumeName = vol.source || `${deterministicId(serviceSeed(project, vol.target), "anon-vol").slice(0, 64)}`;
        }
        return {
            Type: "volume",
            Name: volumeName,
            Source: `/var/lib/docker/volumes/${volumeName}/_data`,
            Destination: vol.target,
            Driver: "local",
            Mode: vol.readOnly ? "ro" : "z",
            RW: !vol.readOnly,
        };
    });
}

// --- NetworkSettings ---

function buildNetworkSettings(
    project: string,
    serviceName: string,
    svc: ParsedService,
    networkMap: Map<string, NetworkInspect>,
    containerIds: Map<string, string>,
    parsedNetworks: Record<string, ParsedNetwork>,
): NetworkSettings {
    const settings: NetworkSettings = {
        Bridge: "",
        SandboxID: deterministicId(serviceSeed(project, serviceName), "sandbox-id"),
        HairpinMode: false,
        LinkLocalIPv6Address: "",
        LinkLocalIPv6PrefixLen: 0,
        Ports: {},
        SandboxKey: `/var/run/docker/netns/${deterministicId(serviceSeed(project, serviceName), "netns").slice(0, 12)}`,
        SecondaryIPAddresses: null as unknown as undefined,
        SecondaryIPv6Addresses: null as unknown as undefined,
        Networks: {},
    };

    // Build ports map
    for (const p of svc.ports) {
        const key = `${p.target}/${p.protocol}`;
        if (p.published !== undefined) {
            if (!settings.Ports![key]) settings.Ports![key] = [];
            (settings.Ports![key] as PortBinding[]).push({
                HostIp: p.hostIp || "0.0.0.0",
                HostPort: String(p.published),
            });
        } else {
            settings.Ports![key] = null;
        }
    }

    // Handle network mode
    if (svc.networkMode === "host") {
        settings.Networks = { host: buildMinimalEndpoint("host", serviceName) };
        return settings;
    }
    if (svc.networkMode === "none") {
        settings.Networks = {};
        return settings;
    }
    if (svc.networkMode?.startsWith("service:") || svc.networkMode?.startsWith("container:")) {
        settings.Networks = {};
        return settings;
    }

    // Regular networks
    const serviceNetworks = svc.networks.length > 0
        ? svc.networks
        : [{ name: "default" }]; // implicit default

    for (const svcNet of serviceNetworks) {
        const resolvedName = resolveNetworkName(project, svcNet.name, parsedNetworks);
        const network = networkMap.get(resolvedName);
        const networkId = network?.Id || deterministicId(networkSeed(resolvedName), "network-id");

        const subnet = network?.IPAM?.Config?.[0]?.Subnet || "172.18.0.0/16";
        const gateway = network?.IPAM?.Config?.[0]?.Gateway || deterministicIp(networkSeed(resolvedName) + "gateway", subnet);

        const epSeed = serviceSeed(project, serviceName) + resolvedName;
        const endpoint: EndpointSettings = {
            NetworkID: networkId,
            EndpointID: deterministicId(epSeed, "endpoint-id"),
            Gateway: gateway,
            IPAddress: svcNet.ipv4Address || deterministicIp(epSeed, subnet),
            IPPrefixLen: parseInt(subnet.split("/")[1] || "16", 10),
            MacAddress: deterministicMac(epSeed),
            DNSNames: [serviceName, `${project}-${serviceName}-1`, resolvedName],
        };

        const cid = containerIds.get(serviceName) || "";
        const defaultAliases = [serviceName, `${project}-${serviceName}-1`, cid.slice(0, 12)];
        endpoint.Aliases = svcNet.aliases
            ? [...svcNet.aliases, ...defaultAliases]
            : defaultAliases;
        if (svcNet.ipv4Address) {
            endpoint.IPAMConfig = { IPv4Address: svcNet.ipv4Address };
        }

        settings.Networks![resolvedName] = endpoint;
    }

    // Set primary network info
    const firstNetName = Object.keys(settings.Networks!)[0];
    if (firstNetName) {
        const first = settings.Networks![firstNetName];
        settings.Gateway = first.Gateway;
        settings.IPAddress = first.IPAddress;
        settings.IPPrefixLen = first.IPPrefixLen;
        settings.MacAddress = first.MacAddress;
    }

    return settings;
}

function buildMinimalEndpoint(networkName: string, serviceName: string): EndpointSettings {
    return {
        NetworkID: "",
        EndpointID: "",
        Gateway: "",
        IPAddress: "",
        IPPrefixLen: 0,
        MacAddress: "",
    };
}

// --- Network generation ---

/**
 * Resolve the Docker network name for a compose network key.
 * If parsedNetworks is provided and the key maps to an external network,
 * the external name is returned instead of prefixing with project_.
 */
export function resolveNetworkName(
    project: string,
    key: string,
    parsedNetworks?: Record<string, ParsedNetwork>,
): string {
    const config = parsedNetworks?.[key];
    if (config) {
        return resolveNetworkNameFromConfig(project, key, config);
    }
    // No config available (e.g. implicit default) — standard prefix
    return `${project}_${key}`;
}

export function resolveNetworkNameFromConfig(
    project: string,
    key: string,
    config: { name?: string; external: boolean },
): string {
    if (config.external) return config.name || key;
    return config.name || `${project}_${key}`;
}

function resolveVolumeName(project: string, key: string, config: ParsedVolume): string {
    if (config.external) return config.name || key;
    return config.name || `${project}_${key}`;
}

function generateNetworks(input: GeneratorInput, baseTime: string): NetworkInspect[] {
    const { project, parsed } = input;
    const networks: NetworkInspect[] = [];

    // Check if implicit default network is needed
    let needsDefault = false;
    for (const svc of Object.values(parsed.services)) {
        if (!svc.networkMode && svc.networks.length === 0) {
            needsDefault = true;
            break;
        }
    }

    // Explicit networks from compose
    const definedKeys = new Set(Object.keys(parsed.networks));

    if (needsDefault && !definedKeys.has("default")) {
        // Add implicit default
        networks.push(generateSingleNetwork(project, "default", {
            driver: "bridge",
            internal: false,
            attachable: false,
            external: false,
            labels: {},
            enableIpv4: true,
            enableIpv6: false,
        }, baseTime));
    }

    for (const [key, config] of Object.entries(parsed.networks)) {
        networks.push(generateSingleNetwork(project, key, config, baseTime));
    }

    return networks;
}

function generateSingleNetwork(
    project: string,
    key: string,
    config: { name?: string; driver?: string; internal?: boolean; attachable?: boolean; external?: boolean; labels?: Record<string, string>; enableIpv4?: boolean; enableIpv6?: boolean; ipam?: { driver?: string; config?: Array<{ subnet?: string; ipRange?: string; gateway?: string }> }; driverOpts?: Record<string, string> },
    baseTime: string,
): NetworkInspect {
    const resolvedName = resolveNetworkNameFromConfig(project, key, { name: config.name, external: !!config.external });
    const seed = networkSeed(resolvedName);
    const id = deterministicId(seed, "network-id");
    const subnet = config.ipam?.config?.[0]?.subnet || `172.${deterministicInt(seed + "subnet-b", 18, 31)}.0.0/16`;
    const gateway = config.ipam?.config?.[0]?.gateway || subnet.replace(/\.0\.0\/\d+$/, ".0.1");

    const net: NetworkInspect = {
        Name: resolvedName,
        Id: id,
        Created: deterministicTimestamp(seed + "created", baseTime),
        Scope: "local",
        Driver: config.driver || "bridge",
        EnableIPv4: config.enableIpv4 !== false,
        EnableIPv6: config.enableIpv6 || false,
        IPAM: {
            Driver: config.ipam?.driver || "default",
            Config: [{
                Subnet: subnet,
                Gateway: gateway,
            }],
            Options: {},
        },
        Internal: config.internal || false,
        Attachable: config.attachable || false,
        Ingress: false,
        ConfigFrom: { Network: "" },
        ConfigOnly: false,
        Containers: {},
        Options: config.driverOpts || {},
        Labels: {
            "com.docker.compose.network": key,
            "com.docker.compose.project": project,
            ...(config.labels || {}),
        },
    };

    return net;
}

// --- Volume generation ---

function generateVolumes(input: GeneratorInput, baseTime: string): VolumeInspect[] {
    const { project, parsed } = input;
    const volumes: VolumeInspect[] = [];

    for (const [key, config] of Object.entries(parsed.volumes)) {
        const resolvedName = resolveVolumeName(project, key, config);
        const vol: VolumeInspect = {
            Name: resolvedName,
            Driver: config.driver,
            Mountpoint: `/var/lib/docker/volumes/${resolvedName}/_data`,
            CreatedAt: deterministicTimestamp(networkSeed(resolvedName) + "vol-created", baseTime),
            Labels: {
                "com.docker.compose.project": project,
                "com.docker.compose.volume": key,
                ...(config.labels || {}),
            },
            Scope: "local",
            Options: config.driverOpts || undefined,
        };
        volumes.push(vol);
    }

    return volumes;
}

// --- Precaptured image lookup ---

function lookupPrecaptured(
    ref: string,
    precaptured?: Map<string, ImageInspect>,
): ImageInspect | undefined {
    if (!precaptured) return undefined;
    // Try exact match
    if (precaptured.has(ref)) return precaptured.get(ref);
    // Try without :latest suffix
    if (ref.endsWith(":latest")) {
        const bare = ref.slice(0, -":latest".length);
        if (precaptured.has(bare)) return precaptured.get(bare);
    }
    // Try adding :latest
    if (!ref.includes(":")) {
        if (precaptured.has(ref + ":latest")) return precaptured.get(ref + ":latest");
    }
    return undefined;
}

// --- Layer 2 merge helpers ---

function mergeEnvironment(
    imageEnv: string[] | undefined,
    composeEnv: Record<string, string>,
): string[] {
    // Start with image env, overlay compose env (compose wins by key)
    const envMap = new Map<string, string>();

    // Image defaults first
    if (imageEnv) {
        for (const entry of imageEnv) {
            const eqIdx = entry.indexOf("=");
            if (eqIdx !== -1) {
                envMap.set(entry.slice(0, eqIdx), entry.slice(eqIdx + 1));
            }
        }
    }

    // Compose overrides
    for (const [k, v] of Object.entries(composeEnv)) {
        envMap.set(k, v);
    }

    return [...envMap.entries()].map(([k, v]) => `${k}=${v}`);
}

function mergeVolumes(
    imageVolumes: Record<string, Record<string, never>> | undefined,
    composeVolumes: ParsedVolumeMount[],
): Record<string, Record<string, never>> | undefined {
    const merged: Record<string, Record<string, never>> = {};
    let hasEntries = false;

    if (imageVolumes) {
        for (const key of Object.keys(imageVolumes)) {
            merged[key] = {};
            hasEntries = true;
        }
    }

    for (const vol of composeVolumes) {
        merged[vol.target] = {};
        hasEntries = true;
    }

    return hasEntries ? merged : undefined;
}

// --- Image generation ---

function generateImage(ref: string, baseTime: string): ImageInspect {
    const seed = imageSeed(ref);
    const id = deterministicId(seed, "image-id");

    return {
        Id: `sha256:${id}`,
        RepoTags: [ref],
        RepoDigests: [`${ref.split(":")[0]}@sha256:${deterministicId(seed, "image-digest")}`],
        Created: deterministicTimestamp(seed + "created", baseTime),
        Architecture: "amd64",
        Os: "linux",
        Size: deterministicInt(seed + "size", 10_000_000, 500_000_000),
        RootFS: {
            Type: "layers",
            Layers: [
                `sha256:${deterministicId(seed, "layer-0")}`,
            ],
        },
        Config: {
            Cmd: ["/bin/sh"],
            Env: ["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],
            Labels: {},
        },
    };
}
