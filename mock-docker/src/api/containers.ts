import type { Route } from "../server.js";
import { sendJSON, sendError, sendNoContent, readJSON, handleMutationResult } from "../server.js";
import {
    containerStart,
    containerStop,
    containerRestart,
    containerPause,
    containerUnpause,
    containerRemove,
    containerCreate,
    containerRename,
    containerKill,
    execCreate,
} from "../mutations.js";
import type { ContainerCreateConfig } from "../mutations.js";
import { parseFilters, applyContainerFilters } from "../filters.js";
import { projectToContainerListEntry } from "../projections.js";
import { resolveByIdOrName } from "../name-resolution.js";
import type { ContainerInspect } from "../types.js";
import { formatTimestamp } from "../logs.js";
import type { LogEntry } from "../state.js";
import { generateStats } from "../stats.js";
import { generateTop } from "../top.js";
import { frameOutput } from "../stream.js";

export const containerRoutes: Route[] = [
    {
        method: "GET",
        pattern: "/containers/json",
        handler: async ({ res, query, state, clock }) => {
            const all = query.all === "1" || query.all === "true";
            const size = query.size === "1" || query.size === "true";
            const filters = parseFilters(query.filters);

            let containers = [...state.containers.values()];
            if (!all) {
                containers = containers.filter((c) => c.State.Running);
            }
            containers = applyContainerFilters(containers, filters);

            const entries = containers.map((c) => projectToContainerListEntry(c, clock, size));
            sendJSON(res, 200, entries);
        },
    },
    {
        method: "POST",
        pattern: "/containers/create",
        handler: async ({ req, res, query, state, emitter, clock }) => {
            const body = await readJSON<ContainerCreateConfig>(req);
            if (!body) {
                sendError(res, 400, "request body required");
                return;
            }
            const config: ContainerCreateConfig = {
                ...body,
                name: query.name || body.name,
            };
            const result = containerCreate(state, config, emitter, clock);
            if ("error" in result) {
                sendError(res, result.statusCode, result.error);
                return;
            }
            sendJSON(res, 201, { Id: result.ok.Id, Warnings: [] });
        },
    },
    {
        method: "GET",
        pattern: "/containers/:id/json",
        handler: async ({ res, params, state }) => {
            const r = resolveByIdOrName(
                state.containers,
                params.id,
                (c: ContainerInspect) => c.Name,
                (c: ContainerInspect) => c.Id,
            );
            if ("error" in r) {
                sendError(res, 404, `No such container: ${params.id}`);
                return;
            }
            sendJSON(res, 200, r.found);
        },
    },
    {
        method: "DELETE",
        pattern: "/containers/:id",
        handler: async ({ res, params, query, state, emitter, clock }) => {
            const force = query.force === "1" || query.force === "true";
            const result = containerRemove(state, params.id, emitter, clock, { force });
            handleMutationResult(res, result, 204);
        },
    },
    {
        method: "POST",
        pattern: "/containers/:id/start",
        handler: async ({ res, params, state, emitter, clock, e2eMode }) => {
            const result = containerStart(state, params.id, emitter, clock, { e2eMode });
            handleMutationResult(res, result, 204);
        },
    },
    {
        method: "POST",
        pattern: "/containers/:id/stop",
        handler: async ({ res, params, state, emitter, clock }) => {
            const result = containerStop(state, params.id, emitter, clock);
            handleMutationResult(res, result, 204);
        },
    },
    {
        method: "POST",
        pattern: "/containers/:id/restart",
        handler: async ({ res, params, state, emitter, clock }) => {
            const result = containerRestart(state, params.id, emitter, clock);
            handleMutationResult(res, result, 204);
        },
    },
    {
        method: "POST",
        pattern: "/containers/:id/kill",
        handler: async ({ res, params, query, state, emitter, clock }) => {
            const signal = query.signal || "SIGKILL";
            const result = containerKill(state, params.id, signal, emitter, clock);
            handleMutationResult(res, result, 204);
        },
    },
    {
        method: "POST",
        pattern: "/containers/:id/pause",
        handler: async ({ res, params, state, emitter, clock }) => {
            const result = containerPause(state, params.id, emitter, clock);
            handleMutationResult(res, result, 204);
        },
    },
    {
        method: "POST",
        pattern: "/containers/:id/unpause",
        handler: async ({ res, params, state, emitter, clock }) => {
            const result = containerUnpause(state, params.id, emitter, clock);
            handleMutationResult(res, result, 204);
        },
    },
    {
        method: "POST",
        pattern: "/containers/:id/rename",
        handler: async ({ res, params, query, state, emitter, clock }) => {
            if (!query.name) {
                sendError(res, 400, "name is required");
                return;
            }
            const result = containerRename(state, params.id, query.name, emitter, clock);
            handleMutationResult(res, result, 204);
        },
    },
    {
        method: "POST",
        pattern: "/containers/:id/update",
        handler: async ({ res }) => {
            sendJSON(res, 200, { Warnings: [] });
        },
    },
    {
        method: "GET",
        pattern: "/containers/:id/logs",
        handler: async (ctx) => {
            const { req, res, params, query, state } = ctx;
            const r = resolveByIdOrName(
                state.containers,
                params.id,
                (c: ContainerInspect) => c.Name,
                (c: ContainerInspect) => c.Id,
            );
            if ("error" in r) {
                sendError(res, 404, `No such container: ${params.id}`);
                return;
            }
            const container = r.found;
            const follow = query.follow === "1" || query.follow === "true";
            const tail = query.tail !== undefined && query.tail !== "all" ? parseInt(query.tail, 10) : undefined;
            // Per Docker API spec, since=0 and until=0 are the defaults meaning
            // "no filter" — treat 0 the same as omitted.
            const since = query.since ? parseFloat(query.since) || undefined : undefined;
            const until = query.until ? parseFloat(query.until) || undefined : undefined;
            const timestamps = query.timestamps === "1" || query.timestamps === "true";
            const isTty = container.Config.Tty || false;

            // Read from the per-container log buffer
            const buf = state.logBuffers.get(container.Id) || [];

            // Filter by since/until
            let filtered: LogEntry[] = buf;
            if (since !== undefined) {
                const sinceMs = since * 1000;
                filtered = filtered.filter((e) => e.ts >= sinceMs);
            }
            if (until !== undefined) {
                const untilMs = until * 1000;
                filtered = filtered.filter((e) => e.ts <= untilMs);
            }

            // Apply tail
            if (tail !== undefined) {
                if (tail <= 0) {
                    filtered = [];
                } else {
                    filtered = filtered.slice(-tail);
                }
            }

            // Format lines: optionally prepend timestamps
            const lines = timestamps
                ? filtered.map((e) => formatTimestamp(new Date(e.ts)) + " " + e.line)
                : filtered.map((e) => e.line);

            if (isTty) {
                res.writeHead(200, { "Content-Type": "application/vnd.docker.raw-stream" });
                for (const line of lines) {
                    res.write(line + "\n");
                }
            } else {
                res.writeHead(200, { "Content-Type": "application/vnd.docker.multiplexed-stream" });
                for (const line of lines) {
                    res.write(frameOutput(line));
                }
            }

            if (!follow || !container.State.Running) {
                res.end();
                return;
            }

            // Follow mode: subscribe to logEmitter for new lines from this container.
            // Track cursor position in the buffer to only send new lines.
            let cursor = (state.logBuffers.get(container.Id) || []).length;
            let stopped = false;

            const write = (line: string) => {
                if (isTty) {
                    res.write(line + "\n");
                } else {
                    res.write(frameOutput(line));
                }
            };

            const onLog = (containerId: string) => {
                if (stopped || containerId !== container.Id) return;
                const currentBuf = state.logBuffers.get(container.Id) || [];
                while (cursor < currentBuf.length) {
                    const entry = currentBuf[cursor++];
                    const line = timestamps
                        ? formatTimestamp(new Date(entry.ts)) + " " + entry.line
                        : entry.line;
                    write(line);
                }
            };
            state.logEmitter.on("log", onLog);

            // Subscribe to Docker events to detect die (stream ends on container stop).
            // Shutdown logs are already appended to the buffer by containerStop,
            // and will be delivered via the onLog listener above.
            const onEvent = (event: import("../list-types.js").DockerEvent) => {
                if (event.Action !== "die" || event.Actor.ID !== container.Id) return;
                stopped = true;
                ctx.emitter.unsubscribe(onEvent);
                state.logEmitter.off("log", onLog);
                // Flush any remaining buffered lines
                const currentBuf = state.logBuffers.get(container.Id) || [];
                while (cursor < currentBuf.length) {
                    const entry = currentBuf[cursor++];
                    const line = timestamps
                        ? formatTimestamp(new Date(entry.ts)) + " " + entry.line
                        : entry.line;
                    write(line);
                }
                res.end();
            };
            ctx.emitter.subscribe(onEvent);

            req.on("close", () => {
                stopped = true;
                ctx.emitter.unsubscribe(onEvent);
                state.logEmitter.off("log", onLog);
            });
        },
    },
    {
        method: "GET",
        pattern: "/containers/:id/stats",
        handler: async (ctx) => {
            const { req, res, params, query, state, clock } = ctx;
            const r = resolveByIdOrName(
                state.containers,
                params.id,
                (c: ContainerInspect) => c.Name,
                (c: ContainerInspect) => c.Id,
            );
            if ("error" in r) {
                sendError(res, 404, `No such container: ${params.id}`);
                return;
            }
            const container = r.found;

            if (!container.State.Running) {
                sendError(res, 409, `Container ${params.id} is not running`);
                return;
            }

            const oneShot = query["one-shot"] === "1" || query["one-shot"] === "true";
            const stream = query.stream !== "false" && !oneShot && !ctx.e2eMode;

            if (!stream) {
                // Single stats response — use persistent counter for varying data
                const counter = ctx.e2eMode ? 0 : ctx.state.nextStatsCounter(container.Id);
                const stats = generateStats(container, counter, clock);
                sendJSON(res, 200, stats);
                return;
            }

            // Streaming mode
            res.writeHead(200, {
                "Content-Type": "application/json",
                "Transfer-Encoding": "chunked",
            });

            let counter = 0;
            // Write first stats immediately
            res.write(JSON.stringify(generateStats(container, counter++, clock)) + "\n");

            const interval = setInterval(() => {
                const stats = generateStats(container, counter++, clock);
                res.write(JSON.stringify(stats) + "\n");
            }, ctx.statsInterval);

            req.on("close", () => {
                clearInterval(interval);
            });
        },
    },
    {
        method: "GET",
        pattern: "/containers/:id/top",
        handler: async ({ res, params, state }) => {
            const r = resolveByIdOrName(
                state.containers,
                params.id,
                (c: ContainerInspect) => c.Name,
                (c: ContainerInspect) => c.Id,
            );
            if ("error" in r) {
                sendError(res, 404, `No such container: ${params.id}`);
                return;
            }
            const container = r.found;

            if (!container.State.Running) {
                sendError(res, 500, `Container ${params.id} is not running`);
                return;
            }

            sendJSON(res, 200, generateTop(container));
        },
    },
    {
        method: "POST",
        pattern: "/containers/:id/exec",
        handler: async ({ req, res, params, state, clock }) => {
            const body = await readJSON<{
                Cmd?: string[];
                AttachStdin?: boolean;
                AttachStdout?: boolean;
                AttachStderr?: boolean;
                Tty?: boolean;
                User?: string;
            }>(req);
            const result = execCreate(state, params.id, {
                Cmd: body?.Cmd || ["/bin/sh"],
                AttachStdin: body?.AttachStdin ?? false,
                AttachStdout: body?.AttachStdout ?? true,
                AttachStderr: body?.AttachStderr ?? true,
                Tty: body?.Tty ?? false,
                User: body?.User,
            }, clock);
            handleMutationResult(res, result, 201);
        },
    },
];
