#!/usr/bin/env node
/**
 * One-time capture script: fetches OCI image configs from registries
 * and writes images.json for use by mock-docker.
 *
 * Usage: npx tsx scripts/capture-images.ts --stacks-dir ./test-data/stacks
 */

import { parseArgs } from "node:util";
import { readFileSync, readdirSync, statSync, writeFileSync, renameSync, existsSync } from "node:fs";
import { join, resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { findComposeFile, parseCompose } from "../src/compose-parser.js";
import { fetchImageInspect, sleep, getDockerAuth, parseImageRef, RateLimitError, type CapturedImageInspect } from "./oci-registry.js";

// ---------------------------------------------------------------------------
// CLI args
// ---------------------------------------------------------------------------

const { values } = parseArgs({
    options: {
        "stacks-dir": { type: "string" },
        output: { type: "string" },
    },
    strict: true,
});

const stacksDir = values["stacks-dir"];
if (!stacksDir) {
    console.error("Usage: npx tsx scripts/capture-images.ts --stacks-dir <path> [--output <path>]");
    process.exit(1);
}

const scriptDir = dirname(fileURLToPath(import.meta.url));
const outputPath = values.output || join(scriptDir, "images.json");

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Returns true if the ref contains unresolved env vars like ${FOO} or $BAR */
function hasUnresolvedVars(ref: string): boolean {
    return /\$\{/.test(ref) || /\$[A-Z_]/.test(ref);
}

/** Returns true if the ref resolves to Docker Hub (registry-1.docker.io) */
function isDockerHub(ref: string): boolean {
    const parsed = parseImageRef(ref);
    return parsed.registry === "registry-1.docker.io";
}

// ---------------------------------------------------------------------------
// Scan stacks for image refs
// ---------------------------------------------------------------------------

function scanImageRefs(dir: string): Set<string> {
    const refs = new Set<string>();
    let entries: string[];
    try {
        entries = readdirSync(dir);
    } catch {
        console.error(`Cannot read stacks dir: ${dir}`);
        process.exit(1);
    }

    for (const entry of entries) {
        const subdir = join(dir, entry);
        try {
            if (!statSync(subdir).isDirectory()) continue;
        } catch {
            continue;
        }

        const composeFile = findComposeFile(subdir);
        if (!composeFile) continue;

        const content = readFileSync(composeFile, "utf-8");
        const parsed = parseCompose(content);

        for (const [, svc] of Object.entries(parsed.services)) {
            if (svc.image) {
                const ref = svc.image.includes(":") ? svc.image : svc.image + ":latest";
                refs.add(ref);
            }
        }
    }

    return refs;
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

async function main() {
    const resolvedStacksDir = resolve(stacksDir!);
    console.log(`Scanning stacks in: ${resolvedStacksDir}`);
    console.log(`Output: ${outputPath}`);

    // Collect unique image refs
    const allRefs = scanImageRefs(resolvedStacksDir);
    console.log(`Found ${allRefs.size} unique image refs`);

    // Filter out unfetchable refs (unresolved env vars)
    const envVarRefs: string[] = [];
    const fetchableRefs: string[] = [];
    for (const ref of allRefs) {
        if (hasUnresolvedVars(ref)) {
            envVarRefs.push(ref);
        } else {
            fetchableRefs.push(ref);
        }
    }
    if (envVarRefs.length > 0) {
        console.log(`Filtered out ${envVarRefs.length} refs with unresolved env vars`);
    }

    // Registry breakdown
    const registryCounts = new Map<string, number>();
    for (const ref of fetchableRefs) {
        const registry = parseImageRef(ref).registry;
        registryCounts.set(registry, (registryCounts.get(registry) || 0) + 1);
    }
    console.log("\nRegistry breakdown:");
    for (const [registry, count] of [...registryCounts.entries()].sort((a, b) => b[1] - a[1])) {
        console.log(`  ${registry}: ${count} images`);
    }
    console.log();

    if (fetchableRefs.length === 0) {
        console.log("No images to capture.");
        return;
    }

    // Load existing images.json for resume support
    let existing: Record<string, CapturedImageInspect> = {};
    if (existsSync(outputPath)) {
        try {
            existing = JSON.parse(readFileSync(outputPath, "utf-8"));
            console.log(`Loaded ${Object.keys(existing).length} already-captured images from ${outputPath}`);
        } catch {
            console.warn("Failed to parse existing images.json, starting fresh");
        }
    }

    // Check auth
    const dockerHubAuth = getDockerAuth("registry-1.docker.io");
    if (dockerHubAuth) {
        try {
            const decoded = Buffer.from(dockerHubAuth, "base64").toString();
            const username = decoded.split(":")[0];
            console.log(`Docker Hub auth: credentials found (user: ${username})`);
        } catch {
            console.log("Docker Hub auth: credentials found");
        }
    } else {
        console.log("Docker Hub auth: anonymous (no credentials found — may get rate limited)");
    }

    // Sort: non-Docker-Hub images first, Docker Hub last
    // This maximizes progress since non-Docker-Hub registries have separate rate limits
    const pending = fetchableRefs.filter((ref) => !existing[ref]);
    pending.sort((a, b) => {
        const aHub = isDockerHub(a) ? 1 : 0;
        const bHub = isDockerHub(b) ? 1 : 0;
        return aHub - bHub;
    });

    const cached = fetchableRefs.length - pending.length;
    if (cached > 0) {
        console.log(`Skipping ${cached} already-cached images`);
    }

    const nonHubPending = pending.filter((r) => !isDockerHub(r)).length;
    const hubPending = pending.filter((r) => isDockerHub(r)).length;
    console.log(`To fetch: ${nonHubPending} non-Docker-Hub, ${hubPending} Docker Hub\n`);

    // Capture loop
    let captured = 0;
    let skipped = 0;
    let consecutive429 = 0; // Only counts Docker Hub 429s
    const RATE_LIMIT_WAIT_MS = 6.5 * 60 * 60 * 1000; // 6.5 hours (6h window + 30min wiggle)

    let i = 0;
    while (i < pending.length) {
        const ref = pending[i];
        const hub = isDockerHub(ref);

        console.log(`  [fetch] ${ref}...`);
        let result: CapturedImageInspect | null = null;
        let rateLimited = false;

        try {
            result = await fetchImageInspect(ref);
        } catch (err) {
            if (err instanceof RateLimitError) {
                rateLimited = true;
            } else {
                throw err;
            }
        }

        if (rateLimited) {
            if (hub) {
                consecutive429++;
                console.log(`    -> Docker Hub rate limited (${consecutive429} consecutive)`);

                if (consecutive429 >= 3) {
                    const total = Object.keys(existing).length;
                    const remaining = pending.length - i;
                    console.log(`\nRate limit reached. ${captured} captured this run, ${total} total. ${remaining} remaining.`);
                    console.log(`Waiting 6.5 hours until rate limit resets... (Ctrl+C is safe — progress is saved)`);
                    await sleep(RATE_LIMIT_WAIT_MS);
                    consecutive429 = 0;
                    // Retry from the first rate-limited image (back up 2)
                    i = Math.max(0, i - 2);
                } else {
                    i++;
                }
            } else {
                // Non-Docker-Hub rate limit — just skip, don't count toward Docker Hub counter
                console.log(`    -> skipped (rate limited by ${parseImageRef(ref).registry})`);
                skipped++;
                i++;
            }
            continue;
        }

        // Any non-429 result resets the Docker Hub consecutive counter
        if (hub) {
            consecutive429 = 0;
        }

        if (result) {
            existing[ref] = result;
            captured++;

            // Atomic write: .tmp + rename
            const tmpPath = outputPath + ".tmp";
            writeFileSync(tmpPath, JSON.stringify(existing, null, 2) + "\n");
            renameSync(tmpPath, outputPath);
            console.log(`    -> captured (${result.Architecture}, ${result.RootFS.Layers.length} layers)`);
        } else {
            skipped++;
            console.log(`    -> skipped (failed)`);
        }

        i++;
        // Rate limit courtesy
        await sleep(500);
    }

    console.log(`\nDone: ${captured} captured, ${skipped} skipped, ${cached} already cached, ${envVarRefs.length} env-var refs filtered`);
    console.log(`Total images in ${outputPath}: ${Object.keys(existing).length}`);
}

main().catch((err) => {
    console.error("Fatal error:", err);
    process.exit(1);
});
