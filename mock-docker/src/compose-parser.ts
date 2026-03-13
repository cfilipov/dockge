import { parse as parseYaml } from "yaml";

// --- Parsed types ---

export interface ParsedPort {
    target: number;
    published?: number;
    protocol: "tcp" | "udp";
    hostIp: string;
}

export interface ParsedVolumeMount {
    type: "volume" | "bind" | "tmpfs";
    source: string;
    target: string;
    readOnly: boolean;
    bindPropagation?: string;
}

export interface ParsedHealthcheck {
    test: string[];
    interval?: number;      // nanoseconds
    timeout?: number;       // nanoseconds
    retries?: number;
    startPeriod?: number;   // nanoseconds
    startInterval?: number; // nanoseconds
    disable?: boolean;
}

export interface ParsedServiceNetwork {
    name: string;
    aliases?: string[];
    ipv4Address?: string;
    ipv6Address?: string;
}

export interface ParsedService {
    image?: string;
    build?: { context: string; dockerfile?: string };
    command?: string | string[];
    entrypoint?: string | string[];
    environment: Record<string, string>;
    envFile: string[];
    ports: ParsedPort[];
    volumes: ParsedVolumeMount[];
    networks: ParsedServiceNetwork[];
    networkMode?: string;
    restart: string;
    hostname?: string;
    domainname?: string;
    user?: string;
    workingDir?: string;
    tty?: boolean;
    stdinOpen?: boolean;
    privileged?: boolean;
    readOnly?: boolean;
    containerName?: string;
    labels: Record<string, string>;
    expose: string[];
    healthcheck?: ParsedHealthcheck;
    logging?: { driver: string; options: Record<string, string> };
    dns?: string[];
    dnsSearch?: string[];
    dnsOpt?: string[];
    extraHosts?: string[];
    capAdd?: string[];
    capDrop?: string[];
    devices?: Array<{ host: string; container: string; permissions: string }>;
    ulimits?: Array<{ name: string; soft: number; hard: number }>;
    sysctls?: Record<string, string>;
    tmpfs?: string[] | Record<string, string>;
    shmSize?: number;
    securityOpt?: string[];
    pid?: string;
    ipc?: string;
    init?: boolean;
    runtime?: string;
    stopSignal?: string;
    stopGracePeriod?: number;   // seconds
    memLimit?: number;
    memReservation?: number;
    cpus?: number;
    cpuShares?: number;
    pidsLimit?: number;
}

export interface ParsedNetwork {
    name?: string;
    driver: string;
    driverOpts?: Record<string, string>;
    internal: boolean;
    attachable: boolean;
    external: boolean;
    labels: Record<string, string>;
    ipam?: {
        driver?: string;
        config?: Array<{
            subnet?: string;
            ipRange?: string;
            gateway?: string;
        }>;
    };
    enableIpv4: boolean;
    enableIpv6: boolean;
}

export interface ParsedVolume {
    name?: string;
    driver: string;
    driverOpts?: Record<string, string>;
    external: boolean;
    labels: Record<string, string>;
}

export interface ParsedCompose {
    services: Record<string, ParsedService>;
    networks: Record<string, ParsedNetwork>;
    volumes: Record<string, ParsedVolume>;
}

// --- Duration parsing ---

/** Parse a Docker duration string to nanoseconds. Supports s, m, h, ms, us. */
export function parseDuration(value: string | number | undefined): number | undefined {
    if (value === undefined || value === null) return undefined;
    if (typeof value === "number") return value * 1_000_000_000; // assume seconds

    const str = String(value).trim();
    if (!str) return undefined;

    let totalNs = 0;
    // Match sequences like "1h", "30m", "10s", "500ms", "100us"
    const regex = /(\d+(?:\.\d+)?)\s*(h|m(?!s)|s|ms|us)/g;
    let match: RegExpExecArray | null;
    let matched = false;

    while ((match = regex.exec(str)) !== null) {
        matched = true;
        const num = parseFloat(match[1]);
        switch (match[2]) {
            case "h": totalNs += num * 3_600_000_000_000; break;
            case "m": totalNs += num * 60_000_000_000; break;
            case "s": totalNs += num * 1_000_000_000; break;
            case "ms": totalNs += num * 1_000_000; break;
            case "us": totalNs += num * 1_000; break;
        }
    }

    if (!matched) {
        // Try plain number (seconds)
        const n = parseFloat(str);
        if (!isNaN(n)) return n * 1_000_000_000;
        return undefined;
    }

    return totalNs;
}

/** Parse a Docker duration string to seconds. */
export function parseDurationSeconds(value: string | number | undefined): number | undefined {
    const ns = parseDuration(value);
    if (ns === undefined) return undefined;
    return ns / 1_000_000_000;
}

/** Parse a Docker byte size string: "256M", "1G", "512k", etc. */
export function parseByteSize(value: string | number | undefined): number | undefined {
    if (value === undefined || value === null) return undefined;
    if (typeof value === "number") return value;

    const str = String(value).trim();
    const match = str.match(/^(\d+(?:\.\d+)?)\s*([kmgtKMGT])?[bB]?$/);
    if (!match) return undefined;

    const num = parseFloat(match[1]);
    switch ((match[2] || "").toUpperCase()) {
        case "K": return Math.floor(num * 1024);
        case "M": return Math.floor(num * 1024 * 1024);
        case "G": return Math.floor(num * 1024 * 1024 * 1024);
        case "T": return Math.floor(num * 1024 * 1024 * 1024 * 1024);
        default: return Math.floor(num);
    }
}

// --- Port parsing ---

export function parsePorts(raw: unknown): ParsedPort[] {
    if (!raw || !Array.isArray(raw)) return [];

    return raw.flatMap((entry): ParsedPort[] => {
        if (typeof entry === "number") {
            return [{ target: entry, protocol: "tcp", hostIp: "" }];
        }
        if (typeof entry === "string") {
            return parsePortShort(entry);
        }
        if (typeof entry === "object" && entry !== null) {
            return [parsePortLong(entry as Record<string, unknown>)];
        }
        return [];
    });
}

function parsePortShort(str: string): ParsedPort[] {
    let protocol: "tcp" | "udp" = "tcp";
    let portStr = str;

    // Extract protocol suffix
    if (portStr.endsWith("/udp")) {
        protocol = "udp";
        portStr = portStr.slice(0, -4);
    } else if (portStr.endsWith("/tcp")) {
        portStr = portStr.slice(0, -4);
    }

    let hostIp = "";
    let published: number | undefined;
    let target: number;

    const parts = portStr.split(":");
    if (parts.length === 3) {
        // host_ip:published:target
        hostIp = parts[0];
        published = parsePortNum(parts[1]);
        target = parsePortNum(parts[2]);
    } else if (parts.length === 2) {
        // Could be host_ip:target or published:target
        // If first part contains a dot, it's an IP
        if (parts[0].includes(".")) {
            hostIp = parts[0];
            target = parsePortNum(parts[1]);
        } else {
            published = parsePortNum(parts[0]);
            target = parsePortNum(parts[1]);
        }
    } else {
        // Just target
        target = parsePortNum(parts[0]);
    }

    // Handle port ranges
    if (String(target).includes("-")) {
        const [startTarget, endTarget] = String(target).split("-").map(Number);
        const ports: ParsedPort[] = [];
        const pubStart = published !== undefined ? published : undefined;
        for (let i = 0; i <= endTarget - startTarget; i++) {
            ports.push({
                target: startTarget + i,
                published: pubStart !== undefined ? pubStart + i : undefined,
                protocol,
                hostIp,
            });
        }
        return ports;
    }

    return [{ target, published, protocol, hostIp }];
}

function parsePortNum(s: string): number {
    return parseInt(s, 10);
}

function parsePortLong(obj: Record<string, unknown>): ParsedPort {
    return {
        target: Number(obj.target),
        published: obj.published !== undefined ? Number(obj.published) : undefined,
        protocol: ((obj.protocol as string) || "tcp") as "tcp" | "udp",
        hostIp: (obj.host_ip as string) || "",
    };
}

// --- Volume mount parsing ---

export function parseVolumeMounts(raw: unknown): ParsedVolumeMount[] {
    if (!raw || !Array.isArray(raw)) return [];

    return raw.map((entry): ParsedVolumeMount => {
        if (typeof entry === "string") {
            return parseVolumeMountShort(entry);
        }
        if (typeof entry === "object" && entry !== null) {
            return parseVolumeMountLong(entry as Record<string, unknown>);
        }
        return { type: "volume", source: "", target: "", readOnly: false };
    });
}

function parseVolumeMountShort(str: string): ParsedVolumeMount {
    const parts = str.split(":");
    let readOnly = false;

    // Check for :ro/:rw suffix
    if (parts.length >= 2 && (parts[parts.length - 1] === "ro" || parts[parts.length - 1] === "rw")) {
        readOnly = parts.pop() === "ro";
    }

    if (parts.length === 1) {
        // Anonymous volume: just a container path
        return { type: "volume", source: "", target: parts[0], readOnly };
    }

    const source = parts[0];
    const target = parts[1];
    const type = isBindSource(source) ? "bind" : "volume";

    return { type, source, target, readOnly };
}

function isBindSource(source: string): boolean {
    return source.startsWith("/") || source.startsWith("./") || source.startsWith("../") || source === "." || source === "..";
}

function parseVolumeMountLong(obj: Record<string, unknown>): ParsedVolumeMount {
    const type = (obj.type as string || "volume") as ParsedVolumeMount["type"];
    const mount: ParsedVolumeMount = {
        type,
        source: (obj.source as string) || "",
        target: (obj.target as string) || "",
        readOnly: Boolean(obj.read_only),
    };

    if (type === "bind" && obj.bind && typeof obj.bind === "object") {
        const bind = obj.bind as Record<string, unknown>;
        if (bind.propagation) {
            mount.bindPropagation = bind.propagation as string;
        }
    }

    return mount;
}

// --- Environment parsing ---

export function parseEnvironment(raw: unknown): Record<string, string> {
    if (!raw) return {};

    if (Array.isArray(raw)) {
        const env: Record<string, string> = {};
        for (const item of raw) {
            const str = String(item);
            const eqIdx = str.indexOf("=");
            if (eqIdx === -1) {
                env[str] = "";
            } else {
                env[str.slice(0, eqIdx)] = str.slice(eqIdx + 1);
            }
        }
        return env;
    }

    if (typeof raw === "object") {
        const env: Record<string, string> = {};
        for (const [key, value] of Object.entries(raw as Record<string, unknown>)) {
            env[key] = value === null || value === undefined ? "" : String(value);
        }
        return env;
    }

    return {};
}

// --- Healthcheck parsing ---

export function parseHealthcheck(raw: unknown): ParsedHealthcheck | undefined {
    if (!raw || typeof raw !== "object") return undefined;

    const obj = raw as Record<string, unknown>;

    // Handle disable
    if (obj.disable === true) {
        return { test: ["NONE"], disable: true };
    }

    const hc: ParsedHealthcheck = {
        test: parseHealthcheckTest(obj.test),
        interval: parseDuration(obj.interval as string | number | undefined),
        timeout: parseDuration(obj.timeout as string | number | undefined),
        retries: obj.retries !== undefined ? Number(obj.retries) : undefined,
        startPeriod: parseDuration(obj.start_period as string | number | undefined),
        startInterval: parseDuration(obj.start_interval as string | number | undefined),
    };

    return hc;
}

function parseHealthcheckTest(raw: unknown): string[] {
    if (typeof raw === "string") {
        return ["CMD-SHELL", raw];
    }
    if (Array.isArray(raw)) {
        return raw.map(String);
    }
    return ["CMD-SHELL", "true"];
}

// --- Device parsing ---

export function parseDevices(raw: unknown): Array<{ host: string; container: string; permissions: string }> | undefined {
    if (!raw || !Array.isArray(raw)) return undefined;

    return raw.map((entry) => {
        if (typeof entry === "string") {
            const parts = entry.split(":");
            return {
                host: parts[0] || "",
                container: parts[1] || parts[0] || "",
                permissions: parts[2] || "rwm",
            };
        }
        return { host: "", container: "", permissions: "rwm" };
    });
}

// --- Ulimit parsing ---

export function parseUlimits(raw: unknown): Array<{ name: string; soft: number; hard: number }> | undefined {
    if (!raw || typeof raw !== "object") return undefined;

    const result: Array<{ name: string; soft: number; hard: number }> = [];
    for (const [name, value] of Object.entries(raw as Record<string, unknown>)) {
        if (typeof value === "number") {
            result.push({ name, soft: value, hard: value });
        } else if (typeof value === "object" && value !== null) {
            const obj = value as Record<string, unknown>;
            result.push({
                name,
                soft: Number(obj.soft ?? 0),
                hard: Number(obj.hard ?? 0),
            });
        }
    }
    return result.length > 0 ? result : undefined;
}

// --- Labels parsing ---

function parseLabels(raw: unknown): Record<string, string> {
    if (!raw) return {};

    if (Array.isArray(raw)) {
        const labels: Record<string, string> = {};
        for (const item of raw) {
            const str = String(item);
            const eqIdx = str.indexOf("=");
            if (eqIdx === -1) {
                labels[str] = "";
            } else {
                labels[str.slice(0, eqIdx)] = str.slice(eqIdx + 1);
            }
        }
        return labels;
    }

    if (typeof raw === "object") {
        const labels: Record<string, string> = {};
        for (const [key, value] of Object.entries(raw as Record<string, unknown>)) {
            labels[key] = value === null || value === undefined ? "" : String(value);
        }
        return labels;
    }

    return {};
}

// --- env_file parsing ---

function parseEnvFile(raw: unknown): string[] {
    if (!raw) return [];
    if (typeof raw === "string") return [raw];
    if (Array.isArray(raw)) {
        return raw.map((entry) => {
            if (typeof entry === "string") return entry;
            if (typeof entry === "object" && entry !== null) {
                return (entry as Record<string, unknown>).path as string || "";
            }
            return "";
        }).filter(Boolean);
    }
    return [];
}

// --- Service network parsing ---

function parseServiceNetworks(raw: unknown): ParsedServiceNetwork[] {
    if (!raw) return [];

    if (Array.isArray(raw)) {
        return raw.map((name) => ({ name: String(name) }));
    }

    if (typeof raw === "object") {
        const result: ParsedServiceNetwork[] = [];
        for (const [name, config] of Object.entries(raw as Record<string, unknown>)) {
            const net: ParsedServiceNetwork = { name };
            if (config && typeof config === "object") {
                const cfg = config as Record<string, unknown>;
                if (cfg.aliases && Array.isArray(cfg.aliases)) {
                    net.aliases = cfg.aliases.map(String);
                }
                if (cfg.ipv4_address) net.ipv4Address = String(cfg.ipv4_address);
                if (cfg.ipv6_address) net.ipv6Address = String(cfg.ipv6_address);
            }
            result.push(net);
        }
        return result;
    }

    return [];
}

// --- Service parsing ---

function parseService(raw: Record<string, unknown>): ParsedService {
    const svc: ParsedService = {
        environment: parseEnvironment(raw.environment),
        envFile: parseEnvFile(raw.env_file),
        ports: parsePorts(raw.ports),
        volumes: parseVolumeMounts(raw.volumes),
        networks: parseServiceNetworks(raw.networks),
        restart: (raw.restart as string) || "no",
        labels: parseLabels(raw.labels),
        expose: raw.expose ? (raw.expose as unknown[]).map(String) : [],
    };

    if (raw.image) svc.image = String(raw.image);
    if (raw.build !== undefined) {
        if (typeof raw.build === "string") {
            svc.build = { context: raw.build };
        } else if (typeof raw.build === "object" && raw.build !== null) {
            const b = raw.build as Record<string, unknown>;
            svc.build = {
                context: (b.context as string) || ".",
                dockerfile: b.dockerfile as string | undefined,
            };
        }
    }
    if (raw.command !== undefined) svc.command = raw.command as string | string[];
    if (raw.entrypoint !== undefined) svc.entrypoint = raw.entrypoint as string | string[];
    if (raw.network_mode) svc.networkMode = String(raw.network_mode);
    if (raw.hostname) svc.hostname = String(raw.hostname);
    if (raw.domainname) svc.domainname = String(raw.domainname);
    if (raw.user) svc.user = String(raw.user);
    if (raw.working_dir) svc.workingDir = String(raw.working_dir);
    if (raw.tty !== undefined) svc.tty = Boolean(raw.tty);
    if (raw.stdin_open !== undefined) svc.stdinOpen = Boolean(raw.stdin_open);
    if (raw.privileged !== undefined) svc.privileged = Boolean(raw.privileged);
    if (raw.read_only !== undefined) svc.readOnly = Boolean(raw.read_only);
    if (raw.container_name) svc.containerName = String(raw.container_name);
    if (raw.healthcheck) svc.healthcheck = parseHealthcheck(raw.healthcheck);
    if (raw.dns) svc.dns = Array.isArray(raw.dns) ? raw.dns.map(String) : [String(raw.dns)];
    if (raw.dns_search) svc.dnsSearch = Array.isArray(raw.dns_search) ? raw.dns_search.map(String) : [String(raw.dns_search)];
    if (raw.dns_opt) svc.dnsOpt = Array.isArray(raw.dns_opt) ? raw.dns_opt.map(String) : [String(raw.dns_opt)];
    if (raw.extra_hosts) svc.extraHosts = Array.isArray(raw.extra_hosts) ? raw.extra_hosts.map(String) : [String(raw.extra_hosts)];
    if (raw.cap_add) svc.capAdd = (raw.cap_add as unknown[]).map(String);
    if (raw.cap_drop) svc.capDrop = (raw.cap_drop as unknown[]).map(String);
    if (raw.devices) svc.devices = parseDevices(raw.devices);
    if (raw.ulimits) svc.ulimits = parseUlimits(raw.ulimits);
    if (raw.security_opt) svc.securityOpt = (raw.security_opt as unknown[]).map(String);
    if (raw.pid) svc.pid = String(raw.pid);
    if (raw.ipc) svc.ipc = String(raw.ipc);
    if (raw.init !== undefined) svc.init = Boolean(raw.init);
    if (raw.runtime) svc.runtime = String(raw.runtime);
    if (raw.stop_signal) svc.stopSignal = String(raw.stop_signal);
    if (raw.stop_grace_period) svc.stopGracePeriod = parseDurationSeconds(raw.stop_grace_period as string | number);

    // Logging
    if (raw.logging && typeof raw.logging === "object") {
        const log = raw.logging as Record<string, unknown>;
        svc.logging = {
            driver: (log.driver as string) || "json-file",
            options: (log.options as Record<string, string>) || {},
        };
    }

    // Sysctls
    if (raw.sysctls) {
        if (Array.isArray(raw.sysctls)) {
            svc.sysctls = {};
            for (const item of raw.sysctls) {
                const s = String(item);
                const eq = s.indexOf("=");
                if (eq !== -1) svc.sysctls[s.slice(0, eq)] = s.slice(eq + 1);
            }
        } else {
            svc.sysctls = raw.sysctls as Record<string, string>;
        }
    }

    // Tmpfs
    if (raw.tmpfs) {
        if (typeof raw.tmpfs === "string") {
            svc.tmpfs = [raw.tmpfs];
        } else if (Array.isArray(raw.tmpfs)) {
            svc.tmpfs = raw.tmpfs.map(String);
        }
    }

    // Resource limits from deploy.resources
    if (raw.deploy && typeof raw.deploy === "object") {
        const deploy = raw.deploy as Record<string, unknown>;
        if (deploy.resources && typeof deploy.resources === "object") {
            const resources = deploy.resources as Record<string, unknown>;
            if (resources.limits && typeof resources.limits === "object") {
                const limits = resources.limits as Record<string, unknown>;
                if (limits.memory) svc.memLimit = parseByteSize(limits.memory as string | number);
                if (limits.cpus) svc.cpus = parseFloat(String(limits.cpus));
                if (limits.pids) svc.pidsLimit = Number(limits.pids);
            }
            if (resources.reservations && typeof resources.reservations === "object") {
                const reservations = resources.reservations as Record<string, unknown>;
                if (reservations.memory) svc.memReservation = parseByteSize(reservations.memory as string | number);
                if (reservations.cpus) svc.cpuShares = Math.floor(parseFloat(String(reservations.cpus)) * 1024);
            }
        }
    }

    // Legacy resource limits (non-deploy)
    if (raw.mem_limit) svc.memLimit = parseByteSize(raw.mem_limit as string | number);
    if (raw.mem_reservation) svc.memReservation = parseByteSize(raw.mem_reservation as string | number);
    if (raw.cpus) svc.cpus = parseFloat(String(raw.cpus));
    if (raw.cpu_shares) svc.cpuShares = Number(raw.cpu_shares);
    if (raw.pids_limit) svc.pidsLimit = Number(raw.pids_limit);

    if (raw.shm_size) svc.shmSize = parseByteSize(raw.shm_size as string | number);

    return svc;
}

// --- Network parsing ---

function parseNetwork(raw: unknown): ParsedNetwork {
    if (!raw || typeof raw !== "object") {
        return {
            driver: "bridge",
            internal: false,
            attachable: false,
            external: false,
            labels: {},
            enableIpv4: true,
            enableIpv6: false,
        };
    }

    const obj = raw as Record<string, unknown>;
    const net: ParsedNetwork = {
        driver: (obj.driver as string) || "bridge",
        internal: Boolean(obj.internal),
        attachable: Boolean(obj.attachable),
        external: parseExternal(obj.external),
        labels: parseLabels(obj.labels),
        enableIpv4: obj.enable_ipv4 !== undefined ? Boolean(obj.enable_ipv4) : true,
        enableIpv6: Boolean(obj.enable_ipv6),
    };

    if (obj.name) net.name = String(obj.name);
    if (obj.driver_opts && typeof obj.driver_opts === "object") {
        net.driverOpts = obj.driver_opts as Record<string, string>;
    }

    if (obj.ipam && typeof obj.ipam === "object") {
        const ipamRaw = obj.ipam as Record<string, unknown>;
        net.ipam = {
            driver: ipamRaw.driver as string | undefined,
            config: ipamRaw.config && Array.isArray(ipamRaw.config)
                ? (ipamRaw.config as Array<Record<string, unknown>>).map((c) => ({
                    subnet: c.subnet as string | undefined,
                    ipRange: c.ip_range as string | undefined,
                    gateway: c.gateway as string | undefined,
                }))
                : undefined,
        };
    }

    return net;
}

function parseExternal(raw: unknown): boolean {
    if (raw === true) return true;
    if (typeof raw === "object" && raw !== null) return true;
    return false;
}

// --- Volume parsing ---

function parseVolumeDef(raw: unknown): ParsedVolume {
    if (!raw || typeof raw !== "object") {
        return { driver: "local", external: false, labels: {} };
    }

    const obj = raw as Record<string, unknown>;
    const vol: ParsedVolume = {
        driver: (obj.driver as string) || "local",
        external: parseExternal(obj.external),
        labels: parseLabels(obj.labels),
    };

    if (obj.name) vol.name = String(obj.name);
    if (obj.driver_opts && typeof obj.driver_opts === "object") {
        vol.driverOpts = obj.driver_opts as Record<string, string>;
    }

    return vol;
}

// --- Compose file discovery ---

const COMPOSE_FILENAMES = [
    "compose.yaml",
    "compose.yml",
    "docker-compose.yml",
    "docker-compose.yaml",
];

import { existsSync } from "node:fs";
import { join } from "node:path";

export function findComposeFile(dir: string): string | null {
    for (const name of COMPOSE_FILENAMES) {
        const path = join(dir, name);
        if (existsSync(path)) return path;
    }
    return null;
}

// --- Main entry ---

export function parseCompose(yamlContent: string): ParsedCompose {
    const raw = parseYaml(yamlContent) as Record<string, unknown> | null;

    if (!raw || typeof raw !== "object") {
        return { services: {}, networks: {}, volumes: {} };
    }

    const result: ParsedCompose = {
        services: {},
        networks: {},
        volumes: {},
    };

    // Parse services
    const rawServices = raw.services as Record<string, Record<string, unknown>> | undefined;
    if (rawServices && typeof rawServices === "object") {
        for (const [name, svcRaw] of Object.entries(rawServices)) {
            if (svcRaw && typeof svcRaw === "object") {
                result.services[name] = parseService(svcRaw);
            }
        }
    }

    // Parse networks
    const rawNetworks = raw.networks as Record<string, unknown> | undefined;
    if (rawNetworks && typeof rawNetworks === "object") {
        for (const [name, netRaw] of Object.entries(rawNetworks)) {
            result.networks[name] = parseNetwork(netRaw);
        }
    }

    // Parse volumes
    const rawVolumes = raw.volumes as Record<string, unknown> | undefined;
    if (rawVolumes && typeof rawVolumes === "object") {
        for (const [name, volRaw] of Object.entries(rawVolumes)) {
            result.volumes[name] = parseVolumeDef(volRaw);
        }
    }

    return result;
}
