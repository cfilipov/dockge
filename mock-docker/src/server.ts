import { createServer as createHttpServer, type IncomingMessage, type ServerResponse } from "node:http";
import type { MockState } from "./state.js";
import type { EventEmitter } from "./events.js";
import type { Clock } from "./clock.js";
import type { InitOptions } from "./init.js";
import type { MutationResult } from "./mutations.js";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface RequestContext {
    req: IncomingMessage;
    res: ServerResponse;
    params: Record<string, string>;
    query: Record<string, string>;
    state: MockState;
    emitter: EventEmitter;
    clock: Clock;
    initOpts: InitOptions;
    e2eMode: boolean;
    logInterval: number;
    statsInterval: number;
}

export type RouteHandler = (ctx: RequestContext) => Promise<void>;

export interface Route {
    method: string;
    pattern: string;
    handler: RouteHandler;
}

export interface ServerOptions {
    socketPath: string;
    state: MockState;
    emitter: EventEmitter;
    clock: Clock;
    initOpts: InitOptions;
    e2eMode?: boolean;
    logInterval?: number;
    statsInterval?: number;
}

// ---------------------------------------------------------------------------
// Response helpers
// ---------------------------------------------------------------------------

export function sendJSON(res: ServerResponse, statusCode: number, body: unknown): void {
    const json = JSON.stringify(body);
    res.writeHead(statusCode, {
        "Content-Type": "application/json",
        "Content-Length": Buffer.byteLength(json),
    });
    res.end(json);
}

export function sendError(res: ServerResponse, statusCode: number, message: string): void {
    sendJSON(res, statusCode, { message });
}

export function sendPlain(res: ServerResponse, statusCode: number, text: string): void {
    res.writeHead(statusCode, {
        "Content-Type": "text/plain; charset=utf-8",
        "Content-Length": Buffer.byteLength(text),
    });
    res.end(text);
}

export function sendNoContent(res: ServerResponse, statusCode: number = 204): void {
    res.writeHead(statusCode);
    res.end();
}

// ---------------------------------------------------------------------------
// Request helpers
// ---------------------------------------------------------------------------

export async function readJSON<T = unknown>(req: IncomingMessage): Promise<T | null> {
    return new Promise((resolve, reject) => {
        const chunks: Buffer[] = [];
        req.on("data", (chunk: Buffer) => chunks.push(chunk));
        req.on("end", () => {
            const body = Buffer.concat(chunks).toString();
            if (!body || body.trim() === "") {
                resolve(null);
                return;
            }
            try {
                resolve(JSON.parse(body) as T);
            } catch {
                reject(new Error("invalid JSON in request body"));
            }
        });
        req.on("error", reject);
    });
}

// ---------------------------------------------------------------------------
// Mutation result → HTTP response
// ---------------------------------------------------------------------------

export function handleMutationResult<T>(
    res: ServerResponse,
    result: MutationResult<T>,
    successCode: number = 200,
): void {
    if ("error" in result) {
        if (result.statusCode === 304) {
            sendNoContent(res, 304);
        } else {
            sendError(res, result.statusCode, result.error);
        }
        return;
    }

    if (result.ok === undefined || result.ok === null) {
        sendNoContent(res, successCode === 200 ? 204 : successCode);
    } else {
        sendJSON(res, successCode, result.ok);
    }
}

// ---------------------------------------------------------------------------
// Router
// ---------------------------------------------------------------------------

const VERSION_PREFIX_RE = /^\/v\d+\.\d+/;

interface ParsedRoute {
    method: string;
    segments: string[];
    greedy: boolean;
    handler: RouteHandler;
}

function parsePattern(pattern: string): { segments: string[]; greedy: boolean } {
    const parts = pattern.split("/").filter(Boolean);
    const greedy = parts.length > 0 && parts[parts.length - 1] === "*";
    if (greedy) parts.pop();
    return { segments: parts, greedy };
}

function matchRoute(
    route: ParsedRoute,
    method: string,
    pathSegments: string[],
): Record<string, string> | null {
    if (route.method !== method) return null;

    const params: Record<string, string> = {};

    if (route.greedy) {
        // Greedy: route segments must match as prefix, rest captured as "*"
        if (pathSegments.length < route.segments.length) return null;
        for (let i = 0; i < route.segments.length; i++) {
            const seg = route.segments[i];
            if (seg.startsWith(":")) {
                params[seg.slice(1)] = pathSegments[i];
            } else if (seg !== pathSegments[i]) {
                return null;
            }
        }
        params["*"] = pathSegments.slice(route.segments.length).join("/");
        return params;
    }

    // Exact length match for non-greedy routes
    if (pathSegments.length !== route.segments.length) return null;

    for (let i = 0; i < route.segments.length; i++) {
        const seg = route.segments[i];
        if (seg.startsWith(":")) {
            params[seg.slice(1)] = pathSegments[i];
        } else if (seg !== pathSegments[i]) {
            return null;
        }
    }

    return params;
}

function parseQueryString(search: string): Record<string, string> {
    const params: Record<string, string> = {};
    if (!search) return params;
    const qs = search.startsWith("?") ? search.slice(1) : search;
    for (const pair of qs.split("&")) {
        const eqIdx = pair.indexOf("=");
        if (eqIdx === -1) {
            params[decodeURIComponent(pair)] = "";
        } else {
            params[decodeURIComponent(pair.slice(0, eqIdx))] = decodeURIComponent(pair.slice(eqIdx + 1));
        }
    }
    return params;
}

// ---------------------------------------------------------------------------
// Server factory
// ---------------------------------------------------------------------------

export function createServer(opts: ServerOptions, routes: Route[]) {
    const parsedRoutes: ParsedRoute[] = routes.map((r) => {
        const { segments, greedy } = parsePattern(r.pattern);
        return { method: r.method, segments, greedy, handler: r.handler };
    });

    const server = createHttpServer(async (req, res) => {
        try {
            const url = new URL(req.url || "/", "http://localhost");
            // Strip version prefix
            let pathname = url.pathname;
            pathname = pathname.replace(VERSION_PREFIX_RE, "");
            if (!pathname.startsWith("/")) pathname = "/" + pathname;

            const method = (req.method || "GET").toUpperCase();
            const pathSegments = pathname.split("/").filter(Boolean);
            const query = parseQueryString(url.search);

            // Find matching route
            for (const route of parsedRoutes) {
                const params = matchRoute(route, method, pathSegments);
                if (params !== null) {
                    const ctx: RequestContext = {
                        req,
                        res,
                        params,
                        query,
                        state: opts.state,
                        emitter: opts.emitter,
                        clock: opts.clock,
                        initOpts: opts.initOpts,
                        e2eMode: opts.e2eMode ?? false,
                        logInterval: opts.logInterval ?? 5000,
                        statsInterval: opts.statsInterval ?? 1000,
                    };
                    await route.handler(ctx);
                    return;
                }
            }

            // No match
            sendJSON(res, 404, { message: "page not found" });
        } catch (err) {
            const message = err instanceof Error ? err.message : "internal server error";
            console.error("Handler error:", err);
            sendError(res, 500, message);
        }
    });

    return {
        server,
        start(): Promise<void> {
            return new Promise((resolve) => {
                server.listen(opts.socketPath, () => resolve());
            });
        },
        stop(): Promise<void> {
            return new Promise((resolve, reject) => {
                server.close((err) => (err ? reject(err) : resolve()));
            });
        },
    };
}
