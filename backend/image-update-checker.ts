import { DockgeServer } from "./dockge-server";
import { R } from "redbean-node";
import { Settings } from "./settings";
import { Stack } from "./stack";
import { log } from "./log";
import yaml from "yaml";
import childProcessAsync from "promisify-child-process";
import { LABEL_IMAGEUPDATES_CHECK, LABEL_IMAGEUPDATES_IGNORE } from "../common/compose-labels";

const MODULE = "image-update-checker";

// Default interval: 6 hours
const DEFAULT_CHECK_INTERVAL_HOURS = 6;
// Delay before first check after startup: 5 minutes
const INITIAL_DELAY_MS = 5 * 60 * 1000;
// Request timeout for registry calls
const REGISTRY_TIMEOUT_MS = 15_000;
// Max concurrent image checks
const CONCURRENCY_LIMIT = 3;

export interface ParsedImageRef {
    registry: string;
    repository: string;
    tag: string;
}

/**
 * Parse a Docker image reference into its components.
 * Examples:
 *   "nginx" → { registry: "registry-1.docker.io", repository: "library/nginx", tag: "latest" }
 *   "nginx:1.25" → { registry: "registry-1.docker.io", repository: "library/nginx", tag: "1.25" }
 *   "ghcr.io/foo/bar:v1" → { registry: "ghcr.io", repository: "foo/bar", tag: "v1" }
 *   "myregistry.com:5000/myimage:tag" → { registry: "myregistry.com:5000", repository: "myimage", tag: "tag" }
 */
export function parseImageReference(imageRef: string): ParsedImageRef {
    let ref = imageRef.trim();

    // Handle @sha256: digest pinned images — skip these, no tag to check
    if (ref.includes("@sha256:")) {
        const parts = ref.split("@sha256:");
        ref = parts[0]; // strip digest, treat as tagless
    }

    let registry = "registry-1.docker.io";
    let repository: string;
    let tag = "latest";

    // Split tag from repository
    // Be careful: registry:port/repo:tag has colons in both
    const lastColon = ref.lastIndexOf(":");
    const lastSlash = ref.lastIndexOf("/");

    if (lastColon > lastSlash && lastColon !== -1) {
        tag = ref.substring(lastColon + 1);
        ref = ref.substring(0, lastColon);
    }

    // Determine if the first component is a registry
    const parts = ref.split("/");
    if (parts.length >= 2 && (parts[0].includes(".") || parts[0].includes(":"))) {
        // First part looks like a hostname
        registry = parts[0];
        repository = parts.slice(1).join("/");
    } else if (parts.length === 1) {
        // Short name like "nginx" → Docker Hub library image
        repository = "library/" + parts[0];
    } else {
        // e.g. "louislam/uptime-kuma" → Docker Hub user image
        repository = ref;
    }

    return { registry, repository, tag };
}

/**
 * Fetch the remote digest for an image from a container registry.
 * Uses the Docker Registry HTTP API V2.
 * Returns null on any failure.
 */
async function fetchRemoteDigest(parsed: ParsedImageRef): Promise<string | null> {
    const { registry, repository, tag } = parsed;

    const registryUrl = registry === "registry-1.docker.io"
        ? "https://registry-1.docker.io"
        : `https://${registry}`;

    const manifestUrl = `${registryUrl}/v2/${repository}/manifests/${tag}`;
    const acceptHeaders = [
        "application/vnd.docker.distribution.manifest.v2+json",
        "application/vnd.docker.distribution.manifest.list.v2+json",
        "application/vnd.oci.image.manifest.v1+json",
        "application/vnd.oci.image.index.v1+json",
    ].join(", ");

    try {
        // First attempt — may get 401
        let res = await fetch(manifestUrl, {
            method: "HEAD",
            headers: { "Accept": acceptHeaders },
            signal: AbortSignal.timeout(REGISTRY_TIMEOUT_MS),
        });

        // Handle auth challenge
        if (res.status === 401) {
            const wwwAuth = res.headers.get("www-authenticate") || "";
            const token = await fetchBearerToken(wwwAuth, repository);
            if (!token) {
                return null;
            }

            res = await fetch(manifestUrl, {
                method: "HEAD",
                headers: {
                    "Accept": acceptHeaders,
                    "Authorization": `Bearer ${token}`,
                },
                signal: AbortSignal.timeout(REGISTRY_TIMEOUT_MS),
            });
        }

        if (!res.ok) {
            log.debug(MODULE, `Registry returned ${res.status} for ${repository}:${tag}`);
            return null;
        }

        return res.headers.get("docker-content-digest");
    } catch (e) {
        log.debug(MODULE, `Failed to fetch remote digest for ${repository}:${tag}: ${e}`);
        return null;
    }
}

/**
 * Parse a Www-Authenticate header and fetch a bearer token.
 */
async function fetchBearerToken(wwwAuth: string, repository: string): Promise<string | null> {
    // Parse: Bearer realm="https://auth.docker.io/token",service="registry.docker.io",scope="repository:library/nginx:pull"
    const realmMatch = wwwAuth.match(/realm="([^"]+)"/);
    const serviceMatch = wwwAuth.match(/service="([^"]+)"/);

    if (!realmMatch) {
        return null;
    }

    const realm = realmMatch[1];
    const service = serviceMatch ? serviceMatch[1] : "";
    const scope = `repository:${repository}:pull`;

    const tokenUrl = `${realm}?service=${encodeURIComponent(service)}&scope=${encodeURIComponent(scope)}`;

    try {
        const res = await fetch(tokenUrl, {
            signal: AbortSignal.timeout(REGISTRY_TIMEOUT_MS),
        });
        if (!res.ok) {
            return null;
        }
        const data = await res.json() as { token?: string; access_token?: string };
        return data.token || data.access_token || null;
    } catch {
        return null;
    }
}

/**
 * Get the local digest for an image via `docker image inspect`.
 * Returns null if the image is not found locally.
 */
async function fetchLocalDigest(imageRef: string): Promise<string | null> {
    try {
        const res = await childProcessAsync.spawn("docker", ["image", "inspect", "--format", "json", imageRef], {
            encoding: "utf-8",
            timeout: 10_000,
        });

        if (!res.stdout) {
            return null;
        }

        const data = JSON.parse(res.stdout.toString());
        // docker image inspect --format json returns an array
        const inspectArr = Array.isArray(data) ? data : [data];
        if (inspectArr.length === 0) {
            return null;
        }

        const repoDigests = inspectArr[0].RepoDigests;
        if (Array.isArray(repoDigests) && repoDigests.length > 0) {
            // Format: "nginx@sha256:abc123..."
            const parts = repoDigests[0].split("@");
            return parts.length >= 2 ? parts[1] : null;
        }

        return null;
    } catch {
        return null;
    }
}

/**
 * Run async tasks with a concurrency limit.
 */
async function parallelLimit<T>(tasks: (() => Promise<T>)[], limit: number): Promise<T[]> {
    const results: T[] = [];
    let index = 0;

    async function worker() {
        while (index < tasks.length) {
            const i = index++;
            results[i] = await tasks[i]();
        }
    }

    const workers = Array.from({ length: Math.min(limit, tasks.length) }, () => worker());
    await Promise.all(workers);
    return results;
}

// Per-stack cache entry
interface StackUpdateInfo {
    hasUpdates: boolean;
    services: Map<string, boolean>;
}

export class ImageUpdateChecker {
    private server: DockgeServer;
    private cache: Map<string, StackUpdateInfo> = new Map();
    private interval?: NodeJS.Timeout;
    private initialTimeout?: NodeJS.Timeout;
    private checking = false;

    constructor(server: DockgeServer) {
        this.server = server;
    }

    /**
     * Load cached results from the database into the in-memory Map.
     * Called once on startup — a single SELECT, instant.
     */
    async loadCacheFromDB(): Promise<void> {
        try {
            const rows = await R.getAll("SELECT stack_name, service_name, has_update FROM image_update_cache");
            this.cache.clear();

            for (const row of rows) {
                const stackName = row.stack_name;
                const serviceName = row.service_name;
                const hasUpdate = !!row.has_update;

                if (!this.cache.has(stackName)) {
                    this.cache.set(stackName, { hasUpdates: false, services: new Map() });
                }
                const entry = this.cache.get(stackName)!;
                entry.services.set(serviceName, hasUpdate);
                if (hasUpdate) {
                    entry.hasUpdates = true;
                }
            }

            log.info(MODULE, `Loaded ${rows.length} cached image update entries`);
        } catch (e) {
            log.warn(MODULE, `Failed to load cache from DB: ${e}`);
        }
    }

    /**
     * Start the background checking interval.
     * - Load DB cache immediately (instant).
     * - First registry check after INITIAL_DELAY_MS.
     * - Then repeat every N hours.
     */
    async startInterval(): Promise<void> {
        await this.loadCacheFromDB();

        this.initialTimeout = setTimeout(async () => {
            await this.runCheck();
            const intervalHours = await this.getIntervalHours();
            this.interval = setInterval(() => this.runCheck(), intervalHours * 60 * 60 * 1000);
        }, INITIAL_DELAY_MS);

        log.info(MODULE, `Scheduled first check in ${INITIAL_DELAY_MS / 1000}s`);
    }

    private async getIntervalHours(): Promise<number> {
        const val = await Settings.get("imageUpdateCheckInterval");
        const hours = typeof val === "number" ? val : DEFAULT_CHECK_INTERVAL_HOURS;
        return Math.max(1, hours);
    }

    private async runCheck(): Promise<void> {
        if (this.checking) {
            log.debug(MODULE, "Check already in progress, skipping");
            return;
        }
        try {
            await this.checkAll();
        } catch (e) {
            log.error(MODULE, `checkAll failed: ${e}`);
        }
    }

    /**
     * Full check cycle: parse all stacks, check registries, write DB, rebuild cache, push to clients.
     */
    async checkAll(): Promise<void> {
        const enabled = await Settings.get("imageUpdateCheckEnabled");
        if (enabled === false) {
            log.debug(MODULE, "Image update check is disabled");
            return;
        }

        this.checking = true;
        log.info(MODULE, "Starting image update check for all stacks");

        try {
            const stackList = await Stack.getStackList(this.server, true);
            const tasks: (() => Promise<void>)[] = [];

            for (const [stackName, stack] of stackList) {
                if (!stack.isManagedByDockge) {
                    continue;
                }

                const composeYAML = stack.composeYAML;
                if (!composeYAML) {
                    continue;
                }

                let doc;
                try {
                    doc = yaml.parse(composeYAML);
                } catch {
                    log.debug(MODULE, `Failed to parse YAML for ${stackName}`);
                    continue;
                }

                if (!doc?.services || typeof doc.services !== "object") {
                    continue;
                }

                for (const [serviceName, serviceConfig] of Object.entries(doc.services)) {
                    const svc = serviceConfig as Record<string, unknown>;
                    if (!svc.image || typeof svc.image !== "string") {
                        continue;
                    }

                    // Check dockge labels
                    const labels = svc.labels as Record<string, string> | undefined;
                    if (labels && labels[LABEL_IMAGEUPDATES_CHECK] === "false") {
                        continue;
                    }

                    const ignoreDigest = labels?.[LABEL_IMAGEUPDATES_IGNORE];
                    const imageRef = svc.image as string;

                    tasks.push(() => this.checkSingleImage(stackName, serviceName, imageRef, ignoreDigest));
                }
            }

            await parallelLimit(tasks, CONCURRENCY_LIMIT);
            this.rebuildCache();
            log.info(MODULE, `Check complete: ${tasks.length} images checked`);

            // Push updated flags to all clients
            this.server.sendStackList();
        } finally {
            this.checking = false;
        }
    }

    /**
     * Check a single stack (for manual "Check Now" button).
     */
    async checkStack(stackName: string): Promise<void> {
        log.info(MODULE, `Checking stack: ${stackName}`);

        const stack = await Stack.getStack(this.server, stackName);
        if (!stack.isManagedByDockge) {
            return;
        }

        const composeYAML = stack.composeYAML;
        if (!composeYAML) {
            return;
        }

        let doc;
        try {
            doc = yaml.parse(composeYAML);
        } catch {
            return;
        }

        if (!doc?.services || typeof doc.services !== "object") {
            return;
        }

        const tasks: (() => Promise<void>)[] = [];
        for (const [serviceName, serviceConfig] of Object.entries(doc.services)) {
            const svc = serviceConfig as Record<string, unknown>;
            if (!svc.image || typeof svc.image !== "string") {
                continue;
            }

            const labels = svc.labels as Record<string, string> | undefined;
            if (labels && labels[LABEL_IMAGEUPDATES_CHECK] === "false") {
                continue;
            }

            const ignoreDigest = labels?.[LABEL_IMAGEUPDATES_IGNORE];
            const imageRef = svc.image as string;

            tasks.push(() => this.checkSingleImage(stackName, serviceName, imageRef, ignoreDigest));
        }

        await parallelLimit(tasks, CONCURRENCY_LIMIT);
        this.rebuildCache();

        // Push to all clients
        this.server.sendStackList();
    }

    /**
     * Check a single image: fetch remote + local digests, compare, write to DB.
     */
    private async checkSingleImage(
        stackName: string,
        serviceName: string,
        imageRef: string,
        ignoreDigest?: string
    ): Promise<void> {
        try {
            const parsed = parseImageReference(imageRef);
            const [remoteDigest, localDigest] = await Promise.all([
                fetchRemoteDigest(parsed),
                fetchLocalDigest(imageRef),
            ]);

            let hasUpdate = false;
            if (remoteDigest && localDigest) {
                hasUpdate = remoteDigest !== localDigest;

                // Check if this digest should be ignored
                if (hasUpdate && ignoreDigest && remoteDigest === ignoreDigest) {
                    hasUpdate = false;
                }
            }

            const now = Math.floor(Date.now() / 1000);

            // Upsert into DB
            const existing = await R.getRow(
                "SELECT id FROM image_update_cache WHERE stack_name = ? AND service_name = ?",
                [stackName, serviceName]
            );

            if (existing) {
                await R.exec(
                    `UPDATE image_update_cache SET image_reference = ?, local_digest = ?, remote_digest = ?, has_update = ?, last_checked = ? WHERE id = ?`,
                    [imageRef, localDigest, remoteDigest, hasUpdate ? 1 : 0, now, existing.id]
                );
            } else {
                await R.exec(
                    `INSERT INTO image_update_cache (stack_name, service_name, image_reference, local_digest, remote_digest, has_update, last_checked) VALUES (?, ?, ?, ?, ?, ?, ?)`,
                    [stackName, serviceName, imageRef, localDigest, remoteDigest, hasUpdate ? 1 : 0, now]
                );
            }

            log.debug(MODULE, `${stackName}/${serviceName} (${imageRef}): update=${hasUpdate}`);
        } catch (e) {
            log.warn(MODULE, `Error checking ${stackName}/${serviceName}: ${e}`);
        }
    }

    /**
     * Rebuild the in-memory cache from the DB.
     */
    private async rebuildCache(): Promise<void> {
        await this.loadCacheFromDB();
    }

    /**
     * Get whether a stack has any image updates available.
     * Pure Map.get() — zero I/O.
     */
    getStackHasUpdates(stackName: string): boolean {
        return this.cache.get(stackName)?.hasUpdates ?? false;
    }

    /**
     * Get per-service update status for a stack.
     * Returns a plain object { serviceName: boolean }.
     */
    getServiceUpdateMap(stackName: string): Record<string, boolean> {
        const entry = this.cache.get(stackName);
        if (!entry) {
            return {};
        }
        return Object.fromEntries(entry.services);
    }

    /**
     * Stop the checker, clean up timers.
     */
    stop(): void {
        if (this.initialTimeout) {
            clearTimeout(this.initialTimeout);
            this.initialTimeout = undefined;
        }
        if (this.interval) {
            clearInterval(this.interval);
            this.interval = undefined;
        }
        log.info(MODULE, "Stopped");
    }
}
