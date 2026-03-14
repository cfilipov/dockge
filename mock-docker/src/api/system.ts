import type { Route } from "../server.js";
import { sendJSON, sendPlain, sendNoContent, sendError } from "../server.js";
import { parseFilters, applyEventFilters } from "../filters.js";
import type { DockerEvent } from "../list-types.js";

const PING_HEADERS = {
    "API-Version": "1.47",
    "Docker-Experimental": "false",
};

export const systemRoutes: Route[] = [
    {
        method: "GET",
        pattern: "/_ping",
        handler: async ({ res }) => {
            for (const [k, v] of Object.entries(PING_HEADERS)) {
                res.setHeader(k, v);
            }
            sendPlain(res, 200, "OK");
        },
    },
    {
        method: "HEAD",
        pattern: "/_ping",
        handler: async ({ res }) => {
            for (const [k, v] of Object.entries(PING_HEADERS)) {
                res.setHeader(k, v);
            }
            sendNoContent(res, 200);
        },
    },
    {
        method: "GET",
        pattern: "/version",
        handler: async ({ res }) => {
            sendJSON(res, 200, {
                Version: "27.5.1",
                ApiVersion: "1.47",
                MinAPIVersion: "1.24",
                GitCommit: "mock",
                GoVersion: "go1.22.0",
                Os: "linux",
                Arch: "amd64",
                KernelVersion: "6.1.0-mock",
                BuildTime: "2025-01-15T00:00:00.000000000+00:00",
            });
        },
    },
    {
        method: "GET",
        pattern: "/info",
        handler: async ({ res, state }) => {
            let running = 0;
            let paused = 0;
            let stopped = 0;
            for (const c of state.containers.values()) {
                if (c.State.Running) {
                    if (c.State.Paused) paused++;
                    else running++;
                } else {
                    stopped++;
                }
            }
            sendJSON(res, 200, {
                Containers: state.containers.size,
                ContainersRunning: running,
                ContainersPaused: paused,
                ContainersStopped: stopped,
                Images: state.images.size,
                Driver: "overlay2",
                DockerRootDir: "/var/lib/docker",
                Name: "mock-docker",
                ServerVersion: "27.5.1",
                OperatingSystem: "Mock Docker Engine",
                OSType: "linux",
                Architecture: "x86_64",
            });
        },
    },
    {
        method: "GET",
        pattern: "/events",
        handler: async ({ req, res, query, emitter, clock }) => {
            const filters = parseFilters(query.filters);
            const since = query.since ? parseFloat(query.since) : undefined;
            const until = query.until ? parseFloat(query.until) : undefined;

            res.writeHead(200, {
                "Content-Type": "application/json",
                "Transfer-Encoding": "chunked",
            });

            // If until is in the past, close immediately
            if (until !== undefined && until <= clock.now().getTime() / 1000) {
                res.end();
                return;
            }

            const listener = (event: DockerEvent) => {
                // Check since/until
                if (since !== undefined && event.time < since) return;
                if (until !== undefined && event.time > until) return;

                if (applyEventFilters(event, filters)) {
                    res.write(JSON.stringify(event) + "\n");
                }
            };

            emitter.subscribe(listener);

            // If until is specified, set a timeout to close
            let untilTimer: ReturnType<typeof setTimeout> | undefined;
            if (until !== undefined) {
                const delayMs = Math.max(0, until * 1000 - clock.now().getTime());
                untilTimer = setTimeout(() => {
                    emitter.unsubscribe(listener);
                    res.end();
                }, delayMs);
            }

            req.on("close", () => {
                emitter.unsubscribe(listener);
                if (untilTimer) clearTimeout(untilTimer);
            });
        },
    },
];
