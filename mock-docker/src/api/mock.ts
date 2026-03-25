import { readdirSync, rmSync } from "node:fs";
import { join } from "node:path";
import type { Route } from "../server.js";
import { sendJSON, sendError } from "../server.js";
import { initState } from "../init.js";
import { FixedClock } from "../clock.js";

export const mockRoutes: Route[] = [
    {
        method: "POST",
        pattern: "/_mock/reset",
        handler: async ({ res, state, clock, initOpts }) => {
            try {
                // Step 1: Clear stacks dir contents
                const entries = readdirSync(initOpts.stacksDir);
                for (const entry of entries) {
                    rmSync(join(initOpts.stacksDir, entry), { recursive: true, force: true });
                }

                // Step 1b: Reset clock tick counter for deterministic timestamps
                if (clock instanceof FixedClock) {
                    clock.resetTick();
                }

                // Step 2: Re-initialize state
                const fresh = await initState(initOpts);

                // Step 3: Copy maps in-place
                state.containers = fresh.containers;
                state.networks = fresh.networks;
                state.volumes = fresh.volumes;
                state.images = fresh.images;
                state.execSessions = fresh.execSessions;
                state.logTemplates = fresh.logTemplates;
                state.logBuffers = fresh.logBuffers;
                // Clear heartbeat intervals from old state
                for (const interval of state.heartbeatIntervals.values()) {
                    clearInterval(interval);
                }
                state.heartbeatIntervals = fresh.heartbeatIntervals;

                sendJSON(res, 200, { status: "ok" });
            } catch (err) {
                const message = err instanceof Error ? err.message : "reset failed";
                sendError(res, 500, message);
            }
        },
    },
];
