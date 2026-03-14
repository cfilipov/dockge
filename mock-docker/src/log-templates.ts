import { readFileSync } from "node:fs";
import { join } from "node:path";
import { parse as parseYaml } from "yaml";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface ImageLogTemplate {
    startup: string[];
    heartbeat: {
        interval: number; // milliseconds
        lines: string[];
    };
    shutdown: string[];
}

export interface LogTemplates {
    baseTime: string;
    images: Map<string, ImageLogTemplate>;
    default: ImageLogTemplate;
}

// ---------------------------------------------------------------------------
// Parsing
// ---------------------------------------------------------------------------

function parseDuration(s: string): number {
    const match = s.match(/^(\d+)(ms|s|m)$/);
    if (!match) return 5000;
    const value = parseInt(match[1], 10);
    switch (match[2]) {
        case "ms": return value;
        case "s": return value * 1000;
        case "m": return value * 60000;
        default: return 5000;
    }
}

function parseImageTemplate(raw: Record<string, unknown>): ImageLogTemplate {
    const startup = Array.isArray(raw.startup) ? (raw.startup as string[]) : [];
    const shutdown = Array.isArray(raw.shutdown) ? (raw.shutdown as string[]) : [];

    let heartbeatInterval = 5000;
    let heartbeatLines: string[] = [];
    if (raw.heartbeat && typeof raw.heartbeat === "object") {
        const hb = raw.heartbeat as Record<string, unknown>;
        if (typeof hb.interval === "string") {
            heartbeatInterval = parseDuration(hb.interval);
        }
        if (Array.isArray(hb.lines)) {
            heartbeatLines = hb.lines as string[];
        }
    }

    return {
        startup,
        heartbeat: { interval: heartbeatInterval, lines: heartbeatLines },
        shutdown,
    };
}

const FALLBACK_DEFAULT: ImageLogTemplate = {
    startup: [
        "{{.Image}} service starting",
        "Configuration loaded",
        "Service ready",
    ],
    heartbeat: {
        interval: 3000,
        lines: [
            "[INFO] Health check OK",
            "[INFO] Request processed #{{.N}}",
        ],
    },
    shutdown: [
        "[INFO] Shutting down gracefully",
        "[INFO] Goodbye",
    ],
};

const RESERVED_KEYS = new Set(["base_time"]);

export function parseLogTemplates(yamlContent: string): LogTemplates {
    const raw = parseYaml(yamlContent) as Record<string, unknown> | null;
    if (!raw || typeof raw !== "object") {
        return { baseTime: "", images: new Map(), default: FALLBACK_DEFAULT };
    }

    const baseTime = typeof raw.base_time === "string" ? raw.base_time : "";
    const images = new Map<string, ImageLogTemplate>();
    let defaultTemplate = FALLBACK_DEFAULT;

    for (const [key, value] of Object.entries(raw)) {
        if (RESERVED_KEYS.has(key)) continue;
        if (!value || typeof value !== "object") continue;

        const template = parseImageTemplate(value as Record<string, unknown>);
        if (key === "default") {
            defaultTemplate = template;
        } else {
            images.set(key, template);
        }
    }

    return { baseTime, images, default: defaultTemplate };
}

// ---------------------------------------------------------------------------
// Loading from disk
// ---------------------------------------------------------------------------

export function loadLogTemplates(stacksSourceDir: string): LogTemplates {
    const filePath = join(stacksSourceDir, "log-templates.yaml");
    try {
        const content = readFileSync(filePath, "utf-8");
        return parseLogTemplates(content);
    } catch {
        // File not found or unreadable — return fallback
        return { baseTime: "", images: new Map(), default: FALLBACK_DEFAULT };
    }
}

// ---------------------------------------------------------------------------
// Template lookup
// ---------------------------------------------------------------------------

/**
 * Extract the base image name from a Docker image reference.
 * "alpine:latest" -> "alpine"
 * "nginx:1.25" -> "nginx"
 * "library/redis:7" -> "redis"
 * "ghcr.io/org/app:v1" -> "app"
 */
export function extractBaseImageName(imageRef: string): string {
    // Remove tag/digest
    let name = imageRef.split(":")[0].split("@")[0];
    // Take the last path component (handles registry/org/name)
    const parts = name.split("/");
    name = parts[parts.length - 1];
    return name;
}

export function lookupTemplate(templates: LogTemplates, imageRef: string): ImageLogTemplate {
    const baseName = extractBaseImageName(imageRef);
    return templates.images.get(baseName) ?? templates.default;
}

// ---------------------------------------------------------------------------
// Placeholder expansion
// ---------------------------------------------------------------------------

export function expandPlaceholders(
    line: string,
    vars: { timestamp?: string; image?: string; n?: number },
): string {
    let result = line;
    if (vars.timestamp !== undefined) {
        result = result.replaceAll("{{.Timestamp}}", vars.timestamp);
    }
    if (vars.image !== undefined) {
        result = result.replaceAll("{{.Image}}", vars.image);
    }
    if (vars.n !== undefined) {
        result = result.replaceAll("{{.N}}", String(vars.n));
    }
    return result;
}
