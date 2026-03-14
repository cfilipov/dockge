import type { Route } from "../server.js";
import { sendJSON, sendError, handleMutationResult } from "../server.js";
import { imageRemove, imagePrune } from "../mutations.js";
import { parseFilters, applyImageFilters } from "../filters.js";
import { projectToImageListEntry } from "../projections.js";
import type { ImageInspect } from "../types.js";
import { deterministicId, deterministicInt } from "../deterministic.js";

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

interface ImageHistoryItem {
    Id: string;
    Created: number;
    CreatedBy: string;
    Size: number;
    Comment: string;
    Tags: string[] | null;
}

function getCmdForImage(imageRef: string): string {
    let baseName = imageRef;
    const slashIdx = baseName.lastIndexOf("/");
    if (slashIdx >= 0) baseName = baseName.slice(slashIdx + 1);
    const colonIdx = baseName.indexOf(":");
    if (colonIdx >= 0) baseName = baseName.slice(0, colonIdx);

    switch (baseName) {
        case "nginx": case "httpd": return `CMD ["nginx", "-g", "daemon off;"]`;
        case "redis": return `CMD ["redis-server"]`;
        case "postgres": return `CMD ["postgres"]`;
        case "mysql": case "mariadb": return `CMD ["mysqld"]`;
        case "node": return `CMD ["node"]`;
        case "python": return `CMD ["python3"]`;
        case "grafana": return `ENTRYPOINT ["/run.sh"]`;
        case "wordpress": return `CMD ["apache2-foreground"]`;
        case "traefik": return `ENTRYPOINT ["/entrypoint.sh"]`;
        case "elasticsearch": return `ENTRYPOINT ["/bin/tini", "--", "/usr/local/bin/docker-entrypoint.sh"]`;
        case "rabbitmq": return `CMD ["rabbitmq-server"]`;
        default: return `CMD ["/bin/sh"]`;
    }
}

function generateImageHistory(img: ImageInspect): ImageHistoryItem[] {
    const seed = img.Id;
    const createdUnix = Math.floor(new Date(img.Created).getTime() / 1000);
    const numLayers = 2 + (deterministicInt(seed + "layers", 0, 2));

    const layers: ImageHistoryItem[] = [];

    // Top layer: image's own CMD, small size
    const topTag = img.RepoTags.length > 0 ? img.RepoTags[0] : img.Id;
    layers.push({
        Id: "sha256:" + deterministicId(seed, "layer-top"),
        Created: createdUnix,
        CreatedBy: getCmdForImage(topTag),
        Size: 0,
        Comment: "",
        Tags: img.RepoTags.length > 0 ? [...img.RepoTags] : null,
    });

    // Middle layers: RUN commands
    for (let i = 1; i < numLayers - 1; i++) {
        layers.push({
            Id: "<missing>",
            Created: createdUnix,
            CreatedBy: "RUN /bin/sh -c set -x && install dependencies # buildkit",
            Size: deterministicInt(seed + `layer-mid-${i}`, 1_000_000, 200_000_000),
            Comment: "",
            Tags: null,
        });
    }

    // Base layer: ADD file
    layers.push({
        Id: "<missing>",
        Created: createdUnix,
        CreatedBy: "/bin/sh -c #(nop) ADD file:... in /",
        Size: deterministicInt(seed + "layer-base", 5_000_000, 500_000_000),
        Comment: "",
        Tags: null,
    });

    return layers;
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
            const isHistory = name.endsWith("/history");
            if (isHistory) {
                name = name.slice(0, -"/history".length);
            } else if (name.endsWith("/json")) {
                name = name.slice(0, -"/json".length);
            }
            const img = resolveImageByName(state.images, name);
            if (!img) {
                sendError(res, 404, `No such image: ${name}`);
                return;
            }
            if (isHistory) {
                sendJSON(res, 200, generateImageHistory(img));
            } else {
                sendJSON(res, 200, img);
            }
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
