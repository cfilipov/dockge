import { readFileSync, readdirSync, statSync, cpSync, rmSync, existsSync } from "node:fs";
import { join, resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { MockState } from "./state.js";
import { parseCompose, findComposeFile } from "./compose-parser.js";
import { parseStackMockConfig, parseGlobalMockConfig } from "./mock-config.js";
import type { MockGlobalConfig } from "./mock-config.js";
import { generateStack } from "./generator.js";
import type { Clock } from "./clock.js";
import type { NetworkInspect, VolumeInspect, ImageInspect } from "./types.js";
import {
    deterministicId,
    deterministicTimestamp,
    deterministicInt,
    networkSeed,
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

    // Step 1: Copy stacks source to runtime dir
    if (stacksSource !== stacksDir) {
        cpSync(stacksSource, stacksDir, { recursive: true });
    }

    // Step 2: Read global .mock.yaml
    const globalMockPath = join(stacksDir, ".mock.yaml");
    const globalMockContent = readFileSafe(globalMockPath);
    const globalConfig = parseGlobalMockConfig(globalMockContent);

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

    // Step 2b: Load pre-captured images
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
