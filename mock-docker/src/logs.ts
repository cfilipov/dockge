import type { ContainerInspect } from "./types.js";
import type { Clock } from "./clock.js";
import type { LogTemplates } from "./log-templates.js";
import { lookupTemplate, expandPlaceholders, extractBaseImageName } from "./log-templates.js";
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
// Helpers
// ---------------------------------------------------------------------------

export function formatTimestamp(date: Date): string {
    return date.toISOString();
}

function getImageRef(container: ContainerInspect): string {
    return container.Config.Image || "unknown";
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

export function generateStartupLogs(
    container: ContainerInspect,
    baseTime: Date,
    templates?: LogTemplates | null,
): string[] {
    const imageRef = getImageRef(container);
    const tmpl = templates ? lookupTemplate(templates, imageRef) : null;

    if (tmpl) {
        const lines: string[] = [];
        const startMs = baseTime.getTime();
        for (let i = 0; i < tmpl.startup.length; i++) {
            const ts = formatTimestamp(new Date(startMs + i * 100));
            lines.push(expandPlaceholders(tmpl.startup[i], {
                timestamp: ts,
                image: extractBaseImageName(imageRef),
                n: i,
            }));
        }
        return lines;
    }

    // Fallback: generic logs (legacy behavior)
    return generateGenericStartupLogs(container, baseTime);
}

export function generateShutdownLogs(
    container: ContainerInspect,
    clock: Clock,
    templates?: LogTemplates | null,
): string[] {
    const imageRef = getImageRef(container);
    const tmpl = templates ? lookupTemplate(templates, imageRef) : null;

    if (tmpl) {
        const lines: string[] = [];
        const baseTime = clock.now();
        for (let i = 0; i < tmpl.shutdown.length; i++) {
            const ts = formatTimestamp(new Date(baseTime.getTime() + i * 100));
            lines.push(expandPlaceholders(tmpl.shutdown[i], {
                timestamp: ts,
                image: extractBaseImageName(imageRef),
                n: i,
            }));
        }
        return lines;
    }

    // Fallback: generic logs (legacy behavior)
    return generateGenericShutdownLogs(container, clock);
}

export function generatePeriodicLogLine(
    container: ContainerInspect,
    lineNumber: number,
    clock: Clock,
    templates?: LogTemplates | null,
): string {
    const imageRef = getImageRef(container);
    const tmpl = templates ? lookupTemplate(templates, imageRef) : null;

    if (tmpl && tmpl.heartbeat.lines.length > 0) {
        const seed = containerSeed(container);
        const lineSeed = seed + String(lineNumber);
        const idx = deterministicInt(lineSeed + "hb-msg", 0, tmpl.heartbeat.lines.length - 1);
        const ts = formatTimestamp(clock.now());
        return expandPlaceholders(tmpl.heartbeat.lines[idx], {
            timestamp: ts,
            image: extractBaseImageName(imageRef),
            n: lineNumber,
        });
    }

    // Fallback: generic logs (legacy behavior)
    return generateGenericPeriodicLogLine(container, lineNumber, clock);
}

export interface TimestampedLine {
    ts: number;
    line: string;
}

export function getHistoricalLogs(
    container: ContainerInspect,
    clock: Clock,
    opts: { tail?: number; since?: number; until?: number; e2eMode?: boolean } = {},
    templates?: LogTemplates | null,
): TimestampedLine[] {
    const imageRef = getImageRef(container);
    const tmpl = templates ? lookupTemplate(templates, imageRef) : null;

    const seed = containerSeed(container);
    const e2e = opts.e2eMode ?? false;
    const startedAt = new Date(container.State.StartedAt).getTime();
    const now = clock.now().getTime();

    const allLines: { ts: number; line: string }[] = [];

    if (tmpl) {
        // Startup lines from template
        for (let i = 0; i < tmpl.startup.length; i++) {
            const ts = startedAt + i * 100;
            const line = expandPlaceholders(tmpl.startup[i], {
                timestamp: formatTimestamp(new Date(ts)),
                image: extractBaseImageName(imageRef),
                n: i,
            });
            allLines.push({ ts, line });
        }

        // In e2e mode: just 1 heartbeat line. Normal mode: fill to 100.
        const startupCount = tmpl.startup.length;
        const periodicCount = e2e ? 1 : (100 - startupCount);
        const span = Math.max(now - startedAt - startupCount * 100, 1);
        const interval = periodicCount > 1 ? span / periodicCount : span;

        for (let i = 0; i < periodicCount; i++) {
            const ts = startedAt + startupCount * 100 + Math.floor(i * interval);
            const lineSeed = seed + String(i);
            const idx = deterministicInt(lineSeed + "hb-msg", 0, tmpl.heartbeat.lines.length - 1);
            const line = expandPlaceholders(tmpl.heartbeat.lines[idx], {
                timestamp: formatTimestamp(new Date(ts)),
                image: extractBaseImageName(imageRef),
                n: i,
            });
            allLines.push({ ts, line });
        }
    } else {
        // Fallback: generic logs (legacy behavior)
        return getGenericHistoricalLogs(container, clock, opts);
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

    return filtered;
}

// ===========================================================================
// Generic fallback (original hardcoded templates)
// ===========================================================================

const LOG_LEVELS = ["INFO", "DEBUG", "WARN", "ERROR"] as const;
const COMPONENTS = ["server", "config", "runtime", "http", "db", "cache", "scheduler", "worker"] as const;

const GENERIC_STARTUP_TEMPLATES = [
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

const GENERIC_SHUTDOWN_TEMPLATES = [
    "Received shutdown signal",
    "Closing active connections",
    "Shutting down gracefully",
];

const GENERIC_PERIODIC_TEMPLATES = [
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

function fillGenericTemplate(template: string, seed: string): string {
    return template
        .replace("{port}", String(deterministicInt(seed + "port", 3000, 9999)))
        .replace("{ratio}", String(deterministicInt(seed + "ratio", 60, 99)))
        .replace("{connections}", String(deterministicInt(seed + "conn", 1, 200)))
        .replace("{latency}", String(deterministicInt(seed + "lat", 1, 500)))
        .replace("{memory}", String(deterministicInt(seed + "mem", 32, 512)));
}

function generateGenericStartupLogs(container: ContainerInspect, baseTime: Date): string[] {
    const seed = containerSeed(container);
    const count = deterministicInt(seed + "startup-count", 5, 8);
    const port = getFirstExposedPort(container);
    const lines: string[] = [];
    const startMs = baseTime.getTime();

    for (let i = 0; i < count; i++) {
        const ts = formatTimestamp(new Date(startMs + i * 100));
        let msg = GENERIC_STARTUP_TEMPLATES[i % GENERIC_STARTUP_TEMPLATES.length];
        msg = msg.replace("{port}", String(port));
        lines.push(`${ts} INFO [server] ${msg}`);
    }
    return lines;
}

function generateGenericShutdownLogs(container: ContainerInspect, clock: Clock): string[] {
    const seed = containerSeed(container);
    const count = deterministicInt(seed + "shutdown-count", 2, 3);
    const lines: string[] = [];
    const baseTime = clock.now();

    for (let i = 0; i < count; i++) {
        const ts = formatTimestamp(new Date(baseTime.getTime() + i * 100));
        const msg = GENERIC_SHUTDOWN_TEMPLATES[i % GENERIC_SHUTDOWN_TEMPLATES.length];
        lines.push(`${ts} INFO [server] ${msg}`);
    }
    return lines;
}

function generateGenericPeriodicLogLine(container: ContainerInspect, lineNumber: number, clock: Clock): string {
    const seed = containerSeed(container);
    const lineSeed = seed + String(lineNumber);
    const level = LOG_LEVELS[deterministicInt(lineSeed + "level", 0, LOG_LEVELS.length - 1)];
    const component = COMPONENTS[deterministicInt(lineSeed + "component", 0, COMPONENTS.length - 1)];
    const templateIdx = deterministicInt(lineSeed + "msg", 0, GENERIC_PERIODIC_TEMPLATES.length - 1);
    const msg = fillGenericTemplate(GENERIC_PERIODIC_TEMPLATES[templateIdx], lineSeed);
    const ts = formatTimestamp(clock.now());
    return `${ts} ${level} [${component}] ${msg}`;
}

function getGenericHistoricalLogs(
    container: ContainerInspect,
    clock: Clock,
    opts: { tail?: number; since?: number; until?: number; e2eMode?: boolean } = {},
): TimestampedLine[] {
    const seed = containerSeed(container);
    const e2e = opts.e2eMode ?? false;
    const startedAt = new Date(container.State.StartedAt).getTime();
    const now = clock.now().getTime();

    const startupCount = deterministicInt(seed + "startup-count", 5, 8);
    const allLines: { ts: number; line: string }[] = [];

    for (let i = 0; i < startupCount; i++) {
        const ts = startedAt + i * 100;
        let msg = GENERIC_STARTUP_TEMPLATES[i % GENERIC_STARTUP_TEMPLATES.length];
        const port = getFirstExposedPort(container);
        msg = msg.replace("{port}", String(port));
        allLines.push({
            ts,
            line: `${formatTimestamp(new Date(ts))} INFO [server] ${msg}`,
        });
    }

    const periodicCount = e2e ? 1 : (100 - startupCount);
    const span = Math.max(now - startedAt - startupCount * 100, 1);
    const interval = periodicCount > 1 ? span / periodicCount : span;

    for (let i = 0; i < periodicCount; i++) {
        const ts = startedAt + startupCount * 100 + Math.floor(i * interval);
        const lineSeed = seed + String(i);
        const level = LOG_LEVELS[deterministicInt(lineSeed + "level", 0, LOG_LEVELS.length - 1)];
        const component = COMPONENTS[deterministicInt(lineSeed + "component", 0, COMPONENTS.length - 1)];
        const templateIdx = deterministicInt(lineSeed + "msg", 0, GENERIC_PERIODIC_TEMPLATES.length - 1);
        const msg = fillGenericTemplate(GENERIC_PERIODIC_TEMPLATES[templateIdx], lineSeed);
        allLines.push({
            ts,
            line: `${formatTimestamp(new Date(ts))} ${level} [${component}] ${msg}`,
        });
    }

    let filtered = allLines;
    if (opts.since !== undefined) {
        const sinceMs = opts.since * 1000;
        filtered = filtered.filter((l) => l.ts >= sinceMs);
    }
    if (opts.until !== undefined) {
        const untilMs = opts.until * 1000;
        filtered = filtered.filter((l) => l.ts <= untilMs);
    }

    if (opts.tail !== undefined) {
        if (opts.tail <= 0) return [];
        filtered = filtered.slice(-opts.tail);
    }

    return filtered;
}
