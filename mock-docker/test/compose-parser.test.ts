import { describe, it, expect } from "vitest";
import { readFileSync } from "node:fs";
import { join } from "node:path";
import {
    parseCompose,
    parsePorts,
    parseVolumeMounts,
    parseHealthcheck,
    parseEnvironment,
    parseUlimits,
    parseDevices,
    parseDuration,
    parseDurationSeconds,
    parseByteSize,
    findComposeFile,
} from "../src/compose-parser.js";

describe("parsePorts", () => {
    it("parses simple published:target", () => {
        const ports = parsePorts(["8080:80"]);
        expect(ports).toEqual([
            { target: 80, published: 8080, protocol: "tcp", hostIp: "" },
        ]);
    });

    it("parses with /udp protocol", () => {
        const ports = parsePorts(["8080:80/udp"]);
        expect(ports).toEqual([
            { target: 80, published: 8080, protocol: "udp", hostIp: "" },
        ]);
    });

    it("parses with host IP", () => {
        const ports = parsePorts(["127.0.0.1:8080:80"]);
        expect(ports).toEqual([
            { target: 80, published: 8080, protocol: "tcp", hostIp: "127.0.0.1" },
        ]);
    });

    it("parses expose-only (target only)", () => {
        const ports = parsePorts(["80"]);
        expect(ports).toEqual([
            { target: 80, published: undefined, protocol: "tcp", hostIp: "" },
        ]);
    });

    it("parses numeric entries", () => {
        const ports = parsePorts([3000]);
        expect(ports).toEqual([
            { target: 3000, protocol: "tcp", hostIp: "" },
        ]);
    });

    it("parses long syntax", () => {
        const ports = parsePorts([
            { target: 80, published: 8080, protocol: "udp", host_ip: "0.0.0.0" },
        ]);
        expect(ports).toEqual([
            { target: 80, published: 8080, protocol: "udp", hostIp: "0.0.0.0" },
        ]);
    });

    it("handles /tcp protocol suffix", () => {
        const ports = parsePorts(["9090:9090/tcp"]);
        expect(ports[0].protocol).toBe("tcp");
    });
});

describe("parseVolumeMounts", () => {
    it("parses named volume", () => {
        const mounts = parseVolumeMounts(["mydata:/data"]);
        expect(mounts).toEqual([
            { type: "volume", source: "mydata", target: "/data", readOnly: false },
        ]);
    });

    it("parses bind mount with relative path", () => {
        const mounts = parseVolumeMounts(["./config:/etc/config"]);
        expect(mounts).toEqual([
            { type: "bind", source: "./config", target: "/etc/config", readOnly: false },
        ]);
    });

    it("parses bind mount with absolute path", () => {
        const mounts = parseVolumeMounts(["/host/path:/container/path:ro"]);
        expect(mounts).toEqual([
            { type: "bind", source: "/host/path", target: "/container/path", readOnly: true },
        ]);
    });

    it("parses anonymous volume (just container path)", () => {
        const mounts = parseVolumeMounts(["/data"]);
        expect(mounts).toEqual([
            { type: "volume", source: "", target: "/data", readOnly: false },
        ]);
    });

    it("parses long syntax with bind propagation", () => {
        const mounts = parseVolumeMounts([
            { type: "bind", source: "./data", target: "/app/data", read_only: true, bind: { propagation: "rslave" } },
        ]);
        expect(mounts).toEqual([
            { type: "bind", source: "./data", target: "/app/data", readOnly: true, bindPropagation: "rslave" },
        ]);
    });
});

describe("parseHealthcheck", () => {
    it("parses CMD array", () => {
        const hc = parseHealthcheck({
            test: ["CMD", "curl", "-f", "http://localhost"],
            interval: "30s",
            timeout: "10s",
            retries: 3,
        });
        expect(hc!.test).toEqual(["CMD", "curl", "-f", "http://localhost"]);
        expect(hc!.interval).toBe(30_000_000_000);
        expect(hc!.timeout).toBe(10_000_000_000);
        expect(hc!.retries).toBe(3);
    });

    it("parses CMD-SHELL string", () => {
        const hc = parseHealthcheck({
            test: "curl -f http://localhost || exit 1",
        });
        expect(hc!.test).toEqual(["CMD-SHELL", "curl -f http://localhost || exit 1"]);
    });

    it("parses CMD-SHELL array", () => {
        const hc = parseHealthcheck({
            test: ["CMD-SHELL", "pg_isready -U postgres"],
        });
        expect(hc!.test).toEqual(["CMD-SHELL", "pg_isready -U postgres"]);
    });

    it("parses disable: true", () => {
        const hc = parseHealthcheck({ disable: true });
        expect(hc!.test).toEqual(["NONE"]);
        expect(hc!.disable).toBe(true);
    });
});

describe("parseEnvironment", () => {
    it("parses map syntax", () => {
        const env = parseEnvironment({ KEY: "value", NUM: 42 });
        expect(env).toEqual({ KEY: "value", NUM: "42" });
    });

    it("parses list syntax", () => {
        const env = parseEnvironment(["KEY=value", "EMPTY="]);
        expect(env).toEqual({ KEY: "value", EMPTY: "" });
    });

    it("handles null values in map", () => {
        const env = parseEnvironment({ KEY: null });
        expect(env).toEqual({ KEY: "" });
    });
});

describe("parseUlimits", () => {
    it("parses simple number values", () => {
        const ulimits = parseUlimits({ nofile: 65535 });
        expect(ulimits).toEqual([{ name: "nofile", soft: 65535, hard: 65535 }]);
    });

    it("parses soft/hard objects", () => {
        const ulimits = parseUlimits({ nofile: { soft: 1024, hard: 65535 } });
        expect(ulimits).toEqual([{ name: "nofile", soft: 1024, hard: 65535 }]);
    });
});

describe("parseDevices", () => {
    it("parses device strings", () => {
        const devices = parseDevices(["/dev/sda:/dev/xvdc:rwm"]);
        expect(devices).toEqual([{ host: "/dev/sda", container: "/dev/xvdc", permissions: "rwm" }]);
    });

    it("defaults permissions to rwm", () => {
        const devices = parseDevices(["/dev/sda:/dev/xvdc"]);
        expect(devices![0].permissions).toBe("rwm");
    });
});

describe("parseDuration", () => {
    it("parses seconds", () => {
        expect(parseDuration("30s")).toBe(30_000_000_000);
    });

    it("parses compound duration", () => {
        expect(parseDuration("1m30s")).toBe(90_000_000_000);
    });

    it("parses milliseconds", () => {
        expect(parseDuration("500ms")).toBe(500_000_000);
    });

    it("parses plain number as seconds", () => {
        expect(parseDuration(30)).toBe(30_000_000_000);
    });
});

describe("parseDurationSeconds", () => {
    it("returns seconds", () => {
        expect(parseDurationSeconds("10s")).toBe(10);
    });
});

describe("parseByteSize", () => {
    it("parses megabytes", () => {
        expect(parseByteSize("256M")).toBe(268435456);
    });

    it("parses gigabytes", () => {
        expect(parseByteSize("1G")).toBe(1073741824);
    });

    it("parses kilobytes", () => {
        expect(parseByteSize("512k")).toBe(524288);
    });

    it("returns number as-is", () => {
        expect(parseByteSize(1024)).toBe(1024);
    });
});

describe("parseCompose — full fixtures", () => {
    const fixturesDir = join(import.meta.dirname, "fixtures/stacks");

    it("parses basic single-service compose", () => {
        const yaml = readFileSync(join(fixturesDir, "basic/compose.yaml"), "utf-8");
        const parsed = parseCompose(yaml);
        expect(Object.keys(parsed.services)).toEqual(["web"]);
        expect(parsed.services.web.image).toBe("nginx:latest");
        expect(parsed.services.web.ports).toHaveLength(1);
        expect(parsed.services.web.ports[0].target).toBe(80);
        expect(parsed.services.web.ports[0].published).toBe(8080);
        expect(parsed.services.web.environment.APP_ENV).toBe("production");
    });

    it("parses multi-network compose", () => {
        const yaml = readFileSync(join(fixturesDir, "multi-net/compose.yaml"), "utf-8");
        const parsed = parseCompose(yaml);
        expect(Object.keys(parsed.services)).toEqual(["web", "api"]);
        expect(Object.keys(parsed.networks)).toEqual(["frontend", "backend"]);
        expect(parsed.services.api.networks).toHaveLength(2);
    });

    it("parses host-mode compose", () => {
        const yaml = readFileSync(join(fixturesDir, "host-mode/compose.yaml"), "utf-8");
        const parsed = parseCompose(yaml);
        expect(parsed.services.app.networkMode).toBe("host");
        expect(parsed.services.app.networks).toHaveLength(0);
    });

    it("parses with-volumes compose", () => {
        const yaml = readFileSync(join(fixturesDir, "with-volumes/compose.yaml"), "utf-8");
        const parsed = parseCompose(yaml);
        expect(parsed.services.db.volumes).toHaveLength(2);
        const namedVol = parsed.services.db.volumes.find((v) => v.type === "volume");
        expect(namedVol).toBeDefined();
        expect(namedVol!.source).toBe("pgdata");
        const bindMount = parsed.services.db.volumes.find((v) => v.type === "bind");
        expect(bindMount).toBeDefined();
        expect(Object.keys(parsed.volumes)).toContain("pgdata");
    });

    it("parses env-file compose", () => {
        const yaml = readFileSync(join(fixturesDir, "env-file/compose.yaml"), "utf-8");
        const parsed = parseCompose(yaml);
        expect(parsed.services.app.envFile).toEqual([".env"]);
    });

    it("handles empty sections gracefully", () => {
        const parsed = parseCompose(`
services:
  app:
    image: alpine
networks: {}
volumes: {}
`);
        expect(Object.keys(parsed.services)).toEqual(["app"]);
        expect(Object.keys(parsed.networks)).toEqual([]);
        expect(Object.keys(parsed.volumes)).toEqual([]);
    });

    it("handles build-only service (no image)", () => {
        const parsed = parseCompose(`
services:
  app:
    build: .
    ports:
      - "3000:3000"
`);
        expect(parsed.services.app.image).toBeUndefined();
        expect(parsed.services.app.build).toEqual({ context: "." });
    });

    it("parses network_mode variations", () => {
        const parsed = parseCompose(`
services:
  a:
    image: alpine
    network_mode: host
  b:
    image: alpine
    network_mode: none
  c:
    image: alpine
    network_mode: "service:a"
  d:
    image: alpine
    network_mode: bridge
`);
        expect(parsed.services.a.networkMode).toBe("host");
        expect(parsed.services.b.networkMode).toBe("none");
        expect(parsed.services.c.networkMode).toBe("service:a");
        expect(parsed.services.d.networkMode).toBe("bridge");
    });
});

describe("findComposeFile", () => {
    const fixturesDir = join(import.meta.dirname, "fixtures/stacks");

    it("finds compose.yaml", () => {
        const result = findComposeFile(join(fixturesDir, "basic"));
        expect(result).toContain("compose.yaml");
    });

    it("returns null for empty dir", () => {
        const result = findComposeFile("/tmp");
        expect(result).toBeNull();
    });
});
