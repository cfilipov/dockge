import { describe, it, expect } from "vitest";
import {
    parseFilters,
    applyContainerFilters,
    applyNetworkFilters,
    applyVolumeFilters,
    applyImageFilters,
    applyEventFilters,
} from "../src/filters.js";
import type { ContainerInspect, NetworkInspect, VolumeInspect, ImageInspect } from "../src/types.js";
import type { DockerEvent } from "../src/list-types.js";

// Minimal container builder for filter tests.
function makeContainer(opts: {
    id?: string;
    name?: string;
    labels?: Record<string, string>;
    status?: string;
    image?: string;
    imageId?: string;
    networks?: Record<string, { NetworkID?: string }>;
    mounts?: Array<{ Name?: string }>;
}): ContainerInspect {
    return {
        Id: opts.id ?? "a".repeat(64),
        Name: opts.name ?? "/test-container",
        State: { Status: opts.status ?? "running" } as any,
        Config: {
            Image: opts.image ?? "nginx:latest",
            Labels: opts.labels ?? {},
        } as any,
        Image: opts.imageId ?? "sha256:abc",
        NetworkSettings: {
            Networks: (opts.networks ?? {}) as any,
        } as any,
        Mounts: (opts.mounts ?? []) as any,
    } as ContainerInspect;
}

function makeNetwork(opts: {
    id?: string;
    name?: string;
    labels?: Record<string, string>;
    driver?: string;
    scope?: string;
}): NetworkInspect {
    return {
        Id: opts.id ?? "net" + "0".repeat(61),
        Name: opts.name ?? "test-net",
        Labels: opts.labels ?? {},
        Driver: opts.driver ?? "bridge",
        Scope: opts.scope ?? "local",
    } as NetworkInspect;
}

function makeVolume(opts: {
    name?: string;
    labels?: Record<string, string>;
    driver?: string;
}): VolumeInspect {
    return {
        Name: opts.name ?? "test-vol",
        Labels: opts.labels ?? {},
        Driver: opts.driver ?? "local",
    } as VolumeInspect;
}

function makeImageInspect(opts: {
    id?: string;
    repoTags?: string[];
    labels?: Record<string, string>;
}): ImageInspect {
    return {
        Id: opts.id ?? "sha256:img",
        RepoTags: opts.repoTags ?? ["nginx:latest"],
        Config: { Labels: opts.labels ?? {} } as any,
    } as ImageInspect;
}

// === parseFilters ===

describe("parseFilters", () => {
    it("parses valid JSON filter", () => {
        const result = parseFilters('{"label":["com.docker.compose.project=mystack"]}');
        expect(result.get("label")).toEqual(["com.docker.compose.project=mystack"]);
    });

    it("returns empty map for undefined", () => {
        expect(parseFilters(undefined).size).toBe(0);
    });

    it("returns empty map for empty string", () => {
        expect(parseFilters("").size).toBe(0);
    });

    it("returns empty map for malformed JSON", () => {
        expect(parseFilters("{invalid json}").size).toBe(0);
    });

    it("returns empty map for non-object JSON", () => {
        expect(parseFilters('"hello"').size).toBe(0);
        expect(parseFilters("[1,2,3]").size).toBe(0);
    });

    it("skips non-string-array values", () => {
        const result = parseFilters('{"good":["a"],"bad":123}');
        expect(result.has("good")).toBe(true);
        expect(result.has("bad")).toBe(false);
    });

    it("handles multiple keys", () => {
        const result = parseFilters('{"label":["env=prod"],"status":["running","paused"]}');
        expect(result.get("label")).toEqual(["env=prod"]);
        expect(result.get("status")).toEqual(["running", "paused"]);
    });
});

// === applyContainerFilters ===

describe("applyContainerFilters", () => {
    const containers = [
        makeContainer({ id: "abcdef" + "0".repeat(58), name: "/stack1-web-1", labels: { "com.docker.compose.project": "stack1", env: "prod" }, status: "running", image: "nginx:latest", imageId: "sha256:nginx1", networks: { stack1_default: { NetworkID: "net1" } }, mounts: [{ Name: "data-vol" }] }),
        makeContainer({ id: "123456" + "0".repeat(58), name: "/stack2-api-1", labels: { "com.docker.compose.project": "stack2", env: "dev" }, status: "exited", image: "node:18", imageId: "sha256:node1", networks: { stack2_default: { NetworkID: "net2" } }, mounts: [] }),
        makeContainer({ id: "fedcba" + "0".repeat(58), name: "/stack1-db-1", labels: { "com.docker.compose.project": "stack1" }, status: "running", image: "postgres:15", imageId: "sha256:pg1", networks: { stack1_default: { NetworkID: "net1" } }, mounts: [{ Name: "pg-data" }] }),
    ];

    it("returns all when filters are empty", () => {
        const result = applyContainerFilters(containers, new Map());
        expect(result).toHaveLength(3);
    });

    it("filters by label key-only", () => {
        const result = applyContainerFilters(containers, new Map([["label", ["env"]]]));
        expect(result).toHaveLength(2);
    });

    it("filters by label key=value", () => {
        const result = applyContainerFilters(containers, new Map([["label", ["com.docker.compose.project=stack1"]]]));
        expect(result).toHaveLength(2);
        expect(result.every((c) => c.Config.Labels!["com.docker.compose.project"] === "stack1")).toBe(true);
    });

    it("filters by status (single value)", () => {
        const result = applyContainerFilters(containers, new Map([["status", ["running"]]]));
        expect(result).toHaveLength(2);
    });

    it("filters by status (multiple values — OR)", () => {
        const result = applyContainerFilters(containers, new Map([["status", ["running", "exited"]]]));
        expect(result).toHaveLength(3);
    });

    it("filters by name (without leading /)", () => {
        const result = applyContainerFilters(containers, new Map([["name", ["stack2-api-1"]]]));
        expect(result).toHaveLength(1);
        expect(result[0].Name).toBe("/stack2-api-1");
    });

    it("filters by name (with leading /)", () => {
        const result = applyContainerFilters(containers, new Map([["name", ["/stack1-web-1"]]]));
        expect(result).toHaveLength(1);
    });

    it("filters by name substring match", () => {
        const result = applyContainerFilters(containers, new Map([["name", ["web"]]]));
        expect(result).toHaveLength(1);
        expect(result[0].Name).toBe("/stack1-web-1");
    });

    it("filters by name prefix match across multiple containers", () => {
        const result = applyContainerFilters(containers, new Map([["name", ["stack1"]]]));
        expect(result).toHaveLength(2);
    });

    it("filters by id prefix", () => {
        const result = applyContainerFilters(containers, new Map([["id", ["abcdef"]]]));
        expect(result).toHaveLength(1);
        expect(result[0].Id.startsWith("abcdef")).toBe(true);
    });

    it("AND across multiple filter keys", () => {
        const result = applyContainerFilters(containers, new Map([
            ["label", ["com.docker.compose.project=stack1"]],
            ["status", ["running"]],
        ]));
        expect(result).toHaveLength(2);
    });

    it("filters by ancestor (image name)", () => {
        const result = applyContainerFilters(containers, new Map([["ancestor", ["nginx:latest"]]]));
        expect(result).toHaveLength(1);
    });

    it("filters by ancestor (image ID)", () => {
        const result = applyContainerFilters(containers, new Map([["ancestor", ["sha256:node1"]]]));
        expect(result).toHaveLength(1);
    });

    it("filters by ancestor (image ID prefix)", () => {
        const result = applyContainerFilters(containers, new Map([["ancestor", ["sha256:ngin"]]]));
        expect(result).toHaveLength(1);
    });

    it("filters by ancestor (image name without tag)", () => {
        const result = applyContainerFilters(containers, new Map([["ancestor", ["nginx"]]]));
        expect(result).toHaveLength(1);
        expect(result[0].Name).toBe("/stack1-web-1");
    });

    it("filters by network name", () => {
        const result = applyContainerFilters(containers, new Map([["network", ["stack1_default"]]]));
        expect(result).toHaveLength(2);
    });

    it("filters by network ID", () => {
        const result = applyContainerFilters(containers, new Map([["network", ["net2"]]]));
        expect(result).toHaveLength(1);
    });

    it("filters by volume name", () => {
        const result = applyContainerFilters(containers, new Map([["volume", ["data-vol"]]]));
        expect(result).toHaveLength(1);
    });
});

// === applyNetworkFilters ===

describe("applyNetworkFilters", () => {
    const networks = [
        makeNetwork({ id: "net1" + "0".repeat(60), name: "bridge", driver: "bridge", scope: "local" }),
        makeNetwork({ id: "net2" + "0".repeat(60), name: "host", driver: "host", scope: "local" }),
        makeNetwork({ id: "net3" + "0".repeat(60), name: "myapp_default", driver: "bridge", scope: "local", labels: { "com.docker.compose.project": "myapp" } }),
        makeNetwork({ id: "net4" + "0".repeat(60), name: "overlay-net", driver: "overlay", scope: "swarm" }),
    ];

    it("returns all when filters are empty", () => {
        expect(applyNetworkFilters(networks, new Map())).toHaveLength(4);
    });

    it("filters by driver", () => {
        const result = applyNetworkFilters(networks, new Map([["driver", ["bridge"]]]));
        expect(result).toHaveLength(2);
    });

    it("filters by name", () => {
        const result = applyNetworkFilters(networks, new Map([["name", ["myapp_default"]]]));
        expect(result).toHaveLength(1);
    });

    it("filters by id prefix", () => {
        const result = applyNetworkFilters(networks, new Map([["id", ["net3"]]]));
        expect(result).toHaveLength(1);
    });

    it("filters by type custom", () => {
        const result = applyNetworkFilters(networks, new Map([["type", ["custom"]]]));
        expect(result).toHaveLength(2); // myapp_default and overlay-net
        expect(result.every((n) => !["bridge", "host", "none"].includes(n.Name))).toBe(true);
    });

    it("filters by type builtin", () => {
        const result = applyNetworkFilters(networks, new Map([["type", ["builtin"]]]));
        expect(result).toHaveLength(2); // bridge and host
    });

    it("filters by scope", () => {
        const result = applyNetworkFilters(networks, new Map([["scope", ["swarm"]]]));
        expect(result).toHaveLength(1);
    });

    it("filters by label", () => {
        const result = applyNetworkFilters(networks, new Map([["label", ["com.docker.compose.project=myapp"]]]));
        expect(result).toHaveLength(1);
    });
});

// === applyVolumeFilters ===

describe("applyVolumeFilters", () => {
    const volumes = [
        makeVolume({ name: "data-vol", labels: { app: "web" }, driver: "local" }),
        makeVolume({ name: "cache-vol", labels: {}, driver: "local" }),
        makeVolume({ name: "nfs-share", labels: { type: "nfs" }, driver: "nfs" }),
    ];

    it("returns all when filters are empty", () => {
        expect(applyVolumeFilters(volumes, new Map())).toHaveLength(3);
    });

    it("filters by name", () => {
        const result = applyVolumeFilters(volumes, new Map([["name", ["data-vol"]]]));
        expect(result).toHaveLength(1);
    });

    it("filters by label", () => {
        const result = applyVolumeFilters(volumes, new Map([["label", ["app"]]]));
        expect(result).toHaveLength(1);
    });

    it("filters by driver", () => {
        const result = applyVolumeFilters(volumes, new Map([["driver", ["nfs"]]]));
        expect(result).toHaveLength(1);
    });

    it("filters dangling=true (volume not used by any container)", () => {
        const c = makeContainer({ mounts: [{ Name: "data-vol" }] });
        const containers = new Map([[c.Id, c]]);
        const result = applyVolumeFilters(volumes, new Map([["dangling", ["true"]]]), containers);
        // cache-vol and nfs-share are dangling (not mounted by any container)
        expect(result).toHaveLength(2);
        expect(result.map((v) => v.Name).sort()).toEqual(["cache-vol", "nfs-share"]);
    });

    it("filters dangling=false (volume in use)", () => {
        const c = makeContainer({ mounts: [{ Name: "data-vol" }] });
        const containers = new Map([[c.Id, c]]);
        const result = applyVolumeFilters(volumes, new Map([["dangling", ["false"]]]), containers);
        expect(result).toHaveLength(1);
        expect(result[0].Name).toBe("data-vol");
    });
});

// === applyImageFilters ===

describe("applyImageFilters", () => {
    const images = [
        makeImageInspect({ id: "sha256:img1", repoTags: ["nginx:latest", "nginx:1.25"], labels: { maintainer: "nginx" } }),
        makeImageInspect({ id: "sha256:img2", repoTags: ["node:18-alpine"], labels: { env: "prod" } }),
        makeImageInspect({ id: "sha256:img3", repoTags: [], labels: {} }), // dangling
    ];

    it("returns all when filters are empty", () => {
        expect(applyImageFilters(images, new Map())).toHaveLength(3);
    });

    it("filters by label", () => {
        const result = applyImageFilters(images, new Map([["label", ["maintainer"]]]));
        expect(result).toHaveLength(1);
    });

    it("filters dangling=true (no RepoTags)", () => {
        const result = applyImageFilters(images, new Map([["dangling", ["true"]]]));
        expect(result).toHaveLength(1);
        expect(result[0].Id).toBe("sha256:img3");
    });

    it("filters dangling=false", () => {
        const result = applyImageFilters(images, new Map([["dangling", ["false"]]]));
        expect(result).toHaveLength(2);
    });

    it("filters by reference (exact)", () => {
        const result = applyImageFilters(images, new Map([["reference", ["nginx:latest"]]]));
        expect(result).toHaveLength(1);
    });

    it("filters by reference (wildcard)", () => {
        const result = applyImageFilters(images, new Map([["reference", ["nginx:*"]]]));
        expect(result).toHaveLength(1);
    });

    it("filters by reference (wildcard matching multiple tags)", () => {
        const result = applyImageFilters(images, new Map([["reference", ["*:*alpine*"]]]));
        expect(result).toHaveLength(1);
        expect(result[0].Id).toBe("sha256:img2");
    });
});

// === applyEventFilters ===

describe("applyEventFilters", () => {
    const event: DockerEvent = {
        Type: "container",
        Action: "start",
        Actor: {
            ID: "abcdef1234",
            Attributes: { name: "mycontainer", "com.docker.compose.project": "mystack" },
        },
        time: 1735689600,
        timeNano: 1735689600000000000,
    };

    it("returns true when no filters", () => {
        expect(applyEventFilters(event, new Map())).toBe(true);
    });

    it("filters by type", () => {
        expect(applyEventFilters(event, new Map([["type", ["container"]]]))).toBe(true);
        expect(applyEventFilters(event, new Map([["type", ["network"]]]))).toBe(false);
    });

    it("filters by event (action)", () => {
        expect(applyEventFilters(event, new Map([["event", ["start"]]]))).toBe(true);
        expect(applyEventFilters(event, new Map([["event", ["stop"]]]))).toBe(false);
    });

    it("filters by container ID", () => {
        expect(applyEventFilters(event, new Map([["container", ["abcdef1234"]]]))).toBe(true);
        expect(applyEventFilters(event, new Map([["container", ["abcdef"]]]))).toBe(true); // prefix
        expect(applyEventFilters(event, new Map([["container", ["zzz"]]]))).toBe(false);
    });

    it("filters by label", () => {
        expect(applyEventFilters(event, new Map([["label", ["com.docker.compose.project=mystack"]]]))).toBe(true);
        expect(applyEventFilters(event, new Map([["label", ["com.docker.compose.project=other"]]]))).toBe(false);
    });

    it("combines type + event (AND)", () => {
        expect(applyEventFilters(event, new Map([["type", ["container"]], ["event", ["start"]]]))).toBe(true);
        expect(applyEventFilters(event, new Map([["type", ["container"]], ["event", ["stop"]]]))).toBe(false);
    });
});

// === Integration: compose project label filter ===

describe("integration: compose project label filter", () => {
    it("filters containers by com.docker.compose.project label", () => {
        const containers = [
            makeContainer({ name: "/mystack-web-1", labels: { "com.docker.compose.project": "mystack" } }),
            makeContainer({ name: "/mystack-db-1", labels: { "com.docker.compose.project": "mystack" } }),
            makeContainer({ name: "/other-app-1", labels: { "com.docker.compose.project": "other" } }),
        ];

        const filters = parseFilters('{"label":["com.docker.compose.project=mystack"]}');
        const result = applyContainerFilters(containers, filters);
        expect(result).toHaveLength(2);
        expect(result.every((c) => c.Config.Labels!["com.docker.compose.project"] === "mystack")).toBe(true);
    });
});
