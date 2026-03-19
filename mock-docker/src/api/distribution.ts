import type { Route } from "../server.js";
import { sendJSON, sendError } from "../server.js";
import type { ImageInspect } from "../types.js";
import type { MockState } from "../state.js";
import { createHash } from "node:crypto";

function resolveImageByName(images: Map<string, ImageInspect>, name: string): ImageInspect | null {
    for (const img of images.values()) {
        for (const tag of img.RepoTags) {
            if (tag === name) return img;
            // Match without tag (e.g., "nginx" matches "nginx:latest")
            const tagBase = tag.split(":")[0];
            if (tagBase === name) return img;
        }
        for (const digest of img.RepoDigests) {
            if (digest.startsWith(name + "@")) return img;
        }
    }
    return null;
}

/**
 * Check if the given image has an update available (from global .mock.yaml).
 */
function hasUpdateAvailable(state: MockState, imageName: string): boolean {
    // Exact match (e.g. "nginx:latest")
    if (state.updateImages.has(imageName)) return true;
    // Match without tag (e.g. "nginx" matches "nginx:latest")
    for (const ref of state.updateImages) {
        if (ref.split(":")[0] === imageName) return true;
    }
    return false;
}

function alteredDigest(digest: string): string {
    return "sha256:" + createHash("sha256").update(digest + "-updated").digest("hex");
}

export const distributionRoutes: Route[] = [
    {
        method: "GET",
        pattern: "/distribution/*",
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

            // Extract digest from RepoDigests or use image ID
            let digest = img.Id;
            if (img.RepoDigests.length > 0) {
                const atIdx = img.RepoDigests[0].indexOf("@");
                if (atIdx !== -1) {
                    digest = img.RepoDigests[0].slice(atIdx + 1);
                }
            }

            // If this image has update_available in global .mock.yaml, return an altered digest
            if (hasUpdateAvailable(state, name)) {
                digest = alteredDigest(digest);
            }

            sendJSON(res, 200, {
                Descriptor: {
                    mediaType: "application/vnd.oci.image.index.v1+json",
                    digest,
                    size: img.Size,
                },
                Platforms: [
                    { architecture: "amd64", os: "linux" },
                    { architecture: "arm64", os: "linux" },
                ],
            });
        },
    },
];
