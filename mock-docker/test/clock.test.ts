import { describe, it, expect } from "vitest";
import { RealClock, FixedClock, createClock } from "../src/clock.js";

describe("RealClock", () => {
    it("returns current-ish time", () => {
        const clock = new RealClock();
        const before = Date.now();
        const now = clock.now().getTime();
        const after = Date.now();
        expect(now).toBeGreaterThanOrEqual(before);
        expect(now).toBeLessThanOrEqual(after);
    });
});

describe("FixedClock", () => {
    it("returns the fixed time", () => {
        const base = new Date("2025-06-15T12:00:00Z");
        const clock = new FixedClock(base);
        expect(clock.now().toISOString()).toBe("2025-06-15T12:00:00.000Z");
    });

    it("returns the same time on multiple calls", () => {
        const clock = new FixedClock(new Date("2025-01-01T00:00:00Z"));
        const a = clock.now();
        const b = clock.now();
        expect(a.getTime()).toBe(b.getTime());
    });

    it("advance moves time forward", () => {
        const clock = new FixedClock(new Date("2025-01-01T00:00:00Z"));
        clock.advance(5000); // 5 seconds
        expect(clock.now().toISOString()).toBe("2025-01-01T00:00:05.000Z");
    });

    it("advance accumulates", () => {
        const clock = new FixedClock(new Date("2025-01-01T00:00:00Z"));
        clock.advance(1000);
        clock.advance(2000);
        expect(clock.now().toISOString()).toBe("2025-01-01T00:00:03.000Z");
    });

    it("returns a copy, not the internal reference", () => {
        const clock = new FixedClock(new Date("2025-01-01T00:00:00Z"));
        const t1 = clock.now();
        t1.setFullYear(2000); // mutate the returned Date
        expect(clock.now().getFullYear()).toBe(2025); // internal unchanged
    });
});

describe("createClock", () => {
    it("creates a RealClock by default", () => {
        const clock = createClock();
        expect(clock).toBeInstanceOf(RealClock);
    });

    it("creates a FixedClock when fixed=true", () => {
        const clock = createClock({ fixed: true });
        expect(clock).toBeInstanceOf(FixedClock);
    });

    it("uses provided base for FixedClock", () => {
        const clock = createClock({ fixed: true, base: "2025-03-15T10:30:00Z" });
        expect(clock.now().toISOString()).toBe("2025-03-15T10:30:00.000Z");
    });

    it("defaults to 2025-01-01 when fixed without base", () => {
        const clock = createClock({ fixed: true });
        expect(clock.now().toISOString()).toBe("2025-01-01T00:00:00.000Z");
    });
});
