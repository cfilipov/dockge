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
import { getHistoricalLogs, generatePeriodicLogLine, generateShutdownLogs, generateStartupLogs } from "../logs.js";
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
            const follow = query.follow === "1" || query.follow === "true";
            const tail = query.tail !== undefined && query.tail !== "all" ? parseInt(query.tail, 10) : undefined;
            const since = query.since ? parseFloat(query.since) : undefined;
            const until = query.until ? parseFloat(query.until) : undefined;
            const isTty = container.Config.Tty || false;

            // Get historical logs
            const lines = getHistoricalLogs(container, clock, { tail, since, until });

            if (isTty) {
                // Raw mode for TTY containers
                res.writeHead(200, { "Content-Type": "application/vnd.docker.raw-stream" });
                for (const line of lines) {
                    res.write(line + "\n");
                }
            } else {
                // Multiplexed framing
                res.writeHead(200, { "Content-Type": "application/vnd.docker.multiplexed-stream" });
                for (const line of lines) {
                    res.write(frameOutput(line));
                }
            }

            if (!follow || !container.State.Running || ctx.e2eMode) {
                res.end();
                return;
            }

            const write = (line: string) => {
                if (isTty) {
                    res.write(line + "\n");
                } else {
                    res.write(frameOutput(line));
                }
            };

            // Emit startup logs for freshly-started containers
            const startupLines = generateStartupLogs(container, clock);
            for (const line of startupLines) {
                write(line);
            }

            // Follow mode: stream periodic lines, stop when container dies
            let lineCounter = 0;
            let stopped = false;
            const interval = setInterval(() => {
                if (stopped) return;
                const line = generatePeriodicLogLine(container, lineCounter++, clock);
                write(line);
            }, ctx.logInterval);

            // Subscribe to events so we detect stop synchronously (same
            // emitter.emit() call that fires the die event to /events).
            // This ensures shutdown logs are written to the log stream
            // before the Go backend receives the die event.
            const onEvent = (event: import("../list-types.js").DockerEvent) => {
                if (event.Action !== "die" || event.Actor.ID !== container.Id) return;
                stopped = true;
                clearInterval(interval);
                ctx.emitter.unsubscribe(onEvent);
                const shutdownLines = generateShutdownLogs(container, clock);
                for (const line of shutdownLines) {
                    write(line);
                }
                res.end();
            };
            ctx.emitter.subscribe(onEvent);

            req.on("close", () => {
                clearInterval(interval);
                ctx.emitter.unsubscribe(onEvent);
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
                // Single stats response
                const stats = generateStats(container, 0, clock);
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
