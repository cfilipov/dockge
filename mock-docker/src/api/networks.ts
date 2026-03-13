import type { Route } from "../server.js";
import { sendJSON, sendError, readJSON, handleMutationResult } from "../server.js";
import { networkCreate, networkRemove, networkConnect, networkDisconnect } from "../mutations.js";
import type { NetworkCreateConfig } from "../mutations.js";
import { parseFilters, applyNetworkFilters } from "../filters.js";
import { resolveByIdOrName } from "../name-resolution.js";
import type { NetworkInspect } from "../types.js";

export const networkRoutes: Route[] = [
    {
        method: "GET",
        pattern: "/networks",
        handler: async ({ res, query, state }) => {
            const filters = parseFilters(query.filters);
            const networks = applyNetworkFilters([...state.networks.values()], filters);
            sendJSON(res, 200, networks);
        },
    },
    {
        method: "POST",
        pattern: "/networks/create",
        handler: async ({ req, res, state, emitter, clock }) => {
            const body = await readJSON<NetworkCreateConfig>(req);
            if (!body || !body.Name) {
                sendError(res, 400, "network name is required");
                return;
            }
            const result = networkCreate(state, body, emitter, clock);
            if ("error" in result) {
                sendError(res, result.statusCode, result.error);
                return;
            }
            sendJSON(res, 201, { Id: result.ok.Id });
        },
    },
    {
        method: "GET",
        pattern: "/networks/:id",
        handler: async ({ res, params, state }) => {
            const r = resolveByIdOrName(
                state.networks,
                params.id,
                (n: NetworkInspect) => n.Name,
                (n: NetworkInspect) => n.Id,
            );
            if ("error" in r) {
                sendError(res, 404, `network ${params.id} not found`);
                return;
            }
            sendJSON(res, 200, r.found);
        },
    },
    {
        method: "DELETE",
        pattern: "/networks/:id",
        handler: async ({ res, params, state, emitter, clock }) => {
            const result = networkRemove(state, params.id, emitter, clock);
            handleMutationResult(res, result, 204);
        },
    },
    {
        method: "POST",
        pattern: "/networks/:id/connect",
        handler: async ({ req, res, params, state, emitter, clock }) => {
            const body = await readJSON<{ Container?: string; EndpointConfig?: { IPv4Address?: string; IPv6Address?: string } }>(req);
            if (!body?.Container) {
                sendError(res, 400, "Container is required");
                return;
            }
            const config = body.EndpointConfig ? {
                IPv4Address: body.EndpointConfig.IPv4Address,
                IPv6Address: body.EndpointConfig.IPv6Address,
            } : undefined;
            const result = networkConnect(state, params.id, body.Container, config, emitter, clock);
            handleMutationResult(res, result, 200);
        },
    },
    {
        method: "POST",
        pattern: "/networks/:id/disconnect",
        handler: async ({ req, res, params, state, emitter, clock }) => {
            const body = await readJSON<{ Container?: string; Force?: boolean }>(req);
            if (!body?.Container) {
                sendError(res, 400, "Container is required");
                return;
            }
            const result = networkDisconnect(state, params.id, body.Container, emitter, clock);
            handleMutationResult(res, result, 200);
        },
    },
];
