import type { Route } from "../server.js";
import { sendJSON, sendError, readJSON, handleMutationResult } from "../server.js";
import { volumeCreate, volumeRemove } from "../mutations.js";
import type { VolumeCreateConfig } from "../mutations.js";
import { parseFilters, applyVolumeFilters } from "../filters.js";

export const volumeRoutes: Route[] = [
    {
        method: "GET",
        pattern: "/volumes",
        handler: async ({ res, query, state }) => {
            const filters = parseFilters(query.filters);
            const volumes = applyVolumeFilters([...state.volumes.values()], filters, state.containers);
            sendJSON(res, 200, { Volumes: volumes, Warnings: [] });
        },
    },
    {
        method: "POST",
        pattern: "/volumes/create",
        handler: async ({ req, res, state, emitter, clock }) => {
            const body = await readJSON<VolumeCreateConfig>(req);
            if (!body || !body.Name) {
                sendError(res, 400, "volume name is required");
                return;
            }
            const result = volumeCreate(state, body, emitter, clock);
            if ("error" in result) {
                sendError(res, result.statusCode, result.error);
                return;
            }
            sendJSON(res, 201, result.ok);
        },
    },
    {
        method: "GET",
        pattern: "/volumes/:name",
        handler: async ({ res, params, state }) => {
            const vol = state.volumes.get(params.name);
            if (!vol) {
                sendError(res, 404, `get ${params.name}: no such volume`);
                return;
            }
            sendJSON(res, 200, vol);
        },
    },
    {
        method: "DELETE",
        pattern: "/volumes/:name",
        handler: async ({ res, params, state, emitter, clock }) => {
            const result = volumeRemove(state, params.name, emitter, clock);
            handleMutationResult(res, result, 204);
        },
    },
];
