import { describe, it, expect } from "vitest";
import { FixedClock, createClock } from "../src/clock.js";

describe("FixedClock", () => {
    it("returns the fixed time", () => {
        const base = new Date("2025-06-15T12:00:00Z");
        const clock = new FixedClock(base);
        expect(clock.now().toISOString()).toBe("2025-06-15T12:00:00.000Z");
    });

    it("advances by tick interval on each call", () => {
        const clock = new FixedClock(new Date("2025-01-01T00:00:00Z"));
        const a = clock.now();
        const b = clock.now();
        expect(b.getTime() - a.getTime()).toBe(1);
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
    it("creates a FixedClock by default", () => {
        const clock = createClock();
        expect(clock).toBeInstanceOf(FixedClock);
    });

    it("uses provided base", () => {
        const clock = createClock({ base: "2025-03-15T10:30:00Z" });
        expect(clock.now().toISOString()).toBe("2025-03-15T10:30:00.000Z");
    });

    it("defaults to 2025-01-01 without base", () => {
        const clock = createClock();
        expect(clock.now().toISOString()).toBe("2025-01-01T00:00:00.000Z");
    });
});
