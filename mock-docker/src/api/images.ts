import type { Route } from "../server.js";
import { sendJSON, sendError, handleMutationResult } from "../server.js";
import { imageRemove, imagePrune } from "../mutations.js";
import { parseFilters, applyImageFilters } from "../filters.js";
import { projectToImageListEntry } from "../projections.js";
import type { ImageInspect } from "../types.js";

function resolveImageByName(images: Map<string, ImageInspect>, name: string): ImageInspect | null {
    // Exact ID match
    const byId = images.get(name);
    if (byId) return byId;

    // Search by tag or digest
    for (const img of images.values()) {
        for (const tag of img.RepoTags) {
            if (tag === name) return img;
        }
        for (const digest of img.RepoDigests) {
            if (digest === name) return img;
        }
        // Short ID prefix
        if (name.length >= 3 && img.Id.startsWith("sha256:" + name)) return img;
        if (name.length >= 3 && img.Id.startsWith(name)) return img;
    }
    return null;
}

export const imageRoutes: Route[] = [
    {
        method: "GET",
        pattern: "/images/json",
        handler: async ({ res, query, state }) => {
            const filters = parseFilters(query.filters);
            const images = applyImageFilters([...state.images.values()], filters);
            const entries = images.map((img) => projectToImageListEntry(img, state.containers));
            sendJSON(res, 200, entries);
        },
    },
    {
        method: "POST",
        pattern: "/images/prune",
        handler: async ({ res, query, state, emitter, clock }) => {
            const filters = parseFilters(query.filters);
            // If dangling=false is in filters, that means prune all (not just dangling)
            const danglingValues = filters.get("dangling");
            const all = danglingValues ? danglingValues.includes("false") : false;
            const result = imagePrune(state, all, emitter, clock);
            handleMutationResult(res, result, 200);
        },
    },
    {
        method: "GET",
        pattern: "/images/*",
        handler: async ({ res, params, state }) => {
            let name = params["*"];
            // Strip /json suffix
            if (name.endsWith("/json")) {
                name = name.slice(0, -5);
            }
            const img = resolveImageByName(state.images, name);
            if (!img) {
                sendError(res, 404, `No such image: ${name}`);
                return;
            }
            sendJSON(res, 200, img);
        },
    },
    {
        method: "DELETE",
        pattern: "/images/*",
        handler: async ({ res, params, query, state, emitter, clock }) => {
            let name = params["*"];
            // Strip /json suffix just in case
            if (name.endsWith("/json")) {
                name = name.slice(0, -5);
            }
            const force = query.force === "1" || query.force === "true";
            const result = imageRemove(state, name, emitter, clock, { force });
            handleMutationResult(res, result, 200);
        },
    },
];
