/**
 * One-time generator for .mock.yaml sidecar files.
 *
 * Usage:
 *   npx tsx scripts/generate-mock-configs.ts --stacks-dir ../stacks
 *
 * Produces:
 *   {stacks-dir}/.mock.yaml          — global networks, volumes, standalone containers
 *   {stacks-dir}/{stack}/.mock.yaml   — per-stack service overrides (only for stacks that need them)
 *
 * All randomness is deterministic (seeded by stack/service name) so re-running
 * produces identical output.
 */

import { parseArgs } from "node:util";
import { readdirSync, readFileSync, statSync, writeFileSync, renameSync, existsSync } from "node:fs";
import { join } from "node:path";
import { createHash } from "node:crypto";
import { parse as yamlParse, stringify as yamlStringify } from "yaml";

// ---------------------------------------------------------------------------
// CLI
// ---------------------------------------------------------------------------

const { values } = parseArgs({
    options: {
        "stacks-dir": { type: "string" },
    },
    strict: true,
});

const stacksDir = values["stacks-dir"];
if (!stacksDir) {
    console.error("Usage: npx tsx scripts/generate-mock-configs.ts --stacks-dir <path>");
    process.exit(1);
}

// ---------------------------------------------------------------------------
// Deterministic helpers (mirrors deterministic.ts but standalone)
// ---------------------------------------------------------------------------

const GEN_SEED = "mock-config-gen-v1";

function seed(parts: string[]): string {
    return createHash("sha256").update(parts.join("\0")).digest("hex");
}

/** Returns a float in [0, 1) deterministically. */
function deterministicFloat(s: string): number {
    const hash = createHash("sha256").update(s).digest();
    return hash.readUInt32BE(0) / 0x100000000;
}

/** Returns true with probability p (deterministic). */
function chance(s: string, p: number): boolean {
    return deterministicFloat(s) < p;
}

/** Pick one item from an array deterministically. */
function pick<T>(s: string, items: T[]): T {
    const hash = createHash("sha256").update(s).digest();
    return items[hash.readUInt32BE(0) % items.length];
}

// ---------------------------------------------------------------------------
// Compose file scanning
// ---------------------------------------------------------------------------

function findComposeFile(dir: string): string | null {
    for (const name of ["compose.yaml", "compose.yml", "docker-compose.yaml", "docker-compose.yml"]) {
        const p = join(dir, name);
        if (existsSync(p)) return p;
    }
    return null;
}

interface ServiceInfo {
    name: string;
    hasHealthcheck: boolean;
}

function getServices(composeContent: string): ServiceInfo[] {
    const doc = yamlParse(composeContent);
    if (!doc || !doc.services) return [];

    return Object.entries(doc.services).map(([name, svc]: [string, any]) => ({
        name,
        hasHealthcheck: !!(svc?.healthcheck && !svc.healthcheck?.disable),
    }));
}

// ---------------------------------------------------------------------------
// Global config generation
// ---------------------------------------------------------------------------

function generateGlobalConfig(): Record<string, unknown> {
    return {
        networks: {
            "proxy_default": { driver: "bridge" },
            "monitoring_default": { driver: "bridge" },
            "database_shared": { driver: "bridge", internal: true },
        },
        volumes: {
            "shared_certs": { driver: "local" },
            "backup_staging": { driver: "local" },
        },
        containers: [
            {
                name: "nginx-proxy",
                image: "nginx:alpine",
                state: "running",
                ports: ["8080:80/tcp"],
                networks: ["proxy_default"],
                labels: { "maintainer": "ops-team" },
            },
            {
                name: "redis-cache",
                image: "redis:7-alpine",
                state: "running",
                ports: ["6379:6379/tcp"],
            },
            {
                name: "postgres-shared",
                image: "postgres:16-alpine",
                state: "running",
                ports: ["5432:5432/tcp"],
                networks: ["database_shared"],
                volumes: ["pgdata:/var/lib/postgresql/data"],
                environment: ["POSTGRES_PASSWORD=devpass", "POSTGRES_DB=shared"],
            },
            {
                name: "alpine-cron",
                image: "alpine:3.19",
                state: "running",
                command: "crond -f",
            },
            {
                name: "dev-mailhog",
                image: "mailhog/mailhog:latest",
                state: "running",
                ports: ["1025:1025/tcp", "8025:8025/tcp"],
            },
            {
                name: "portainer-agent",
                image: "portainer/agent:2.19.4",
                state: "running",
                ports: ["9001:9001/tcp"],
            },
            {
                name: "watchtower",
                image: "containrrr/watchtower:latest",
                state: "running",
            },
            {
                name: "debug-busybox",
                image: "busybox:1.36",
                state: "exited",
                exit_code: 0,
                command: "sh -c 'echo done'",
            },
        ],
    };
}

// ---------------------------------------------------------------------------
// Per-stack config generation
// ---------------------------------------------------------------------------

interface StackOverride {
    deployed?: boolean;
    untracked?: boolean;
    services?: Record<string, Record<string, unknown>>;
}

function generateStackConfig(stackName: string, services: ServiceInfo[]): StackOverride | null {
    const baseSeed = seed([GEN_SEED, "stack", stackName]);

    // ~10% chance: not deployed
    if (chance(seed([baseSeed, "deployed"]), 0.10)) {
        return { deployed: false };
    }

    // ~5% chance: untracked (external stack)
    if (chance(seed([baseSeed, "untracked"]), 0.05)) {
        return { untracked: true };
    }

    // Generate per-service overrides
    const svcOverrides: Record<string, Record<string, unknown>> = {};
    let hasOverrides = false;

    for (const svc of services) {
        const svcSeed = seed([GEN_SEED, "service", stackName, svc.name]);
        const override: Record<string, unknown> = {};
        let changed = false;

        // ~15% chance: exited
        if (chance(seed([svcSeed, "exited"]), 0.15)) {
            override.state = "exited";
            override.exit_code = chance(seed([svcSeed, "exitcode"]), 0.3) ? 1 : 0;
            changed = true;
        }
        // ~5% chance: paused (only if not already exited)
        else if (chance(seed([svcSeed, "paused"]), 0.05)) {
            override.state = "paused";
            changed = true;
        }

        // ~1% chance: unhealthy (only makes sense for services with healthchecks,
        // but we can override health on any service)
        if (chance(seed([svcSeed, "unhealthy"]), 0.01)) {
            override.health = "unhealthy";
            changed = true;
        }
        // ~3% chance: starting
        else if (chance(seed([svcSeed, "starting"]), 0.03)) {
            override.health = "starting";
            changed = true;
        }

        // ~8% chance: update available
        if (chance(seed([svcSeed, "update"]), 0.08)) {
            override.update_available = true;
            changed = true;
        }

        // ~5% chance: needs recreation
        if (chance(seed([svcSeed, "recreation"]), 0.05)) {
            override.needs_recreation = true;
            changed = true;
        }

        if (changed) {
            svcOverrides[svc.name] = override;
            hasOverrides = true;
        }
    }

    if (!hasOverrides) return null;
    return { services: svcOverrides };
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

function main() {
    // Generate global config
    const globalConfig = generateGlobalConfig();
    const globalPath = join(stacksDir, ".mock.yaml");
    writeAtomic(globalPath, yamlStringify(globalConfig));
    console.log(`Wrote ${globalPath}`);

    // Scan stacks
    const entries = readdirSync(stacksDir).sort();
    let stackCount = 0;
    let configCount = 0;
    let notDeployed = 0;
    let untracked = 0;
    const serviceStats = { exited: 0, paused: 0, unhealthy: 0, starting: 0, update: 0, recreation: 0 };

    for (const entry of entries) {
        const subdir = join(stacksDir, entry);
        try {
            if (!statSync(subdir).isDirectory()) continue;
        } catch {
            continue;
        }

        const composePath = findComposeFile(subdir);
        if (!composePath) continue;
        stackCount++;

        let services: ServiceInfo[];
        try {
            const content = readFileSync(composePath, "utf-8");
            services = getServices(content);
        } catch {
            continue;
        }

        const config = generateStackConfig(entry, services);
        if (!config) continue;

        // Track stats
        if (config.deployed === false) notDeployed++;
        if (config.untracked === true) untracked++;
        if (config.services) {
            for (const svc of Object.values(config.services)) {
                if (svc.state === "exited") serviceStats.exited++;
                if (svc.state === "paused") serviceStats.paused++;
                if (svc.health === "unhealthy") serviceStats.unhealthy++;
                if (svc.health === "starting") serviceStats.starting++;
                if (svc.update_available) serviceStats.update++;
                if (svc.needs_recreation) serviceStats.recreation++;
            }
        }

        const mockPath = join(subdir, ".mock.yaml");
        writeAtomic(mockPath, yamlStringify(config));
        configCount++;
    }

    console.log(`\nProcessed ${stackCount} stacks, wrote ${configCount} .mock.yaml files`);
    console.log(`  Not deployed: ${notDeployed}`);
    console.log(`  Untracked: ${untracked}`);
    console.log(`  Service overrides:`);
    console.log(`    exited: ${serviceStats.exited}`);
    console.log(`    paused: ${serviceStats.paused}`);
    console.log(`    unhealthy: ${serviceStats.unhealthy}`);
    console.log(`    starting: ${serviceStats.starting}`);
    console.log(`    update_available: ${serviceStats.update}`);
    console.log(`    needs_recreation: ${serviceStats.recreation}`);
}

function writeAtomic(path: string, content: string) {
    const tmp = path + ".tmp";
    writeFileSync(tmp, content);
    renameSync(tmp, path);
}

main();
