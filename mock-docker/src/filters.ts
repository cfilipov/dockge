// Generic filter engine for Docker list endpoints.

import type { ContainerInspect, NetworkInspect, VolumeInspect, ImageInspect } from "./types.js";
import type { ParsedFilters, DockerEvent } from "./list-types.js";

/**
 * Parse the JSON-encoded filters query parameter into a Map.
 * Returns empty Map for undefined, empty string, or malformed JSON.
 */
export function parseFilters(queryParam: string | undefined): ParsedFilters {
    if (!queryParam) return new Map();
    try {
        const parsed = JSON.parse(queryParam);
        if (typeof parsed !== "object" || parsed === null || Array.isArray(parsed)) {
            return new Map();
        }
        const result: ParsedFilters = new Map();
        for (const [key, value] of Object.entries(parsed)) {
            if (Array.isArray(value) && value.every((v) => typeof v === "string")) {
                // Array format: {"label": ["key=val"]}
                result.set(key, value as string[]);
            } else if (typeof value === "object" && value !== null && !Array.isArray(value)) {
                // Map format (Go SDK): {"label": {"key=val": true}}
                result.set(key, Object.keys(value as Record<string, unknown>));
            }
        }
        return result;
    } catch {
        return new Map();
    }
}

/**
 * Check if a labels map matches a label filter string.
 * "key=value" → exact match on key and value.
 * "key" → key exists.
 */
function matchLabel(labels: Record<string, string>, filter: string): boolean {
    const eqIdx = filter.indexOf("=");
    if (eqIdx !== -1) {
        const key = filter.slice(0, eqIdx);
        const value = filter.slice(eqIdx + 1);
        return labels[key] === value;
    }
    return filter in labels;
}

/**
 * Check if any value in the filter array matches using a predicate.
 * This implements the OR-within-key logic.
 */
function matchAny(values: string[], predicate: (v: string) => boolean): boolean {
    return values.some(predicate);
}

/**
 * Check if a container matches an ancestor filter value.
 * Matches: exact Config.Image, exact Image (sha256 ID), image ID prefix,
 * or image name without tag (e.g. "nginx" matches "nginx:latest").
 */
function matchAncestor(c: ContainerInspect, v: string): boolean {
    const configImage = c.Config.Image ?? "";
    // Exact match on image name or sha256 ID
    if (configImage === v || c.Image === v) return true;
    // Image ID prefix match (e.g. "sha256:abc" matches "sha256:abcdef...")
    if (c.Image.startsWith(v)) return true;
    // Name without tag: "nginx" should match "nginx:latest"
    // Strip tag from Config.Image and compare
    const nameWithoutTag = configImage.split(":")[0];
    if (nameWithoutTag && nameWithoutTag === v) return true;
    return false;
}

/**
 * Filter containers. Logic: OR within a key, AND across keys.
 */
export function applyContainerFilters(containers: ContainerInspect[], filters: ParsedFilters): ContainerInspect[] {
    if (filters.size === 0) return containers;

    return containers.filter((c) => {
        for (const [key, values] of filters) {
            let matches = false;
            switch (key) {
                case "id":
                    matches = matchAny(values, (v) => c.Id.startsWith(v));
                    break;
                case "name": {
                    const name = c.Name.startsWith("/") ? c.Name.slice(1) : c.Name;
                    matches = matchAny(values, (v) => {
                        const vName = v.startsWith("/") ? v.slice(1) : v;
                        return name.includes(vName);
                    });
                    break;
                }
                case "label":
                    matches = matchAny(values, (v) => matchLabel(c.Config.Labels ?? {}, v));
                    break;
                case "status":
                    matches = matchAny(values, (v) => c.State.Status === v);
                    break;
                case "ancestor":
                    matches = matchAny(values, (v) => matchAncestor(c, v));
                    break;
                case "network":
                    matches = matchAny(values, (v) => {
                        const nets = c.NetworkSettings.Networks ?? {};
                        for (const [netName, endpoint] of Object.entries(nets)) {
                            if (netName === v || endpoint.NetworkID === v) return true;
                        }
                        return false;
                    });
                    break;
                case "volume":
                    matches = matchAny(values, (v) => {
                        return (c.Mounts ?? []).some((m) => m.Name === v);
                    });
                    break;
                default:
                    // Unknown filter key — skip (Docker ignores unknown keys)
                    matches = true;
            }
            if (!matches) return false;
        }
        return true;
    });
}

/**
 * Filter networks.
 */
export function applyNetworkFilters(networks: NetworkInspect[], filters: ParsedFilters): NetworkInspect[] {
    if (filters.size === 0) return networks;

    const BUILTIN_NAMES = new Set(["bridge", "host", "none"]);

    return networks.filter((n) => {
        for (const [key, values] of filters) {
            let matches = false;
            switch (key) {
                case "id":
                    matches = matchAny(values, (v) => n.Id.startsWith(v));
                    break;
                case "name":
                    matches = matchAny(values, (v) => n.Name === v);
                    break;
                case "label":
                    matches = matchAny(values, (v) => matchLabel(n.Labels ?? {}, v));
                    break;
                case "driver":
                    matches = matchAny(values, (v) => n.Driver === v);
                    break;
                case "scope":
                    matches = matchAny(values, (v) => n.Scope === v);
                    break;
                case "type":
                    matches = matchAny(values, (v) => {
                        const isBuiltin = BUILTIN_NAMES.has(n.Name);
                        if (v === "builtin") return isBuiltin;
                        if (v === "custom") return !isBuiltin;
                        return false;
                    });
                    break;
                default:
                    matches = true;
            }
            if (!matches) return false;
        }
        return true;
    });
}

/**
 * Filter volumes.
 */
export function applyVolumeFilters(
    volumes: VolumeInspect[],
    filters: ParsedFilters,
    containers?: Map<string, ContainerInspect>,
): VolumeInspect[] {
    if (filters.size === 0) return volumes;

    return volumes.filter((vol) => {
        for (const [key, values] of filters) {
            let matches = false;
            switch (key) {
                case "name":
                    matches = matchAny(values, (v) => vol.Name === v);
                    break;
                case "label":
                    matches = matchAny(values, (v) => matchLabel(vol.Labels ?? {}, v));
                    break;
                case "driver":
                    matches = matchAny(values, (v) => vol.Driver === v);
                    break;
                case "dangling":
                    matches = matchAny(values, (v) => {
                        if (!containers) return true; // can't evaluate without containers
                        const isDangling = !isVolumeInUse(vol.Name, containers);
                        return v === "true" ? isDangling : !isDangling;
                    });
                    break;
                default:
                    matches = true;
            }
            if (!matches) return false;
        }
        return true;
    });
}

function isVolumeInUse(volumeName: string, containers: Map<string, ContainerInspect>): boolean {
    for (const c of containers.values()) {
        if ((c.Mounts ?? []).some((m) => m.Name === volumeName)) return true;
    }
    return false;
}

/**
 * Filter images.
 */
export function applyImageFilters(images: ImageInspect[], filters: ParsedFilters): ImageInspect[] {
    if (filters.size === 0) return images;

    return images.filter((img) => {
        for (const [key, values] of filters) {
            let matches = false;
            switch (key) {
                case "label":
                    matches = matchAny(values, (v) => matchLabel(img.Config?.Labels ?? {}, v));
                    break;
                case "dangling":
                    matches = matchAny(values, (v) => {
                        const isDangling = !img.RepoTags || img.RepoTags.length === 0;
                        return v === "true" ? isDangling : !isDangling;
                    });
                    break;
                case "reference":
                    matches = matchAny(values, (pattern) => {
                        return (img.RepoTags ?? []).some((tag) => globMatch(tag, pattern));
                    });
                    break;
                default:
                    matches = true;
            }
            if (!matches) return false;
        }
        return true;
    });
}

/**
 * Simple glob match supporting * wildcard.
 */
function globMatch(str: string, pattern: string): boolean {
    // Convert glob pattern to regex: escape everything except *, then replace * with .*
    const escaped = pattern.replace(/[.+^${}()|[\]\\]/g, "\\$&").replace(/\*/g, ".*");
    return new RegExp(`^${escaped}$`).test(str);
}

/**
 * Check if an event passes the given filters. Returns true if event matches.
 */
export function applyEventFilters(event: DockerEvent, filters: ParsedFilters): boolean {
    if (filters.size === 0) return true;

    for (const [key, values] of filters) {
        let matches = false;
        switch (key) {
            case "type":
                matches = matchAny(values, (v) => event.Type === v);
                break;
            case "event":
                matches = matchAny(values, (v) => event.Action === v);
                break;
            case "container":
            case "image":
            case "network":
            case "volume":
                matches = matchAny(values, (v) => event.Actor.ID === v || event.Actor.ID.startsWith(v));
                break;
            case "label":
                matches = matchAny(values, (v) => matchLabel(event.Actor.Attributes ?? {}, v));
                break;
            default:
                matches = true;
        }
        if (!matches) return false;
    }
    return true;
}
