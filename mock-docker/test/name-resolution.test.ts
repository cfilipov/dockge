import { describe, it, expect } from "vitest";
import { resolveByIdOrName } from "../src/name-resolution.js";

interface TestItem {
    id: string;
    name: string;
}

function makeItems(...entries: [string, string][]): Map<string, TestItem> {
    const map = new Map<string, TestItem>();
    for (const [id, name] of entries) {
        map.set(id, { id, name });
    }
    return map;
}

const getName = (item: TestItem) => item.name;
const getId = (item: TestItem) => item.id;

describe("resolveByIdOrName", () => {
    it("resolves by full ID", () => {
        const items = makeItems(["abc123def456", "/my-container"]);
        const result = resolveByIdOrName(items, "abc123def456", getName, getId);
        expect(result).toEqual({ found: { id: "abc123def456", name: "/my-container" } });
    });

    it("resolves by short ID prefix (3+ chars)", () => {
        const items = makeItems(["abc123def456", "/my-container"]);
        const result = resolveByIdOrName(items, "abc", getName, getId);
        expect(result).toEqual({ found: { id: "abc123def456", name: "/my-container" } });
    });

    it("resolves by name", () => {
        const items = makeItems(["abc123def456", "/my-container"]);
        const result = resolveByIdOrName(items, "my-container", getName, getId);
        expect(result).toEqual({ found: { id: "abc123def456", name: "/my-container" } });
    });

    it("resolves by name with leading slash", () => {
        const items = makeItems(["abc123def456", "/my-container"]);
        const result = resolveByIdOrName(items, "/my-container", getName, getId);
        expect(result).toEqual({ found: { id: "abc123def456", name: "/my-container" } });
    });

    it("resolves names without leading slash in the data", () => {
        const items = makeItems(["abc123", "my-network"]);
        const result = resolveByIdOrName(items, "my-network", getName, getId);
        expect(result).toEqual({ found: { id: "abc123", name: "my-network" } });
    });

    it("returns error for ambiguous prefix", () => {
        const items = makeItems(
            ["abc123", "/container-a"],
            ["abc456", "/container-b"],
        );
        const result = resolveByIdOrName(items, "abc", getName, getId);
        expect(result).toEqual({ error: 'multiple items match prefix "abc"' });
    });

    it("returns error when not found", () => {
        const items = makeItems(["abc123", "/my-container"]);
        const result = resolveByIdOrName(items, "xyz", getName, getId);
        expect(result).toEqual({ error: 'no item found with id or name "xyz"' });
    });

    it("prefers full ID over prefix match", () => {
        // "abc" is both a full ID key and a prefix of "abcdef"
        const items = makeItems(
            ["abc", "/short"],
            ["abcdef", "/long"],
        );
        const result = resolveByIdOrName(items, "abc", getName, getId);
        expect(result).toEqual({ found: { id: "abc", name: "/short" } });
    });

    it("does not prefix-match with fewer than 3 chars", () => {
        const items = makeItems(["abc123", "/container"]);
        const result = resolveByIdOrName(items, "ab", getName, getId);
        // "ab" is not a full ID and too short for prefix, falls through to name match
        expect(result).toEqual({ error: 'no item found with id or name "ab"' });
    });
});
