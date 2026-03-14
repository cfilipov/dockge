import { parse as parseYaml } from "yaml";

// Per §3.4 — only these 5 fields
export interface MockServiceOverride {
    state?: "running" | "exited" | "paused" | "created";
    exitCode?: number;
    health?: "healthy" | "unhealthy" | "starting" | "none";
    updateAvailable?: boolean;
    needsRecreation?: boolean;
}

export interface MockStackConfig {
    deployed: boolean;       // default: true. If false, skip entirely.
    untracked: boolean;      // default: false. If true, delete runtime dir after init.
    services: Record<string, MockServiceOverride>;
}

// Per §3.3 — only networks and volumes
export interface MockGlobalNetworkDef {
    driver: string;
    subnet?: string;
    gateway?: string;
    internal?: boolean;
}

export interface MockGlobalVolumeDef {
    driver: string;
}

/** Standalone container not part of any compose stack. */
export interface MockStandaloneContainer {
    name: string;
    image: string;
    state?: "running" | "exited" | "paused" | "created";
    exitCode?: number;
    command?: string;
    ports?: string[];           // e.g. ["8080:80/tcp", "443/tcp"]
    networks?: string[];        // global network names to attach to
    volumes?: string[];         // e.g. ["pgdata:/var/lib/postgresql/data"]
    environment?: string[];     // e.g. ["KEY=value"]
    labels?: Record<string, string>;
}

export interface MockDanglingImageDef {
    size: number;
    created: string;
}

export interface MockGlobalConfig {
    networks: Record<string, MockGlobalNetworkDef>;
    volumes: Record<string, MockGlobalVolumeDef>;
    containers: MockStandaloneContainer[];
    danglingImages: MockDanglingImageDef[];
}

function parseServiceOverride(raw: Record<string, unknown>): MockServiceOverride {
    const override: MockServiceOverride = {};
    if (raw.state !== undefined) {
        override.state = raw.state as MockServiceOverride["state"];
    }
    if (raw.exit_code !== undefined) {
        override.exitCode = raw.exit_code as number;
    }
    if (raw.health !== undefined) {
        override.health = raw.health as MockServiceOverride["health"];
    }
    if (raw.update_available !== undefined) {
        override.updateAvailable = raw.update_available as boolean;
    }
    if (raw.needs_recreation !== undefined) {
        override.needsRecreation = raw.needs_recreation as boolean;
    }
    return override;
}

export function parseStackMockConfig(yamlContent: string | null): MockStackConfig {
    const defaults: MockStackConfig = {
        deployed: true,
        untracked: false,
        services: {},
    };

    if (yamlContent === null || yamlContent.trim() === "") {
        return defaults;
    }

    const raw = parseYaml(yamlContent) as Record<string, unknown> | null;
    if (!raw || typeof raw !== "object") {
        return defaults;
    }

    const config: MockStackConfig = {
        deployed: raw.deployed !== undefined ? Boolean(raw.deployed) : true,
        untracked: raw.untracked !== undefined ? Boolean(raw.untracked) : false,
        services: {},
    };

    if (raw.services && typeof raw.services === "object") {
        const services = raw.services as Record<string, Record<string, unknown>>;
        for (const [name, svcRaw] of Object.entries(services)) {
            if (svcRaw && typeof svcRaw === "object") {
                config.services[name] = parseServiceOverride(svcRaw);
            }
        }
    }

    return config;
}

export function parseGlobalMockConfig(yamlContent: string | null): MockGlobalConfig {
    const defaults: MockGlobalConfig = {
        networks: {},
        volumes: {},
        containers: [],
        danglingImages: [],
    };

    if (yamlContent === null || yamlContent.trim() === "") {
        return defaults;
    }

    const raw = parseYaml(yamlContent) as Record<string, unknown> | null;
    if (!raw || typeof raw !== "object") {
        return defaults;
    }

    const config: MockGlobalConfig = {
        networks: {},
        volumes: {},
        containers: [],
        danglingImages: [],
    };

    if (raw.networks && typeof raw.networks === "object") {
        const networks = raw.networks as Record<string, Record<string, unknown>>;
        for (const [name, netRaw] of Object.entries(networks)) {
            if (netRaw && typeof netRaw === "object") {
                config.networks[name] = {
                    driver: (netRaw.driver as string) || "bridge",
                    subnet: netRaw.subnet as string | undefined,
                    gateway: netRaw.gateway as string | undefined,
                    internal: netRaw.internal as boolean | undefined,
                };
            }
        }
    }

    if (raw.volumes && typeof raw.volumes === "object") {
        const volumes = raw.volumes as Record<string, Record<string, unknown>>;
        for (const [name, volRaw] of Object.entries(volumes)) {
            if (volRaw && typeof volRaw === "object") {
                config.volumes[name] = {
                    driver: (volRaw.driver as string) || "local",
                };
            }
        }
    }

    if (raw.containers && Array.isArray(raw.containers)) {
        for (const cRaw of raw.containers) {
            if (cRaw && typeof cRaw === "object") {
                const c = cRaw as Record<string, unknown>;
                config.containers.push({
                    name: (c.name as string) || "",
                    image: (c.image as string) || "",
                    state: c.state as MockStandaloneContainer["state"],
                    exitCode: c.exit_code as number | undefined,
                    command: c.command as string | undefined,
                    ports: c.ports as string[] | undefined,
                    networks: c.networks as string[] | undefined,
                    volumes: c.volumes as string[] | undefined,
                    environment: c.environment as string[] | undefined,
                    labels: c.labels as Record<string, string> | undefined,
                });
            }
        }
    }

    if (raw.dangling_images && Array.isArray(raw.dangling_images)) {
        for (const dRaw of raw.dangling_images) {
            if (dRaw && typeof dRaw === "object") {
                const d = dRaw as Record<string, unknown>;
                config.danglingImages.push({
                    size: (d.size as number) || 0,
                    created: (d.created as string) || "",
                });
            }
        }
    }

    return config;
}
