/**
 * Log store: a sorted array of log entries with banner interleaving.
 *
 * Receives raw Docker log data (text with timestamps), parses timestamps,
 * strips them, converts ANSI codes to HTML, and inserts entries in sorted
 * order. Docker start/die events from the event store are inserted as
 * banner entries at the correct position.
 *
 * Framework-agnostic — uses a plain array wrapped in Vue's shallowRef
 * for reactivity. The rendering component (LogView.vue) observes the
 * ref and re-renders via Virtua's virtual scroll.
 */

import { shallowRef, type ShallowRef } from "vue";
import { AnsiUp } from "ansi_up";
import { useEventStore } from "../stores/eventStore";
import type { DockerResourceEvent } from "../stores/containerStore";

// ── Entry types ─────────────────────────────────────────────────────────────

export interface LogLineEntry {
    type: "log";
    nanos: number;
    html: string;
}

export interface BannerEntry {
    type: "banner";
    nanos: number;
    action: "start" | "die";
    name: string;
}

export type LogEntry = LogLineEntry | BannerEntry;

// ── Timestamp parsing ───────────────────────────────────────────────────────

// ── Binary search ───────────────────────────────────────────────────────────

/** Find insertion index for nanos in a sorted array. */
function findInsertIndex(entries: LogEntry[], nanos: number): number {
    let lo = 0;
    let hi = entries.length;
    while (lo < hi) {
        const mid = (lo + hi) >>> 1;
        if (entries[mid].nanos < nanos) {
            lo = mid + 1;
        } else {
            hi = mid;
        }
    }
    return lo;
}

// ── Log store ───────────────────────────────────────────────────────────────

export interface LogStoreOptions {
    terminalType: "container-log" | "container-log-by-name" | "combined";
    containerName?: string;
    stackName?: string;
}

export interface LogStore {
    /** Reactive sorted array of log entries. */
    entries: ShallowRef<LogEntry[]>;
    /** Add a single log line (already parsed by the server). */
    addLine(ts: number, line: string): void;
    /** Clean up event store subscription and pending rAF. */
    destroy(): void;
}

export function createLogStore(opts: LogStoreOptions): LogStore {
    const { terminalType, containerName, stackName } = opts;
    const entries: ShallowRef<LogEntry[]> = shallowRef([]);
    const ansi = new AnsiUp();
    ansi.use_classes = false; // inline styles using default palette
    ansi.escape_html = true;

    let pending: LogEntry[] = [];
    let rafId: number | null = null;
    let destroyed = false;

    const eventStore = useEventStore();

    // ── Banner helpers ────────────────────────────────────────────────────

    function makeBanner(event: DockerResourceEvent): BannerEntry {
        return {
            type: "banner",
            nanos: event.timeNano,
            action: event.action as "start" | "die",
            name: event.serviceName || event.name,
        };
    }

    function isRelevantEvent(event: DockerResourceEvent): boolean {
        if (event.type !== "container") {
            return false;
        }
        if (event.action !== "start" && event.action !== "die") {
            return false;
        }
        if (terminalType === "combined") {
            return event.stackName === stackName;
        }
        return event.name === containerName || event.serviceName === containerName;
    }

    // ── Batch insert ────────────────────────────────────────────────────

    function scheduleBatchInsert() {
        if (rafId === null && !destroyed) {
            rafId = requestAnimationFrame(flushPending);
        }
    }

    let historicalBannersLoaded = false;

    function flushPending() {
        rafId = null;
        if (pending.length === 0 && historicalBannersLoaded) {
            return;
        }

        const arr = [...entries.value];

        // On first flush that contains logs, load historical banners
        // that fall within the log time range. This runs once — after
        // feed() has populated entries, so there's no race with async data.
        if (!historicalBannersLoaded && arr.length + pending.length > 0) {
            let earliest = Infinity;
            let latest = -Infinity;
            for (const e of arr) {
                if (e.type === "log") {
                    if (e.nanos < earliest) {
                        earliest = e.nanos;
                    }
                    if (e.nanos > latest) {
                        latest = e.nanos;
                    }
                }
            }
            for (const e of pending) {
                if (e.type === "log") {
                    if (e.nanos < earliest) {
                        earliest = e.nanos;
                    }
                    if (e.nanos > latest) {
                        latest = e.nanos;
                    }
                }
            }

            if (earliest !== Infinity) {
                const historical = (terminalType === "combined" && stackName)
                    ? eventStore.forStack(stackName)
                    : containerName
                        ? eventStore.forContainer(containerName)
                        : { events: [], endIndex: 0 };

                // Only lock once the event store actually had events to check.
                // If bulkLoad hasn't populated it yet, retry on the next flush.
                if (historical.events.length > 0) {
                    historicalBannersLoaded = true;
                }

                for (const event of historical.events) {
                    if (!isRelevantEvent(event)) {
                        continue;
                    }
                    // 60s lower pad: the start event fires before the container
                    // emits its first log, so it's always before `earliest`.
                    // Old-cycle banners from hours/days ago are still filtered.
                    const lo = earliest - 60_000_000_000;
                    if (event.timeNano < lo || event.timeNano > latest) {
                        continue;
                    }
                    pending.push(makeBanner(event));
                }
            }
        }

        if (pending.length === 0) {
            return;
        }

        const batch = pending;
        pending = [];

        for (const entry of batch) {
            const idx = findInsertIndex(arr, entry.nanos);
            arr.splice(idx, 0, entry);
        }
        entries.value = arr;
    }

    // ── Event store subscription (live events) ──────────────────────────

    const unsubscribe = eventStore.onInsert((event: DockerResourceEvent) => {
        if (destroyed || !isRelevantEvent(event)) {
            return;
        }
        pending.push(makeBanner(event));
        scheduleBatchInsert();
    });

    // ── Add line (from server JSON) ─────────────────────────────────────

    function addLine(ts: number, line: string) {
        if (destroyed) {
            return;
        }
        const html = ansi.ansi_to_html(line);
        pending.push({ type: "log", nanos: ts, html });
        scheduleBatchInsert();
    }

    // ── Destroy ─────────────────────────────────────────────────────────

    function destroy() {
        destroyed = true;
        unsubscribe();
        if (rafId !== null) {
            cancelAnimationFrame(rafId);
            rafId = null;
        }
    }

    return { entries, addLine, destroy };
}
