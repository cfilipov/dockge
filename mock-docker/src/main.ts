import { parseArgs } from "node:util";
import { unlinkSync } from "node:fs";
import { createClock } from "./clock.js";
import { initState } from "./init.js";
import type { InitOptions } from "./init.js";
import { EventEmitter } from "./events.js";
import { createServer } from "./server.js";
import type { Route } from "./server.js";

import { systemRoutes } from "./api/system.js";
import { containerRoutes } from "./api/containers.js";
import { networkRoutes } from "./api/networks.js";
import { volumeRoutes } from "./api/volumes.js";
import { imageRoutes } from "./api/images.js";
import { distributionRoutes } from "./api/distribution.js";
import { execRoutes } from "./api/exec.js";
import { mockRoutes } from "./api/mock.js";

// ---------------------------------------------------------------------------
// CLI args
// ---------------------------------------------------------------------------

const { values } = parseArgs({
    options: {
        socket: { type: "string" },
        "stacks-dir": { type: "string" },
        "stacks-source": { type: "string" },
        "e2e": { type: "boolean", default: false },
        "clock-base": { type: "string", default: "2025-01-15T00:00:00Z" },
        "images-json": { type: "string" },
        "log-interval": { type: "string", default: "5000" },
        "stats-interval": { type: "string", default: "1000" },
    },
    strict: true,
});

const socketPath = values.socket;
const stacksDir = values["stacks-dir"];
const stacksSource = values["stacks-source"];

if (!socketPath || !stacksDir || !stacksSource) {
    console.error("Usage: node main.js --socket <path> --stacks-dir <path> --stacks-source <path>");
    process.exit(1);
}

// ---------------------------------------------------------------------------
// Startup
// ---------------------------------------------------------------------------

const clock = createClock({
    base: values["clock-base"],
});

const initOpts: InitOptions = { stacksSource, stacksDir, clock, imagesJsonPath: values["images-json"], e2eMode: values["e2e"] };
const state = await initState(initOpts);
const emitter = new EventEmitter();
emitter.bind(state);

// Collect all routes (order matters — most-specific first)
const routes: Route[] = [
    ...systemRoutes,
    ...mockRoutes,
    ...containerRoutes,
    ...imageRoutes,
    ...distributionRoutes,
    ...networkRoutes,
    ...volumeRoutes,
    ...execRoutes,
];

// Clean up stale socket file
try { unlinkSync(socketPath); } catch { /* ignore */ }

const e2eMode = values["e2e"] || false;
const logInterval = parseInt(values["log-interval"] || "5000", 10);
const statsInterval = parseInt(values["stats-interval"] || "1000", 10);

const { start, stop } = createServer(
    { socketPath, state, emitter, clock, initOpts, e2eMode, logInterval, statsInterval },
    routes,
);

await start();
console.log(`Mock Docker daemon listening on ${socketPath}`);
console.log(`  Stacks dir: ${stacksDir}`);
console.log(`  Stacks source: ${stacksSource}`);
console.log(`  Images JSON: ${values["images-json"] || "(auto-discover)"}`);
console.log(`  Containers: ${state.containers.size}`);
console.log(`  Networks: ${state.networks.size}`);
console.log(`  Images: ${state.images.size}`);

// ---------------------------------------------------------------------------
// Graceful shutdown
// ---------------------------------------------------------------------------

const shutdown = async () => {
    console.log("\nShutting down...");
    await stop();
    try { unlinkSync(socketPath); } catch { /* ignore */ }
    process.exit(0);
};

process.on("SIGINT", shutdown);
process.on("SIGTERM", shutdown);
