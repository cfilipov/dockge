import { describe, it, expect } from "vitest";
import {
    hashToSeed,
    deterministicId,
    deterministicMac,
    deterministicIp,
    deterministicTimestamp,
    deterministicInt,
    projectSeed,
    serviceSeed,
    networkSeed,
    imageSeed,
} from "../src/deterministic.js";

describe("hashToSeed", () => {
    it("returns consistent output for the same input", () => {
        const a = hashToSeed(["hello", "world"]);
        const b = hashToSeed(["hello", "world"]);
        expect(a).toBe(b);
        expect(a).toHaveLength(64);
    });

    it("returns different output for different input", () => {
        const a = hashToSeed(["hello", "world"]);
        const b = hashToSeed(["world", "hello"]);
        expect(a).not.toBe(b);
    });
});

describe("deterministicId", () => {
    it("returns 64 hex chars", () => {
        const id = deterministicId("seed", "purpose");
        expect(id).toMatch(/^[0-9a-f]{64}$/);
    });

    it("is idempotent", () => {
        const a = deterministicId("seed", "container");
        const b = deterministicId("seed", "container");
        expect(a).toBe(b);
    });

    it("differs for different seeds", () => {
        const a = deterministicId("seed1", "container");
        const b = deterministicId("seed2", "container");
        expect(a).not.toBe(b);
    });

    it("differs for different purposes", () => {
        const a = deterministicId("seed", "container");
        const b = deterministicId("seed", "network");
        expect(a).not.toBe(b);
    });
});

describe("deterministicMac", () => {
    it("produces 02:42:xx:xx:xx:xx format", () => {
        const mac = deterministicMac("test-seed");
        expect(mac).toMatch(/^02:42:[0-9a-f]{2}:[0-9a-f]{2}:[0-9a-f]{2}:[0-9a-f]{2}$/);
    });

    it("is idempotent", () => {
        const a = deterministicMac("test-seed");
        const b = deterministicMac("test-seed");
        expect(a).toBe(b);
    });

    it("differs for different seeds", () => {
        const a = deterministicMac("seed1");
        const b = deterministicMac("seed2");
        expect(a).not.toBe(b);
    });
});

describe("deterministicIp", () => {
    it("returns IP within /24 subnet", () => {
        const ip = deterministicIp("test", "192.168.1.0/24");
        const parts = ip.split(".").map(Number);
        expect(parts[0]).toBe(192);
        expect(parts[1]).toBe(168);
        expect(parts[2]).toBe(1);
        expect(parts[3]).toBeGreaterThanOrEqual(1);
        expect(parts[3]).toBeLessThanOrEqual(254);
    });

    it("returns IP within /16 subnet", () => {
        const ip = deterministicIp("test", "172.18.0.0/16");
        const parts = ip.split(".").map(Number);
        expect(parts[0]).toBe(172);
        expect(parts[1]).toBe(18);
        // host portion can span octets 3 and 4
        const host = (parts[2] << 8) | parts[3];
        expect(host).toBeGreaterThanOrEqual(1);
        expect(host).toBeLessThanOrEqual(65534);
    });

    it("is idempotent", () => {
        const a = deterministicIp("seed", "10.0.0.0/8");
        const b = deterministicIp("seed", "10.0.0.0/8");
        expect(a).toBe(b);
    });
});

describe("deterministicTimestamp", () => {
    it("returns valid ISO 8601", () => {
        const ts = deterministicTimestamp("seed", "2025-01-01T00:00:00Z");
        const date = new Date(ts);
        expect(date.getTime()).not.toBeNaN();
        expect(ts).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}/);
    });

    it("is within 24h of base", () => {
        const base = "2025-01-01T00:00:00Z";
        const ts = deterministicTimestamp("seed", base);
        const baseMs = new Date(base).getTime();
        const tsMs = new Date(ts).getTime();
        expect(tsMs).toBeGreaterThanOrEqual(baseMs);
        expect(tsMs).toBeLessThan(baseMs + 86400 * 1000);
    });

    it("is idempotent", () => {
        const a = deterministicTimestamp("seed", "2025-06-01T00:00:00Z");
        const b = deterministicTimestamp("seed", "2025-06-01T00:00:00Z");
        expect(a).toBe(b);
    });
});

describe("deterministicInt", () => {
    it("returns value in [min, max]", () => {
        for (let i = 0; i < 20; i++) {
            const val = deterministicInt(`seed-${i}`, 10, 20);
            expect(val).toBeGreaterThanOrEqual(10);
            expect(val).toBeLessThanOrEqual(20);
        }
    });

    it("returns integer values", () => {
        const val = deterministicInt("seed", 0, 1000);
        expect(Number.isInteger(val)).toBe(true);
    });

    it("is idempotent", () => {
        const a = deterministicInt("seed", 0, 100);
        const b = deterministicInt("seed", 0, 100);
        expect(a).toBe(b);
    });
});

describe("seed hierarchy", () => {
    it("produces distinct seeds per project", () => {
        const a = projectSeed("app1");
        const b = projectSeed("app2");
        expect(a).not.toBe(b);
        expect(a).toHaveLength(64);
    });

    it("produces distinct seeds per service", () => {
        const a = serviceSeed("app", "web");
        const b = serviceSeed("app", "db");
        expect(a).not.toBe(b);
    });

    it("produces distinct seeds per network", () => {
        const a = networkSeed("frontend");
        const b = networkSeed("backend");
        expect(a).not.toBe(b);
    });

    it("produces distinct seeds per image", () => {
        const a = imageSeed("nginx:latest");
        const b = imageSeed("redis:7");
        expect(a).not.toBe(b);
    });

    it("project and service seeds differ even for same name", () => {
        const proj = projectSeed("web");
        const svc = serviceSeed("web", "web");
        expect(proj).not.toBe(svc);
    });
});
