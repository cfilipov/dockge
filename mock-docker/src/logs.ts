import type { ContainerInspect } from "./types.js";
import type { Clock } from "./clock.js";
import { deterministicInt, hashToSeed, serviceSeed } from "./deterministic.js";

// ---------------------------------------------------------------------------
// Seed derivation
// ---------------------------------------------------------------------------

function containerSeed(container: ContainerInspect): string {
    const labels = container.Config.Labels ?? {};
    const project = labels["com.docker.compose.project"];
    const service = labels["com.docker.compose.service"];
    if (project && service) return serviceSeed(project, service);
    return hashToSeed(["portge-mock-v1", "container", container.Id]);
}

// ---------------------------------------------------------------------------
// Templates
// ---------------------------------------------------------------------------

const LOG_LEVELS = ["INFO", "DEBUG", "WARN", "ERROR"] as const;
const COMPONENTS = ["server", "config", "runtime", "http", "db", "cache", "scheduler", "worker"] as const;

const STARTUP_TEMPLATES = [
    "Initializing application...",
    "Loading configuration from environment",
    "Connecting to database",
    "Database connection established",
    "Starting HTTP server",
    "Listening on port {port}",
    "Registering middleware",
    "Health check endpoint ready",
    "Application started successfully",
    "Ready to accept connections",
];

const SHUTDOWN_TEMPLATES = [
    "Received shutdown signal",
    "Closing active connections",
    "Shutting down gracefully",
];

const PERIODIC_TEMPLATES = [
    "Handling request from client",
    "Processing background job",
    "Health check passed",
    "Cache hit ratio: {ratio}%",
    "Active connections: {connections}",
    "Request completed in {latency}ms",
    "Scheduled task executed",
    "Memory usage: {memory}MB",
    "Garbage collection completed",
    "Metrics exported successfully",
];

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function getFirstExposedPort(container: ContainerInspect): number {
    const exposed = container.Config.ExposedPorts;
    if (exposed) {
        const first = Object.keys(exposed)[0];
        if (first) {
            const port = parseInt(first.split("/")[0], 10);
            if (!isNaN(port)) return port;
        }
    }
    return 8080;
}

function formatTimestamp(date: Date): string {
    return date.toISOString();
}

function fillTemplate(template: string, seed: string): string {
    return template
        .replace("{port}", String(deterministicInt(seed + "port", 3000, 9999)))
        .replace("{ratio}", String(deterministicInt(seed + "ratio", 60, 99)))
        .replace("{connections}", String(deterministicInt(seed + "conn", 1, 200)))
        .replace("{latency}", String(deterministicInt(seed + "lat", 1, 500)))
        .replace("{memory}", String(deterministicInt(seed + "mem", 32, 512)));
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

export function generateStartupLogs(container: ContainerInspect, clock: Clock): string[] {
    const seed = containerSeed(container);
    const count = deterministicInt(seed + "startup-count", 5, 8);
    const port = getFirstExposedPort(container);
    const lines: string[] = [];
    const baseTime = new Date(clock.now().getTime() - count * 100); // spread 100ms apart

    for (let i = 0; i < count; i++) {
        const ts = formatTimestamp(new Date(baseTime.getTime() + i * 100));
        let msg = STARTUP_TEMPLATES[i % STARTUP_TEMPLATES.length];
        msg = msg.replace("{port}", String(port));
        lines.push(`${ts} INFO [server] ${msg}`);
    }
    return lines;
}

export function generateShutdownLogs(container: ContainerInspect, clock: Clock): string[] {
    const seed = containerSeed(container);
    const count = deterministicInt(seed + "shutdown-count", 2, 3);
    const lines: string[] = [];
    const baseTime = clock.now();

    for (let i = 0; i < count; i++) {
        const ts = formatTimestamp(new Date(baseTime.getTime() + i * 100));
        const msg = SHUTDOWN_TEMPLATES[i % SHUTDOWN_TEMPLATES.length];
        lines.push(`${ts} INFO [server] ${msg}`);
    }
    return lines;
}

export function generatePeriodicLogLine(container: ContainerInspect, lineNumber: number, clock: Clock): string {
    const seed = containerSeed(container);
    const lineSeed = seed + String(lineNumber);
    const level = LOG_LEVELS[deterministicInt(lineSeed + "level", 0, LOG_LEVELS.length - 1)];
    const component = COMPONENTS[deterministicInt(lineSeed + "component", 0, COMPONENTS.length - 1)];
    const templateIdx = deterministicInt(lineSeed + "msg", 0, PERIODIC_TEMPLATES.length - 1);
    const msg = fillTemplate(PERIODIC_TEMPLATES[templateIdx], lineSeed);
    const ts = formatTimestamp(clock.now());
    return `${ts} ${level} [${component}] ${msg}`;
}

export function getHistoricalLogs(
    container: ContainerInspect,
    clock: Clock,
    opts: { tail?: number; since?: number; until?: number } = {},
): string[] {
    const seed = containerSeed(container);
    const totalLines = 100;
    const startedAt = new Date(container.State.StartedAt).getTime();
    const now = clock.now().getTime();

    // Generate startup logs first
    const startupCount = deterministicInt(seed + "startup-count", 5, 8);
    const allLines: { ts: number; line: string }[] = [];

    // Startup lines
    for (let i = 0; i < startupCount; i++) {
        const ts = startedAt + i * 100;
        let msg = STARTUP_TEMPLATES[i % STARTUP_TEMPLATES.length];
        const port = getFirstExposedPort(container);
        msg = msg.replace("{port}", String(port));
        allLines.push({
            ts,
            line: `${formatTimestamp(new Date(ts))} INFO [server] ${msg}`,
        });
    }

    // Periodic lines — evenly spread between startup and now
    const periodicCount = totalLines - startupCount;
    const span = Math.max(now - startedAt - startupCount * 100, 1);
    const interval = span / periodicCount;

    for (let i = 0; i < periodicCount; i++) {
        const ts = startedAt + startupCount * 100 + Math.floor(i * interval);
        const lineSeed = seed + String(i);
        const level = LOG_LEVELS[deterministicInt(lineSeed + "level", 0, LOG_LEVELS.length - 1)];
        const component = COMPONENTS[deterministicInt(lineSeed + "component", 0, COMPONENTS.length - 1)];
        const templateIdx = deterministicInt(lineSeed + "msg", 0, PERIODIC_TEMPLATES.length - 1);
        const msg = fillTemplate(PERIODIC_TEMPLATES[templateIdx], lineSeed);
        allLines.push({
            ts,
            line: `${formatTimestamp(new Date(ts))} ${level} [${component}] ${msg}`,
        });
    }

    // Apply since/until filters
    let filtered = allLines;
    if (opts.since !== undefined) {
        const sinceMs = opts.since * 1000;
        filtered = filtered.filter((l) => l.ts >= sinceMs);
    }
    if (opts.until !== undefined) {
        const untilMs = opts.until * 1000;
        filtered = filtered.filter((l) => l.ts <= untilMs);
    }

    // Apply tail
    if (opts.tail !== undefined) {
        if (opts.tail <= 0) return [];
        filtered = filtered.slice(-opts.tail);
    }

    return filtered.map((l) => l.line);
}
