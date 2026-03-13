import { describe, it, expect } from "vitest";
import { EventEmitter, makeEvent } from "../src/events.js";
import { FixedClock } from "../src/clock.js";
import type { DockerEvent } from "../src/list-types.js";

describe("EventEmitter", () => {
    it("emits to all subscribers", () => {
        const emitter = new EventEmitter();
        const events1: DockerEvent[] = [];
        const events2: DockerEvent[] = [];

        emitter.subscribe((e) => events1.push(e));
        emitter.subscribe((e) => events2.push(e));

        const clock = new FixedClock(new Date("2025-01-01T00:00:00Z"));
        const event = makeEvent(clock, "container", "start", "abc123", { name: "test" });
        emitter.emit(event);

        expect(events1).toHaveLength(1);
        expect(events2).toHaveLength(1);
        expect(events1[0]).toBe(event);
        expect(events2[0]).toBe(event);
    });

    it("unsubscribed listeners don't receive events", () => {
        const emitter = new EventEmitter();
        const events: DockerEvent[] = [];
        const listener = (e: DockerEvent) => events.push(e);

        emitter.subscribe(listener);
        emitter.unsubscribe(listener);

        const clock = new FixedClock(new Date("2025-01-01T00:00:00Z"));
        emitter.emit(makeEvent(clock, "container", "start", "abc123"));

        expect(events).toHaveLength(0);
    });

    it("multiple events arrive in order", () => {
        const emitter = new EventEmitter();
        const events: DockerEvent[] = [];
        emitter.subscribe((e) => events.push(e));

        const clock = new FixedClock(new Date("2025-01-01T00:00:00Z"));
        emitter.emit(makeEvent(clock, "container", "kill", "abc", { signal: "SIGTERM" }));
        emitter.emit(makeEvent(clock, "container", "die", "abc", { exitCode: "0" }));
        emitter.emit(makeEvent(clock, "container", "stop", "abc"));

        expect(events.map((e) => e.Action)).toEqual(["kill", "die", "stop"]);
    });

    it("Set semantics: same listener added twice gets event once", () => {
        const emitter = new EventEmitter();
        let count = 0;
        const listener = () => { count++; };

        emitter.subscribe(listener);
        emitter.subscribe(listener); // duplicate

        const clock = new FixedClock(new Date("2025-01-01T00:00:00Z"));
        emitter.emit(makeEvent(clock, "container", "start", "abc"));

        expect(count).toBe(1);
    });
});

describe("makeEvent", () => {
    it("builds event with correct time fields", () => {
        const clock = new FixedClock(new Date("2025-06-15T12:30:00Z"));
        const event = makeEvent(clock, "container", "start", "abc123", { name: "test" });

        expect(event.Type).toBe("container");
        expect(event.Action).toBe("start");
        expect(event.Actor.ID).toBe("abc123");
        expect(event.Actor.Attributes).toEqual({ name: "test" });

        const expectedMs = new Date("2025-06-15T12:30:00Z").getTime();
        expect(event.time).toBe(Math.floor(expectedMs / 1000));
        expect(event.timeNano).toBe(expectedMs * 1_000_000);
    });

    it("defaults attributes to empty object", () => {
        const clock = new FixedClock(new Date("2025-01-01T00:00:00Z"));
        const event = makeEvent(clock, "network", "create", "net1");
        expect(event.Actor.Attributes).toEqual({});
    });
});
