import { describe, it, expect } from "vitest";
import {
    parseStackMockConfig,
    parseGlobalMockConfig,
} from "../src/mock-config.js";

describe("parseStackMockConfig", () => {
    it("returns defaults for null input", () => {
        const config = parseStackMockConfig(null);
        expect(config.deployed).toBe(true);
        expect(config.untracked).toBe(false);
        expect(config.services).toEqual({});
    });

    it("returns defaults for empty string", () => {
        const config = parseStackMockConfig("");
        expect(config.deployed).toBe(true);
        expect(config.untracked).toBe(false);
    });

    it("parses deployed: false", () => {
        const config = parseStackMockConfig("deployed: false");
        expect(config.deployed).toBe(false);
    });

    it("parses untracked: true", () => {
        const config = parseStackMockConfig("untracked: true");
        expect(config.untracked).toBe(true);
    });

    it("parses service overrides with snake_case keys", () => {
        const yaml = `
services:
  web:
    state: exited
    exit_code: 137
    health: unhealthy
    needs_recreation: true
`;
        const config = parseStackMockConfig(yaml);
        const web = config.services.web;
        expect(web).toBeDefined();
        expect(web.state).toBe("exited");
        expect(web.exitCode).toBe(137);
        expect(web.health).toBe("unhealthy");
        expect(web.needsRecreation).toBe(true);
    });

    it("parses multiple service overrides", () => {
        const yaml = `
services:
  web:
    state: running
  db:
    state: exited
    exit_code: 0
  worker:
    health: starting
`;
        const config = parseStackMockConfig(yaml);
        expect(Object.keys(config.services)).toEqual(["web", "db", "worker"]);
        expect(config.services.web.state).toBe("running");
        expect(config.services.db.state).toBe("exited");
        expect(config.services.db.exitCode).toBe(0);
        expect(config.services.worker.health).toBe("starting");
    });

    it("deployed defaults to true when not specified", () => {
        const yaml = `
services:
  web:
    state: running
`;
        const config = parseStackMockConfig(yaml);
        expect(config.deployed).toBe(true);
    });

    it("handles partial service override (only some fields)", () => {
        const yaml = `
services:
  web:
    health: none
`;
        const config = parseStackMockConfig(yaml);
        expect(config.services.web.health).toBe("none");
        expect(config.services.web.state).toBeUndefined();
        expect(config.services.web.exitCode).toBeUndefined();
    });
});

describe("parseGlobalMockConfig", () => {
    it("returns empty defaults for null input", () => {
        const config = parseGlobalMockConfig(null);
        expect(config.networks).toEqual({});
        expect(config.volumes).toEqual({});
    });

    it("returns empty defaults for empty string", () => {
        const config = parseGlobalMockConfig("");
        expect(config.networks).toEqual({});
        expect(config.volumes).toEqual({});
    });

    it("parses networks with driver, subnet, gateway", () => {
        const yaml = `
networks:
  proxy:
    driver: bridge
    subnet: "172.30.0.0/16"
    gateway: "172.30.0.1"
  internal:
    driver: bridge
    internal: true
`;
        const config = parseGlobalMockConfig(yaml);
        expect(config.networks.proxy).toEqual({
            driver: "bridge",
            subnet: "172.30.0.0/16",
            gateway: "172.30.0.1",
            internal: undefined,
        });
        expect(config.networks.internal).toEqual({
            driver: "bridge",
            subnet: undefined,
            gateway: undefined,
            internal: true,
        });
    });

    it("parses volumes with driver", () => {
        const yaml = `
volumes:
  shared-data:
    driver: local
  nfs-vol:
    driver: nfs
`;
        const config = parseGlobalMockConfig(yaml);
        expect(config.volumes["shared-data"]).toEqual({ driver: "local" });
        expect(config.volumes["nfs-vol"]).toEqual({ driver: "nfs" });
    });

    it("defaults network driver to bridge", () => {
        const yaml = `
networks:
  mynet: {}
`;
        const config = parseGlobalMockConfig(yaml);
        expect(config.networks.mynet.driver).toBe("bridge");
    });

    it("defaults volume driver to local", () => {
        const yaml = `
volumes:
  myvol: {}
`;
        const config = parseGlobalMockConfig(yaml);
        expect(config.volumes.myvol.driver).toBe("local");
    });

    it("parses images with update_available", () => {
        const yaml = `
images:
  nginx:latest:
    update_available: true
  redis:7:
    update_available: true
  postgres:16: {}
`;
        const config = parseGlobalMockConfig(yaml);
        expect(config.updateImages.has("nginx:latest")).toBe(true);
        expect(config.updateImages.has("redis:7")).toBe(true);
        expect(config.updateImages.has("postgres:16")).toBe(false);
    });

    it("updateImages defaults to empty set", () => {
        const config = parseGlobalMockConfig(null);
        expect(config.updateImages.size).toBe(0);
    });
});
