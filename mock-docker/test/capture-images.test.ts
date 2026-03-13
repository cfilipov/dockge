import { describe, it, expect } from "vitest";
import { parseImageRef, parseWwwAuthenticate } from "../scripts/oci-registry.js";

describe("parseImageRef", () => {
    it("parses bare name (nginx → docker hub library)", () => {
        const r = parseImageRef("nginx");
        expect(r.registry).toBe("registry-1.docker.io");
        expect(r.name).toBe("library/nginx");
        expect(r.tag).toBe("latest");
        expect(r.digest).toBeUndefined();
    });

    it("parses name with tag", () => {
        const r = parseImageRef("nginx:1.25");
        expect(r.registry).toBe("registry-1.docker.io");
        expect(r.name).toBe("library/nginx");
        expect(r.tag).toBe("1.25");
    });

    it("parses ghcr.io ref", () => {
        const r = parseImageRef("ghcr.io/org/repo:v1");
        expect(r.registry).toBe("ghcr.io");
        expect(r.name).toBe("org/repo");
        expect(r.tag).toBe("v1");
    });

    it("parses registry with port", () => {
        const r = parseImageRef("registry:5000/image:tag");
        expect(r.registry).toBe("registry:5000");
        expect(r.name).toBe("image");
        expect(r.tag).toBe("tag");
    });

    it("parses digest ref", () => {
        const r = parseImageRef("image@sha256:abc123");
        expect(r.registry).toBe("registry-1.docker.io");
        expect(r.name).toBe("library/image");
        expect(r.tag).toBe("latest");
        expect(r.digest).toBe("sha256:abc123");
    });

    it("parses Docker Hub org/repo", () => {
        const r = parseImageRef("myorg/myrepo:v2");
        expect(r.registry).toBe("registry-1.docker.io");
        expect(r.name).toBe("myorg/myrepo");
        expect(r.tag).toBe("v2");
    });

    it("parses localhost registry", () => {
        const r = parseImageRef("localhost/myimage:dev");
        expect(r.registry).toBe("localhost");
        expect(r.name).toBe("myimage");
        expect(r.tag).toBe("dev");
    });
});

describe("parseWwwAuthenticate", () => {
    it("parses standard Docker Hub format", () => {
        const result = parseWwwAuthenticate(
            'Bearer realm="https://auth.docker.io/token",service="registry.docker.io",scope="repository:library/nginx:pull"',
        );
        expect(result.realm).toBe("https://auth.docker.io/token");
        expect(result.service).toBe("registry.docker.io");
        expect(result.scope).toBe("repository:library/nginx:pull");
    });

    it("parses ghcr.io format", () => {
        const result = parseWwwAuthenticate(
            'Bearer realm="https://ghcr.io/token",service="ghcr.io",scope="repository:org/repo:pull"',
        );
        expect(result.realm).toBe("https://ghcr.io/token");
        expect(result.service).toBe("ghcr.io");
    });

    it("handles missing fields gracefully", () => {
        const result = parseWwwAuthenticate('Bearer realm="https://example.com/auth"');
        expect(result.realm).toBe("https://example.com/auth");
        expect(result.service).toBe("");
        expect(result.scope).toBe("");
    });
});

// Integration test — only runs when TEST_NETWORK env var is set
const describeNetwork = process.env.TEST_NETWORK ? describe : describe.skip;

describeNetwork("fetchImageInspect (network)", () => {
    it("fetches alpine:latest", async () => {
        const { fetchImageInspect } = await import("../scripts/oci-registry.js");
        const result = await fetchImageInspect("alpine:latest");
        expect(result).not.toBeNull();
        expect(result!.Architecture).toBe("amd64");
        expect(result!.Os).toBe("linux");
        expect(result!.Config.Cmd).toBeDefined();
        expect(result!.RootFS.Layers.length).toBeGreaterThan(0);
    }, 30_000);
});
