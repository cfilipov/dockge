/**
 * stack-data.ts — Parse compose.yaml + mock.yaml from test-data/stacks and compute
 * expected UI state for data-driven E2E tests.
 *
 * Mirrors the Go backend status computation logic from:
 *   - internal/stack/list.go (stack status aggregation)
 *   - internal/docker/mockdata.go (service state resolution)
 *   - internal/compose/parse.go (dockge.status.ignore label)
 */

import * as fs from "node:fs";
import * as path from "node:path";
import { parse as parseYAML } from "yaml";

// Status constants (match web/src/common/util-common.ts)
const CREATED_FILE = 1;
const RUNNING = 3;
const EXITED = 4;
const RUNNING_AND_EXITED = 5;
const UNHEALTHY = 6;

// ── Types ──

export interface StackTestData {
    name: string;
    composeYAML: string;
    services: ServiceTestData[];
    networks: string[];
    volumes: string[];
    mockStatus: string;
    expectedStackStatus: number;
    expectedStackBadge: { label: string; color: string };
    expectedHasUpdateIcon: boolean;
    expectedHasRecreateIcon: boolean;
}

export interface ServiceTestData {
    name: string;
    composeImage: string;
    runningImage: string;
    ports: string[];
    portDisplays: string[];          // expected display text per port (host part only)
    statusIgnore: boolean;
    updateAvailable: boolean;
    mockState: string;
    mockHealth: string;
    expectedBadge: { label: string; color: string };
    isStarted: boolean;
    hasRecreate: boolean;
}

// ── Mock YAML Schema ──

interface MockYAML {
    status?: string;
    services?: Record<string, {
        state?: string;
        health?: string;
        running_image?: string;
        update_available?: boolean;
    }>;
}

// ── Compose YAML Schema ──

interface ComposeYAML {
    services?: Record<string, {
        image?: string;
        ports?: (string | { published?: number | string; target?: number | string })[];
        labels?: Record<string, string> | string[];
        networks?: string[] | Record<string, unknown>;
        [key: string]: unknown;
    }>;
    networks?: Record<string, unknown> | null;
    volumes?: Record<string, unknown> | null;
}

// ── Constants ──

const LABEL_STATUS_IGNORE = "dockge.status.ignore";

// Hardcoded default statuses for stacks 00-08 (from defaultDevStateMap in mockstate.go)
const HARDCODED_DEFAULTS: Record<string, string> = {
    "00-single-service": "running",
    "01-web-app": "running",
    "02-blog": "running",
    "03-monitoring": "running",
    "04-database": "running",
    "05-multi-service": "running",
    "06-mixed-state": "running",
    "07-full-features": "running",
    "08-env-config": "running",
};

const FEATURED_STACKS = [
    "00-single-service",
    "01-web-app",
    "02-blog",
    "03-monitoring",
    "04-database",
    "05-multi-service",
    "06-mixed-state",
    "07-full-features",
    "08-env-config",
];

// ── Badge Mappings ──

function stackStatusToBadge(status: number): { label: string; color: string } {
    switch (status) {
        case UNHEALTHY: return { label: "unhealthy", color: "bg-danger" };
        case RUNNING: return { label: "active", color: "bg-primary" };
        case RUNNING_AND_EXITED: return { label: "partially", color: "bg-info" };
        case EXITED: return { label: "exited", color: "bg-warning" };
        case CREATED_FILE: return { label: "down", color: "bg-dark" };
        default: return { label: "down", color: "bg-secondary" };
    }
}

function containerStateToBadge(state: string, health: string): { label: string; color: string } {
    if (state === "running" && health === "unhealthy") return { label: "unhealthy", color: "bg-danger" };
    if (state === "running") return { label: "running", color: "bg-primary" };
    if (state === "exited") return { label: "exited", color: "bg-warning" };
    // "down" = no container exists (inactive stack)
    return { label: "down", color: "bg-secondary" };
}

// ── Label Parsing ──

function getLabel(labels: Record<string, string> | string[] | undefined, key: string): string | undefined {
    if (!labels) return undefined;
    if (Array.isArray(labels)) {
        for (const entry of labels) {
            const [k, ...rest] = entry.split("=");
            if (k === key) return rest.join("=");
        }
        return undefined;
    }
    return labels[key];
}

function hasStatusIgnore(labels: Record<string, string> | string[] | undefined): boolean {
    return getLabel(labels, LABEL_STATUS_IGNORE) === "true";
}

// ── Port Parsing ──

function parsePorts(ports?: (string | { published?: number | string; target?: number | string })[]): string[] {
    if (!ports) return [];
    return ports.map(p => {
        if (typeof p === "string") return p;
        if (typeof p === "number") return String(p);
        // Object form: { published, target }
        return `${p.published}:${p.target}`;
    });
}

/** Mirrors parseDockerPort().display from web/src/common/util-common.ts:477-548 */
function portToDisplay(portStr: string): string {
    // Strip protocol suffix (e.g., "/tcp", "/udp")
    const slashIdx = portStr.indexOf("/");
    const part1 = slashIdx >= 0 ? portStr.substring(0, slashIdx) : portStr;

    // Handle docker ps arrow format (e.g., "0.0.0.0:8080->80/tcp")
    const arrowIdx = part1.indexOf("->");
    const cleaned = arrowIdx >= 0
        ? (() => {
            const hostSide = part1.substring(0, arrowIdx);
            const colonIdx = hostSide.indexOf(":");
            return colonIdx >= 0 ? hostSide.substring(colonIdx + 1) : hostSide;
        })()
        : part1;

    // Split on last colon
    const lastColon = cleaned.lastIndexOf(":");
    if (lastColon === -1) {
        // No colon — just a port or port range
        return cleaned;
    }
    // Has colon — display is the host part (before last colon)
    return cleaned.substring(0, lastColon);
}

// ── Service State Resolution ──

/** Mirrors MockData.GetServiceState from mockdata.go:304-317 */
function resolveServiceState(stackStatus: string, svcMock?: { state?: string }): string {
    switch (stackStatus) {
        case "running":
        case "paused":
            return svcMock?.state || "running";
        case "inactive":
            // No containers exist — not exited, just "down" (UNKNOWN)
            return "down";
        default:
            // exited stacks: all services get "exited"
            return "exited";
    }
}

function resolveServiceHealth(svcMock?: { health?: string }): string {
    return svcMock?.health || "";
}

// ── Main Loader ──

export function loadStackData(stacksDir: string, stackName: string): StackTestData {
    const stackDir = path.join(stacksDir, stackName);
    const composePath = path.join(stackDir, "compose.yaml");
    const mockPath = path.join(stackDir, "mock.yaml");

    // Read compose.yaml
    const composeYAML = fs.readFileSync(composePath, "utf-8");
    const compose: ComposeYAML = parseYAML(composeYAML) || {};

    // Read mock.yaml (optional)
    let mock: MockYAML = {};
    if (fs.existsSync(mockPath)) {
        mock = parseYAML(fs.readFileSync(mockPath, "utf-8")) || {};
    }

    // Determine effective stack-level status
    // mock.yaml status overrides hardcoded default
    const hardcodedStatus = HARDCODED_DEFAULTS[stackName] || "running";
    const mockStatus = mock.status || hardcodedStatus;

    // Parse top-level networks and volumes
    const networks = compose.networks ? Object.keys(compose.networks) : [];
    const volumes = compose.volumes ? Object.keys(compose.volumes) : [];

    // Build service data
    const services: ServiceTestData[] = [];
    const composeServices = compose.services || {};

    for (const [svcName, svcDef] of Object.entries(composeServices)) {
        const composeImage = svcDef.image || "";
        const svcMock = mock.services?.[svcName];
        const runningImage = svcMock?.running_image || composeImage;
        const updateAvailable = svcMock?.update_available || false;
        const statusIgnore = hasStatusIgnore(svcDef.labels);

        // Resolve mock state/health
        const mockState = resolveServiceState(mockStatus, svcMock);
        const mockHealth = resolveServiceHealth(svcMock);

        // For containers, the actual "state" field passed to the frontend:
        // - If health is "healthy" or "starting", state is still "running"
        // - If health is "unhealthy", state is "running" but badge shows "unhealthy"
        const effectiveState = mockState === "down" ? "down" : mockState;
        const badge = containerStateToBadge(effectiveState, mockHealth);

        const isStarted = effectiveState === "running" && mockHealth !== "unhealthy"
            ? true
            : effectiveState === "running" && mockHealth === "unhealthy"
                ? true
                : false;

        const ports = parsePorts(svcDef.ports);
        // Compute display text for static ports (skip env-var ports — envsubst is server-side)
        const portDisplays = ports
            .filter(p => !p.includes("${"))
            .map(p => portToDisplay(p));

        services.push({
            name: svcName,
            composeImage,
            runningImage,
            ports,
            portDisplays,
            statusIgnore,
            updateAvailable,
            mockState: effectiveState,
            mockHealth,
            expectedBadge: badge,
            isStarted,
            hasRecreate: runningImage !== composeImage,
        });
    }

    // Compute stack-level status from services (mirrors list.go:147-160)
    const expectedStackStatus = computeStackStatus(services, mockStatus);
    const expectedStackBadge = stackStatusToBadge(expectedStackStatus);

    // Stack is "started" if RUNNING, RUNNING_AND_EXITED, or UNHEALTHY
    const stackStarted = expectedStackStatus === RUNNING ||
        expectedStackStatus === RUNNING_AND_EXITED ||
        expectedStackStatus === UNHEALTHY;

    // Recreate icon: stack.started && any service has runningImage !== composeImage
    const expectedHasRecreateIcon = stackStarted &&
        services.some(s => s.hasRecreate);

    // Update icon: any service has update_available (from BoltDB, seeded from mock.yaml)
    const expectedHasUpdateIcon = services.some(s => s.updateAvailable);

    return {
        name: stackName,
        composeYAML,
        services,
        networks,
        volumes,
        mockStatus,
        expectedStackStatus,
        expectedStackBadge,
        expectedHasUpdateIcon,
        expectedHasRecreateIcon,
    };
}

/** Compute stack status from service states (mirrors internal/stack/list.go:147-160) */
function computeStackStatus(services: ServiceTestData[], mockStatus: string): number {
    if (mockStatus === "inactive") {
        return CREATED_FILE; // No containers → "down"
    }

    let running = 0;
    let exited = 0;
    let unhealthy = 0;

    for (const svc of services) {
        if (svc.statusIgnore) continue;

        // Health takes priority
        if (svc.mockHealth === "unhealthy") {
            unhealthy++;
        } else if (svc.mockState === "running") {
            running++;
        } else if (svc.mockState === "exited") {
            exited++;
        }
    }

    // Priority chain (matches list.go)
    if (unhealthy > 0) return UNHEALTHY;
    if (running > 0 && exited > 0) return RUNNING_AND_EXITED;
    if (running > 0) return RUNNING;
    if (exited > 0) return EXITED;

    return CREATED_FILE;
}

export function loadAllStacks(stacksDir: string): StackTestData[] {
    return FEATURED_STACKS.map(name => loadStackData(stacksDir, name));
}

/** Returns expected Docker network names (stackName_networkName or stackName_default) */
export function getExpectedNetworkNames(stacks: StackTestData[]): string[] {
    const names: string[] = [];
    for (const stack of stacks) {
        if (stack.networks.length === 0) {
            // Stacks without explicit networks get a _default network
            // (unless network_mode: host is used for all services)
            const hasHostNetwork = stack.services.length > 0; // check compose
            if (!hasHostNetwork) {
                names.push(`${stack.name}_default`);
            }
        } else {
            for (const net of stack.networks) {
                names.push(`${stack.name}_${net}`);
            }
        }
    }
    return names;
}

/** Returns expected Docker volume names (stackName_volumeName) */
export function getExpectedVolumeNames(stacks: StackTestData[]): string[] {
    const names: string[] = [];
    for (const stack of stacks) {
        for (const vol of stack.volumes) {
            names.push(`${stack.name}_${vol}`);
        }
    }
    return names;
}

/** Returns all unique image refs from compose + running overrides */
export function getExpectedImages(stacks: StackTestData[]): string[] {
    const images = new Set<string>();
    for (const stack of stacks) {
        for (const svc of stack.services) {
            if (svc.composeImage) images.add(svc.composeImage);
            if (svc.runningImage && svc.runningImage !== svc.composeImage) {
                images.add(svc.runningImage);
            }
        }
    }
    return [...images].sort();
}
