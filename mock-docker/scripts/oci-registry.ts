/**
 * OCI registry client helpers for fetching image metadata.
 * No runtime deps beyond native fetch().
 */

import { readFileSync } from "node:fs";
import { join } from "node:path";
import { homedir } from "node:os";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface ImageRef {
    registry: string;
    name: string;
    tag: string;
    digest?: string;
}

export interface WwwAuthenticate {
    realm: string;
    service: string;
    scope: string;
}

interface FatManifestEntry {
    mediaType: string;
    digest: string;
    size: number;
    platform?: { architecture: string; os: string; variant?: string };
}

// ---------------------------------------------------------------------------
// Image ref parsing
// ---------------------------------------------------------------------------

export function parseImageRef(ref: string): ImageRef {
    let registry = "registry-1.docker.io";
    let remainder = ref;
    let digest: string | undefined;

    // Handle digest refs: image@sha256:abc...
    const atIdx = remainder.indexOf("@");
    if (atIdx !== -1) {
        digest = remainder.slice(atIdx + 1);
        remainder = remainder.slice(0, atIdx);
    }

    // Check if first component is a registry (contains . or : or is localhost)
    const firstSlash = remainder.indexOf("/");
    if (firstSlash !== -1) {
        const first = remainder.slice(0, firstSlash);
        if (first.includes(".") || first.includes(":") || first === "localhost") {
            registry = first;
            remainder = remainder.slice(firstSlash + 1);
        }
    }

    // Normalize Docker Hub aliases to canonical registry
    if (registry === "docker.io" || registry === "index.docker.io") {
        registry = "registry-1.docker.io";
    }

    // Docker Hub bare names get library/ prefix
    if (registry === "registry-1.docker.io" && !remainder.includes("/")) {
        remainder = `library/${remainder}`;
    }

    // Split name:tag
    const colonIdx = remainder.lastIndexOf(":");
    let name: string;
    let tag: string;
    if (colonIdx !== -1 && !remainder.slice(colonIdx).includes("/")) {
        name = remainder.slice(0, colonIdx);
        tag = remainder.slice(colonIdx + 1);
    } else {
        name = remainder;
        tag = "latest";
    }

    return { registry, name, tag, digest };
}

// ---------------------------------------------------------------------------
// WWW-Authenticate header parsing
// ---------------------------------------------------------------------------

export function parseWwwAuthenticate(header: string): WwwAuthenticate {
    // Format: Bearer realm="...",service="...",scope="..."
    const result: WwwAuthenticate = { realm: "", service: "", scope: "" };
    const stripped = header.replace(/^Bearer\s+/i, "");

    for (const part of stripped.split(",")) {
        const eq = part.indexOf("=");
        if (eq === -1) continue;
        const key = part.slice(0, eq).trim().toLowerCase();
        let value = part.slice(eq + 1).trim();
        if (value.startsWith('"') && value.endsWith('"')) {
            value = value.slice(1, -1);
        }
        if (key === "realm") result.realm = value;
        else if (key === "service") result.service = value;
        else if (key === "scope") result.scope = value;
    }

    return result;
}

// ---------------------------------------------------------------------------
// Docker credential lookup
// ---------------------------------------------------------------------------

const DOCKER_HUB_KEYS = [
    "registry-1.docker.io",
    "https://index.docker.io/v1/",
    "https://index.docker.io/v1/access-token",
    "index.docker.io",
];

export function getDockerAuth(registry: string): string | null {
    try {
        const configPath = join(homedir(), ".docker", "config.json");
        const config = JSON.parse(readFileSync(configPath, "utf-8"));
        const auths = config.auths || {};

        // Try exact match first
        if (auths[registry]?.auth) return auths[registry].auth;

        // Docker Hub aliases — try all known keys
        if (registry === "registry-1.docker.io" || DOCKER_HUB_KEYS.includes(registry)) {
            for (const key of DOCKER_HUB_KEYS) {
                if (auths[key]?.auth) return auths[key].auth;
            }
        }

        return null;
    } catch {
        return null;
    }
}

// ---------------------------------------------------------------------------
// Token fetch
// ---------------------------------------------------------------------------

export async function fetchToken(parsed: WwwAuthenticate, auth?: string | null): Promise<string> {
    const url = new URL(parsed.realm);
    url.searchParams.set("service", parsed.service);
    url.searchParams.set("scope", parsed.scope);

    const headers: Record<string, string> = {};
    if (auth) {
        // auth is already base64-encoded from ~/.docker/config.json — use directly
        headers["Authorization"] = `Basic ${auth}`;
    }

    const resp = await fetch(url.toString(), { headers });
    if (!resp.ok) {
        const body = await resp.text();
        throw new Error(`Token fetch failed: ${resp.status} ${resp.statusText} — ${body}`);
    }
    const body = await resp.json() as { token?: string; access_token?: string };
    return body.token || body.access_token || "";
}

// ---------------------------------------------------------------------------
// Manifest fetch
// ---------------------------------------------------------------------------

const ACCEPT_TYPES = [
    "application/vnd.oci.image.index.v1+json",
    "application/vnd.docker.distribution.manifest.list.v2+json",
    "application/vnd.oci.image.manifest.v1+json",
    "application/vnd.docker.distribution.manifest.v2+json",
].join(", ");

export async function fetchManifest(
    ref: ImageRef,
    token: string,
): Promise<{ manifest: Record<string, unknown>; digest: string; mediaType: string }> {
    const tag = ref.digest || ref.tag;
    const url = `https://${ref.registry}/v2/${ref.name}/manifests/${tag}`;

    const resp = await fetchWithRetry(url, {
        headers: {
            Accept: ACCEPT_TYPES,
            Authorization: `Bearer ${token}`,
        },
    });

    if (resp.status === 429) {
        throw new RateLimitError("Manifest fetch rate limited (429)", ref.registry);
    }
    if (!resp.ok) {
        throw new Error(`Manifest fetch failed: ${resp.status} ${resp.statusText}`);
    }

    const manifest = await resp.json() as Record<string, unknown>;
    const digest = resp.headers.get("docker-content-digest") || "";
    const mediaType = (manifest.mediaType as string) || resp.headers.get("content-type") || "";

    // If this is a fat manifest (index), find linux/amd64 and re-fetch
    if (isFatManifest(mediaType)) {
        const manifests = (manifest.manifests || []) as FatManifestEntry[];
        const amd64 = manifests.find(
            (m) => m.platform?.architecture === "amd64" && m.platform?.os === "linux",
        );
        if (!amd64) throw new Error("No linux/amd64 manifest found in index");

        return fetchManifest({ ...ref, digest: amd64.digest }, token);
    }

    return { manifest, digest, mediaType };
}

function isFatManifest(mediaType: string): boolean {
    return mediaType.includes("manifest.list") || mediaType.includes("image.index");
}

// ---------------------------------------------------------------------------
// Config blob fetch
// ---------------------------------------------------------------------------

export async function fetchConfigBlob(
    ref: ImageRef,
    token: string,
    configDigest: string,
): Promise<Record<string, unknown>> {
    const url = `https://${ref.registry}/v2/${ref.name}/blobs/${configDigest}`;

    const resp = await fetchWithRetry(url, {
        headers: {
            Authorization: `Bearer ${token}`,
        },
    });

    if (resp.status === 429) {
        throw new RateLimitError("Config blob fetch rate limited (429)", ref.registry);
    }
    if (!resp.ok) {
        throw new Error(`Config blob fetch failed: ${resp.status} ${resp.statusText}`);
    }

    return resp.json() as Promise<Record<string, unknown>>;
}

// ---------------------------------------------------------------------------
// Full pipeline: ref → ImageInspect
// ---------------------------------------------------------------------------

interface ImageConfig {
    Hostname?: string;
    Domainname?: string;
    User?: string;
    Env?: string[];
    Cmd?: string[];
    Entrypoint?: string[];
    WorkingDir?: string;
    ExposedPorts?: Record<string, Record<string, never>>;
    Volumes?: Record<string, Record<string, never>>;
    Labels?: Record<string, string>;
    StopSignal?: string;
    Shell?: string[];
    ArgsEscaped?: boolean;
    OnBuild?: string[];
}

interface OCIConfig {
    architecture?: string;
    os?: string;
    os_version?: string;
    variant?: string;
    config?: ImageConfig;
    created?: string;
    author?: string;
    rootfs?: { type: string; diff_ids?: string[] };
    history?: unknown[];
}

export interface CapturedImageInspect {
    Id: string;
    RepoTags: string[];
    RepoDigests: string[];
    Created: string;
    Architecture: string;
    Variant?: string;
    Os: string;
    OsVersion?: string;
    Author?: string;
    Size: number;
    Config: ImageConfig;
    RootFS: {
        Type: string;
        Layers: string[];
    };
}

export class RateLimitError extends Error {
    registry: string;
    constructor(message: string, registry: string) {
        super(message);
        this.name = "RateLimitError";
        this.registry = registry;
    }
}

export async function fetchImageInspect(ref: string): Promise<CapturedImageInspect | null> {
    try {
        const parsed = parseImageRef(ref);

        // Step 1: Get auth challenge
        const probeUrl = `https://${parsed.registry}/v2/${parsed.name}/manifests/${parsed.tag}`;
        const probeResp = await fetch(probeUrl, { method: "HEAD" });

        let token = "";
        if (probeResp.status === 401) {
            const wwwAuth = probeResp.headers.get("www-authenticate");
            if (!wwwAuth) throw new Error("No WWW-Authenticate header on 401");

            const authParams = parseWwwAuthenticate(wwwAuth);
            const creds = getDockerAuth(parsed.registry);
            token = await fetchToken(authParams, creds);
        } else if (!probeResp.ok) {
            throw new Error(`Probe failed: ${probeResp.status}`);
        }

        // Step 2: Fetch manifest
        const { manifest, digest } = await fetchManifest(parsed, token);

        // Step 3: Fetch config blob
        const configDesc = manifest.config as { digest: string; size: number } | undefined;
        if (!configDesc?.digest) throw new Error("No config descriptor in manifest");

        const ociConfig = await fetchConfigBlob(parsed, token, configDesc.digest) as OCIConfig;

        // Step 4: Build ImageInspect
        const normalizedRef = normalizeRef(ref);
        const repoName = normalizedRef.split(":")[0];

        const imgConfig = ociConfig.config || {};
        const rootfs = ociConfig.rootfs || { type: "layers", diff_ids: [] };

        return {
            Id: configDesc.digest.startsWith("sha256:") ? configDesc.digest : `sha256:${configDesc.digest}`,
            RepoTags: [normalizedRef],
            RepoDigests: digest ? [`${repoName}@${digest}`] : [],
            Created: ociConfig.created || new Date().toISOString(),
            Architecture: ociConfig.architecture || "amd64",
            Variant: ociConfig.variant,
            Os: ociConfig.os || "linux",
            OsVersion: ociConfig.os_version,
            Author: ociConfig.author,
            Size: configDesc.size || 0,
            Config: {
                Hostname: imgConfig.Hostname,
                Domainname: imgConfig.Domainname,
                User: imgConfig.User,
                Env: imgConfig.Env,
                Cmd: imgConfig.Cmd,
                Entrypoint: imgConfig.Entrypoint,
                WorkingDir: imgConfig.WorkingDir,
                ExposedPorts: imgConfig.ExposedPorts,
                Volumes: imgConfig.Volumes,
                Labels: imgConfig.Labels,
                StopSignal: imgConfig.StopSignal,
                Shell: imgConfig.Shell,
                ArgsEscaped: imgConfig.ArgsEscaped,
                OnBuild: imgConfig.OnBuild,
            },
            RootFS: {
                Type: rootfs.type || "layers",
                Layers: rootfs.diff_ids || [],
            },
        };
    } catch (err) {
        if (err instanceof RateLimitError) throw err; // let caller handle
        console.warn(`[oci-registry] Failed to fetch ${ref}:`, (err as Error).message);
        return null;
    }
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function normalizeRef(ref: string): string {
    // Strip registry prefix for Docker Hub images
    let r = ref;
    if (r.startsWith("docker.io/library/")) r = r.slice("docker.io/library/".length);
    else if (r.startsWith("registry-1.docker.io/library/")) r = r.slice("registry-1.docker.io/library/".length);
    else if (r.startsWith("docker.io/")) r = r.slice("docker.io/".length);

    // Remove @digest suffix
    const atIdx = r.indexOf("@");
    if (atIdx !== -1) r = r.slice(0, atIdx);

    // Ensure tag
    if (!r.includes(":")) r += ":latest";
    return r;
}

export function sleep(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
}

// ---------------------------------------------------------------------------
// Fetch with retry (rate limit handling)
// ---------------------------------------------------------------------------

async function fetchWithRetry(url: string, init: RequestInit): Promise<Response> {
    // No retries — just return the response (including 429).
    // The caller (capture script) tracks consecutive 429s and exits early.
    return fetch(url, init);
}
