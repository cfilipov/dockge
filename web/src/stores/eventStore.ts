import { defineStore } from "pinia";
import { ref } from "vue";
import type { DockerResourceEvent } from "./containerStore";

/**
 * Pinia store for Docker events — a sorted array by timeNano.
 * Used by the frontend to insert banners into log terminals.
 */
export const useEventStore = defineStore("events", () => {
    const events = ref<DockerResourceEvent[]>([]);
    const insertCallbacks = new Set<(event: DockerResourceEvent) => void>();

    /** Composite dedup key for an event. */
    function eventKey(e: DockerResourceEvent): string {
        return `${e.timeNano}:${e.type}:${e.action}:${e.id}`;
    }

    /** Insert a single event in sorted order, deduplicating by composite key. */
    function insert(event: DockerResourceEvent) {
        const key = eventKey(event);
        // Binary search for insertion position
        let lo = 0;
        let hi = events.value.length;
        while (lo < hi) {
            const mid = (lo + hi) >>> 1;
            if (events.value[mid].timeNano < event.timeNano) {
                lo = mid + 1;
            } else {
                hi = mid;
            }
        }
        // Check for duplicate at insertion position (and neighbors)
        for (let i = Math.max(0, lo - 1); i <= Math.min(events.value.length - 1, lo + 1); i++) {
            if (eventKey(events.value[i]) === key) {
                return; // Duplicate — skip
            }
        }
        events.value.splice(lo, 0, event);

        // Notify subscribers
        for (const cb of insertCallbacks) cb(event);
    }

    /** Bulk load events from afterLogin payload. Replaces existing events. */
    function bulkLoad(incoming: DockerResourceEvent[]) {
        // Sort and dedup
        const sorted = [...incoming].sort((a, b) => a.timeNano - b.timeNano);
        const deduped: DockerResourceEvent[] = [];
        const seen = new Set<string>();
        for (const e of sorted) {
            const key = eventKey(e);
            if (!seen.has(key)) {
                seen.add(key);
                deduped.push(e);
            }
        }
        events.value = deduped;
    }

    /**
     * Get events for a specific container within a time range.
     * `since` is exclusive (timeNano > since), `until` is inclusive (timeNano <= until).
     * When `until` is undefined, no upper bound.
     */
    function forContainer(
        containerName: string,
        since?: number,
        until?: number,
    ): DockerResourceEvent[] {
        return events.value.filter((e) => {
            if (e.type !== "container") return false;
            if (e.name !== containerName && e.serviceName !== containerName) return false;
            if (since !== undefined && e.timeNano <= since) return false;
            if (until !== undefined && e.timeNano > until) return false;
            return true;
        });
    }

    /**
     * Get events for any container in a stack within a time range.
     * `since` is exclusive (timeNano > since), `until` is inclusive (timeNano <= until).
     * When `until` is undefined, no upper bound.
     */
    function forStack(
        stackName: string,
        since?: number,
        until?: number,
    ): DockerResourceEvent[] {
        return events.value.filter((e) => {
            if (e.type !== "container") return false;
            if (e.stackName !== stackName) return false;
            if (since !== undefined && e.timeNano <= since) return false;
            if (until !== undefined && e.timeNano > until) return false;
            return true;
        });
    }

    /** Register a callback for every insert. Returns an unsubscribe function. */
    function onInsert(cb: (e: DockerResourceEvent) => void): () => void {
        insertCallbacks.add(cb);
        return () => insertCallbacks.delete(cb);
    }

    return {
        events,
        insert,
        bulkLoad,
        forContainer,
        forStack,
        onInsert,
    };
});
