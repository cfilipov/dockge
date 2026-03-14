import { readFileSync, readdirSync, statSync, cpSync, rmSync, existsSync } from "node:fs";
import { join, resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { MockState } from "./state.js";
import { parseCompose, findComposeFile } from "./compose-parser.js";
import { parseStackMockConfig, parseGlobalMockConfig } from "./mock-config.js";
import type { MockGlobalConfig, MockStandaloneContainer } from "./mock-config.js";
import { generateStack } from "./generator.js";
import { loadLogTemplates } from "./log-templates.js";
import type { Clock } from "./clock.js";
import type {
    ContainerInspect, ContainerState, NetworkInspect, VolumeInspect, ImageInspect,
    EndpointSettings, PortBinding,
} from "./types.js";
import {
    deterministicId,
    deterministicMac,
    deterministicIp,
    deterministicTimestamp,
    deterministicInt,
    hashToSeed,
    networkSeed,
    imageSeed,
} from "./deterministic.js";
import { parseEnvironment } from "./compose-parser.js";

export interface InitOptions {
    stacksSource: string;
    stacksDir: string;
    clock: Clock;
    imagesJsonPath?: string;
}

export async function initState(opts: InitOptions): Promise<MockState> {
    const { stacksSource, stacksDir, clock } = opts;
    const state = new MockState();
    const baseTime = clock.now().toISOString();

    // Load log templates from stacks source (before copying, so we read from pristine source)
    state.logTemplates = loadLogTemplates(stacksSource);

    // Step 1: Copy stacks source to runtime dir
    if (stacksSource !== stacksDir) {
        cpSync(stacksSource, stacksDir, { recursive: true });
    }

    // Step 2: Read global .mock.yaml
    const globalMockPath = join(stacksDir, ".mock.yaml");
    const globalMockContent = readFileSafe(globalMockPath);
    const globalConfig = parseGlobalMockConfig(globalMockContent);

    // Create default Docker system networks (bridge, host, none)
    for (const sysDef of DEFAULT_SYSTEM_NETWORKS) {
        const net = createSystemNetwork(sysDef, baseTime);
        state.networks.set(net.Id, net);
    }

    // Create global networks
    for (const [name, netDef] of Object.entries(globalConfig.networks)) {
        const net = createGlobalNetwork(name, netDef, baseTime);
        state.networks.set(net.Id, net);
    }

    // Create global volumes
    for (const [name, volDef] of Object.entries(globalConfig.volumes)) {
        const vol = createGlobalVolume(name, volDef, baseTime);
        state.volumes.set(vol.Name, vol);
    }

    // Create dangling images (no tags)
    for (let i = 0; i < globalConfig.danglingImages.length; i++) {
        const dImg = globalConfig.danglingImages[i];
        const seed = hashToSeed(["dangling-image", String(i)]);
        const id = `sha256:${deterministicId(seed, "image-id")}`;
        state.images.set(id, {
            Id: id,
            RepoTags: [],
            RepoDigests: [],
            Created: dImg.created || deterministicTimestamp(seed + "created", baseTime),
            Architecture: "amd64",
            Os: "linux",
            Size: dImg.size,
            RootFS: { Type: "layers", Layers: [`sha256:${deterministicId(seed, "layer-0")}`] },
            Config: {
                Cmd: ["/bin/sh"],
                Env: ["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],
                Labels: {},
            },
        });
    }

    // Step 2b: Create standalone containers (not part of any compose stack)
    for (const cDef of globalConfig.containers) {
        const container = createStandaloneContainer(cDef, state.networks, baseTime);
        state.containers.set(container.Id, container);
        // Create image if not already present
        const imageRef = normalizeRef(cDef.image);
        const imgSeed = imageSeed(imageRef);
        const imgId = `sha256:${deterministicId(imgSeed, "image-id")}`;
        if (!state.images.has(imgId)) {
            state.images.set(imgId, generateSyntheticImage(imageRef, baseTime));
        }
    }

    // Step 2c: Load pre-captured images
    const precapturedImages = loadPrecapturedImages(opts);

    // Step 3: Scan stacks dir for subdirectories
    const untrackedDirs: string[] = [];
    let entries: string[];
    try {
        entries = readdirSync(stacksDir);
    } catch {
        return state;
    }

    for (const entry of entries) {
        const subdir = join(stacksDir, entry);
        try {
            if (!statSync(subdir).isDirectory()) continue;
        } catch {
            continue;
        }

        try {
            const composeFilePath = findComposeFile(subdir);
            if (!composeFilePath) continue;

            const composeContent = readFileSync(composeFilePath, "utf-8");
            const mockContent = readFileSafe(join(subdir, ".mock.yaml"));

            const parsed = parseCompose(composeContent);
            const mockConfig = parseStackMockConfig(mockContent);

            if (!mockConfig.deployed) continue;

            // Read env_file contents and merge into service environments
            for (const svc of Object.values(parsed.services)) {
                for (const envFilePath of svc.envFile) {
                    const absPath = envFilePath.startsWith("/") ? envFilePath : join(subdir, envFilePath);
                    const envContent = readFileSafe(absPath);
                    if (envContent) {
                        const envVars = parseEnvFileContent(envContent);
                        // env_file values go before compose environment (compose wins)
                        svc.environment = { ...envVars, ...svc.environment };
                    }
                }
            }

            const generated = generateStack({
                project: entry,
                stackDir: resolve(subdir),
                composeFilePath: resolve(composeFilePath),
                parsed,
                mockConfig,
                clock,
                existingNetworks: state.networks,
                precapturedImages,
            });

            // Merge into state
            for (const c of generated.containers) {
                state.containers.set(c.Id, c);
            }
            for (const n of generated.networks) {
                // Don't overwrite existing (e.g. global) networks
                if (!state.networks.has(n.Id)) {
                    state.networks.set(n.Id, n);
                }
            }
            for (const v of generated.volumes) {
                state.volumes.set(v.Name, v);
            }
            for (const img of generated.images) {
                state.images.set(img.Id, img);
            }

            if (mockConfig.untracked) {
                untrackedDirs.push(subdir);
            }
        } catch (err) {
            // Log warning but continue with other stacks
            console.warn(`[mock-docker] Warning: failed to process stack "${entry}":`, err);
        }
    }

    // Step 4: Post-process — populate NetworkInspect.Containers
    for (const container of state.containers.values()) {
        const networks = container.NetworkSettings.Networks;
        if (!networks) continue;

        for (const [netName, endpoint] of Object.entries(networks)) {
            // Find the network by ID
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

    // Step 5: Delete untracked dirs
    for (const dir of untrackedDirs) {
        try {
            rmSync(dir, { recursive: true, force: true });
        } catch {
            // ignore
        }
    }

    return state;
}

function readFileSafe(path: string): string | null {
    try {
        return readFileSync(path, "utf-8");
    } catch {
        return null;
    }
}

function parseEnvFileContent(content: string): Record<string, string> {
    const env: Record<string, string> = {};
    for (const line of content.split("\n")) {
        const trimmed = line.trim();
        if (!trimmed || trimmed.startsWith("#")) continue;
        const eqIdx = trimmed.indexOf("=");
        if (eqIdx === -1) continue;
        const key = trimmed.slice(0, eqIdx).trim();
        let value = trimmed.slice(eqIdx + 1).trim();
        // Strip surrounding quotes
        if ((value.startsWith('"') && value.endsWith('"')) ||
            (value.startsWith("'") && value.endsWith("'"))) {
            value = value.slice(1, -1);
        }
        env[key] = value;
    }
    return env;
}

function createGlobalNetwork(
    name: string,
    def: { driver: string; subnet?: string; gateway?: string; internal?: boolean },
    baseTime: string,
): NetworkInspect {
    const seed = networkSeed(name);
    const id = deterministicId(seed, "network-id");
    const subnet = def.subnet || `172.${deterministicInt(seed + "subnet-b", 17, 31)}.0.0/16`;
    const gateway = def.gateway || subnet.replace(/\.0\.0\/\d+$/, ".0.1");

    return {
        Name: name,
        Id: id,
        Created: deterministicTimestamp(seed + "created", baseTime),
        Scope: "local",
        Driver: def.driver,
        EnableIPv4: true,
        EnableIPv6: false,
        IPAM: {
            Driver: "default",
            Config: [{ Subnet: subnet, Gateway: gateway }],
            Options: {},
        },
        Internal: def.internal || false,
        Attachable: false,
        Ingress: false,
        ConfigFrom: { Network: "" },
        ConfigOnly: false,
        Containers: {},
        Options: {},
        Labels: {},
    };
}

function loadPrecapturedImages(opts: InitOptions): Map<string, ImageInspect> | undefined {
    // Determine path: explicit, or auto-discover from stacks source, or package root
    const candidates: string[] = [];
    if (opts.imagesJsonPath) {
        candidates.push(opts.imagesJsonPath);
    } else {
        candidates.push(join(opts.stacksSource, "images.json"));
        // Try mock-docker package root (two levels up from this file: src/init.ts → mock-docker/)
        try {
            const packageRoot = join(dirname(fileURLToPath(import.meta.url)), "..");
            candidates.push(join(packageRoot, "scripts", "images.json"));
        } catch {
            // import.meta.url might not resolve in all contexts
        }
    }

    for (const path of candidates) {
        if (!existsSync(path)) continue;
        try {
            const raw = JSON.parse(readFileSync(path, "utf-8")) as Record<string, ImageInspect>;
            const map = new Map<string, ImageInspect>();
            for (const [key, value] of Object.entries(raw)) {
                map.set(key, value);
            }
            return map.size > 0 ? map : undefined;
        } catch {
            // Silently skip unparseable files
        }
    }

    return undefined;
}

function createGlobalVolume(
    name: string,
    def: { driver: string },
    baseTime: string,
): VolumeInspect {
    const seed = networkSeed(name); // reuse seed function — just needs uniqueness
    return {
        Name: name,
        Driver: def.driver,
        Mountpoint: `/var/lib/docker/volumes/${name}/_data`,
        CreatedAt: deterministicTimestamp(seed + "vol-created", baseTime),
        Labels: {},
        Scope: "local",
    };
}

// ---------------------------------------------------------------------------
// Standalone containers (no compose labels)
// ---------------------------------------------------------------------------

function normalizeRef(ref: string): string {
    return ref.includes(":") ? ref : ref + ":latest";
}

function standaloneSeed(name: string): string {
    return hashToSeed(["standalone", name]);
}

function createStandaloneContainer(
    def: MockStandaloneContainer,
    existingNetworks: Map<string, NetworkInspect>,
    baseTime: string,
): ContainerInspect {
    const s = standaloneSeed(def.name);
    const containerId = deterministicId(s, "container-id");
    const imageRef = normalizeRef(def.image);
    const imgId = `sha256:${deterministicId(imageSeed(imageRef), "image-id")}`;

    // State
    const stateStr = def.state || "running";
    const running = stateStr === "running";
    const paused = stateStr === "paused";
    const exitCode = def.exitCode ?? 0;
    const startedAt = deterministicTimestamp(s + "started", baseTime);
    const finishedAt = running || paused ? "0001-01-01T00:00:00Z" : deterministicTimestamp(s + "finished", baseTime);

    const state: ContainerState = {
        Status: stateStr,
        Running: running,
        Paused: paused,
        Restarting: false,
        OOMKilled: false,
        Dead: false,
        Pid: running || paused ? deterministicInt(s + "pid", 1000, 65535) : 0,
        ExitCode: exitCode,
        Error: "",
        StartedAt: startedAt,
        FinishedAt: finishedAt,
    };

    // Command
    const cmdParts = def.command ? def.command.split(/\s+/).filter(Boolean) : [];
    const path = cmdParts.length > 0 ? cmdParts[0] : "";
    const args = cmdParts.slice(1);

    // Ports
    const ports: Record<string, PortBinding[] | null> = {};
    const portBindings: Record<string, PortBinding[]> = {};
    const exposedPorts: Record<string, Record<string, never>> = {};

    for (const portStr of def.ports || []) {
        const parsed = parsePortSpec(portStr);
        const key = `${parsed.containerPort}/${parsed.protocol}`;
        exposedPorts[key] = {};
        if (parsed.hostPort !== undefined) {
            const binding: PortBinding = { HostIp: parsed.hostIp || "0.0.0.0", HostPort: String(parsed.hostPort) };
            if (!ports[key]) ports[key] = [];
            (ports[key] as PortBinding[]).push(binding);
            if (!portBindings[key]) portBindings[key] = [];
            portBindings[key].push({ HostIp: parsed.hostIp || "", HostPort: String(parsed.hostPort) });
        } else {
            ports[key] = null;
        }
    }

    // Volumes / mounts
    const mounts = (def.volumes || []).map((v) => {
        const [src, dst] = v.split(":");
        return {
            Type: "volume" as const,
            Name: src,
            Source: `/var/lib/docker/volumes/${src}/_data`,
            Destination: dst || src,
            Driver: "local",
            Mode: "z",
            RW: true,
        };
    });

    // Networks — attach to bridge by default
    const networkNames = def.networks && def.networks.length > 0 ? def.networks : ["bridge"];
    const networks: Record<string, EndpointSettings> = {};
    for (const netName of networkNames) {
        // Find existing network
        let network: NetworkInspect | undefined;
        for (const n of existingNetworks.values()) {
            if (n.Name === netName) { network = n; break; }
        }
        const networkId = network?.Id || deterministicId(networkSeed(netName), "network-id");
        const subnet = network?.IPAM?.Config?.[0]?.Subnet || "172.17.0.0/16";
        const gateway = network?.IPAM?.Config?.[0]?.Gateway || "172.17.0.1";
        const epSeed = s + netName;

        networks[netName] = {
            NetworkID: networkId,
            EndpointID: deterministicId(epSeed, "endpoint-id"),
            Gateway: gateway,
            IPAddress: deterministicIp(epSeed, subnet),
            IPPrefixLen: parseInt(subnet.split("/")[1] || "16", 10),
            MacAddress: deterministicMac(epSeed),
            DNSNames: [def.name],
        };
    }

    // Primary network info
    const firstNet = Object.values(networks)[0];

    const container: ContainerInspect = {
        Id: containerId,
        Created: deterministicTimestamp(s + "created", baseTime),
        Path: path,
        Args: args,
        State: state,
        Image: imgId,
        ResolvConfPath: `/var/lib/docker/containers/${containerId}/resolv.conf`,
        HostnamePath: `/var/lib/docker/containers/${containerId}/hostname`,
        HostsPath: `/var/lib/docker/containers/${containerId}/hosts`,
        LogPath: `/var/lib/docker/containers/${containerId}/${containerId}-json.log`,
        Name: `/${def.name}`,
        RestartCount: 0,
        Driver: "overlay2",
        Platform: "linux",
        MountLabel: "",
        ProcessLabel: "",
        AppArmorProfile: "",
        ExecIDs: null,
        HostConfig: {
            NetworkMode: networkNames[0],
            RestartPolicy: { Name: "", MaximumRetryCount: 0 },
            AutoRemove: false,
            PublishAllPorts: false,
            ReadonlyRootfs: false,
            Privileged: false,
            ConsoleSize: [0, 0],
            Isolation: "",
            ...(Object.keys(portBindings).length > 0 ? { PortBindings: portBindings } : {}),
        },
        Mounts: mounts,
        Config: {
            Hostname: containerId.slice(0, 12),
            Domainname: "",
            User: "",
            AttachStdin: false,
            AttachStdout: true,
            AttachStderr: true,
            ExposedPorts: Object.keys(exposedPorts).length > 0 ? exposedPorts : undefined,
            Tty: false,
            OpenStdin: false,
            StdinOnce: false,
            Env: def.environment || ["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],
            Cmd: cmdParts.length > 0 ? cmdParts : undefined,
            Image: imageRef,
            WorkingDir: "",
            Labels: def.labels || {},  // No com.docker.compose.* labels
            StopSignal: "SIGTERM",
        },
        NetworkSettings: {
            Bridge: "",
            SandboxID: deterministicId(s, "sandbox-id"),
            HairpinMode: false,
            LinkLocalIPv6Address: "",
            LinkLocalIPv6PrefixLen: 0,
            Ports: ports,
            SandboxKey: `/var/run/docker/netns/${deterministicId(s, "netns").slice(0, 12)}`,
            SecondaryIPAddresses: null as unknown as undefined,
            SecondaryIPv6Addresses: null as unknown as undefined,
            Networks: networks,
            Gateway: firstNet?.Gateway,
            IPAddress: firstNet?.IPAddress,
            IPPrefixLen: firstNet?.IPPrefixLen,
            MacAddress: firstNet?.MacAddress,
        },
    };

    return container;
}

function parsePortSpec(spec: string): { hostIp?: string; hostPort?: number; containerPort: number; protocol: string } {
    let protocol = "tcp";
    let portStr = spec;
    if (portStr.endsWith("/udp")) { protocol = "udp"; portStr = portStr.slice(0, -4); }
    else if (portStr.endsWith("/tcp")) { portStr = portStr.slice(0, -4); }

    const parts = portStr.split(":");
    if (parts.length === 3) {
        return { hostIp: parts[0], hostPort: parseInt(parts[1], 10), containerPort: parseInt(parts[2], 10), protocol };
    } else if (parts.length === 2) {
        if (parts[0].includes(".")) {
            return { hostIp: parts[0], containerPort: parseInt(parts[1], 10), protocol };
        }
        return { hostPort: parseInt(parts[0], 10), containerPort: parseInt(parts[1], 10), protocol };
    }
    return { containerPort: parseInt(parts[0], 10), protocol };
}

// ---------------------------------------------------------------------------
// Default Docker system networks (bridge, host, none)
// ---------------------------------------------------------------------------

interface SystemNetworkDef {
    name: string;
    driver: string;
    subnet?: string;
    gateway?: string;
    scope: string;
}

const DEFAULT_SYSTEM_NETWORKS: SystemNetworkDef[] = [
    { name: "bridge", driver: "bridge", subnet: "172.17.0.0/16", gateway: "172.17.0.1", scope: "local" },
    { name: "host", driver: "host", scope: "local" },
    { name: "none", driver: "null", scope: "local" },
];

function createSystemNetwork(def: SystemNetworkDef, baseTime: string): NetworkInspect {
    const seed = networkSeed(def.name);
    const id = deterministicId(seed, "network-id");

    const ipamConfig = def.subnet
        ? [{ Subnet: def.subnet, Gateway: def.gateway || "" }]
        : [];

    return {
        Name: def.name,
        Id: id,
        Created: deterministicTimestamp(seed + "created", baseTime),
        Scope: def.scope,
        Driver: def.driver,
        EnableIPv4: !!def.subnet,
        EnableIPv6: false,
        IPAM: {
            Driver: "default",
            Config: ipamConfig,
            Options: {},
        },
        Internal: false,
        Attachable: false,
        Ingress: false,
        ConfigFrom: { Network: "" },
        ConfigOnly: false,
        Containers: {},
        Options: {},
        Labels: {},
    };
}

function generateSyntheticImage(ref: string, baseTime: string): ImageInspect {
    const s = imageSeed(ref);
    const id = deterministicId(s, "image-id");
    return {
        Id: `sha256:${id}`,
        RepoTags: [ref],
        RepoDigests: [`${ref.split(":")[0]}@sha256:${deterministicId(s, "image-digest")}`],
        Created: deterministicTimestamp(s + "created", baseTime),
        Architecture: "amd64",
        Os: "linux",
        Size: deterministicInt(s + "size", 10_000_000, 500_000_000),
        RootFS: { Type: "layers", Layers: [`sha256:${deterministicId(s, "layer-0")}`] },
        Config: {
            Cmd: ["/bin/sh"],
            Env: ["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],
            Labels: {},
        },
    };
}
