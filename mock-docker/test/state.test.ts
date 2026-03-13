import { describe, it, expect } from "vitest";
import { MockState } from "../src/state.js";

describe("MockState", () => {
    it("constructor creates empty maps", () => {
        const state = new MockState();
        expect(state.containers.size).toBe(0);
        expect(state.networks.size).toBe(0);
        expect(state.volumes.size).toBe(0);
        expect(state.images.size).toBe(0);
        expect(state.execSessions.size).toBe(0);
    });

    it("clear() empties all maps", () => {
        const state = new MockState();

        // Add dummy entries
        state.containers.set("c1", {} as any);
        state.networks.set("n1", {} as any);
        state.volumes.set("v1", {} as any);
        state.images.set("i1", {} as any);
        state.execSessions.set("e1", {} as any);

        expect(state.containers.size).toBe(1);
        expect(state.networks.size).toBe(1);

        state.clear();

        expect(state.containers.size).toBe(0);
        expect(state.networks.size).toBe(0);
        expect(state.volumes.size).toBe(0);
        expect(state.images.size).toBe(0);
        expect(state.execSessions.size).toBe(0);
    });
});
