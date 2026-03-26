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

/**
 * Parse an RFC3339Nano timestamp from the beginning of a log line.
 * Returns nanoseconds since epoch, or null if not parseable.
 */
export function parseTimestampNanos(line: string): number | null {
    const spaceIdx = line.indexOf(" ", 0);
    if (spaceIdx === -1 || spaceIdx > 35) {
        return null;
    }

    const ts = line.substring(0, spaceIdx);
    if (ts.length < 19 || !/^\d{4}-/.test(ts)) {
        return null;
    }

    const year = parseInt(ts.substring(0, 4), 10);
    const month = parseInt(ts.substring(5, 7), 10);
    const day = parseInt(ts.substring(8, 10), 10);
    const hour = parseInt(ts.substring(11, 13), 10);
    const min = parseInt(ts.substring(14, 16), 10);
    const sec = parseInt(ts.substring(17, 19), 10);

    if (isNaN(year) || isNaN(month) || isNaN(day) || isNaN(hour) || isNaN(min) || isNaN(sec)) {
        return null;
    }

    const y = month <= 2 ? year - 1 : year;
    const era = Math.floor(y >= 0 ? y : y - 399) / 400 | 0;
    const yoe = y - era * 400;
    const m = month;
    const doy = Math.floor((153 * (m > 2 ? m - 3 : m + 9) + 2) / 5) + day - 1;
    const doe = yoe * 365 + Math.floor(yoe / 4) - Math.floor(yoe / 100) + doy;
    const days = era * 146097 + doe - 719468;
    const secs = days * 86400 + hour * 3600 + min * 60 + sec;

    let nanos = 0;
    if (ts.length > 19 && ts[19] === ".") {
        let fracEnd = 20;
        while (fracEnd < ts.length && ts[fracEnd] >= "0" && ts[fracEnd] <= "9") {
            fracEnd++;
        }
        const fracStr = ts.substring(20, Math.min(fracEnd, 29));
        if (fracStr.length > 0) {
            nanos = parseInt(fracStr.padEnd(9, "0"), 10);
        }
    }

    return secs * 1_000_000_000 + nanos;
}

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
    onWarning?: (message: string) => void;
}

export interface LogStore {
    /** Reactive sorted array of log entries. */
    entries: ShallowRef<LogEntry[]>;
    /** Feed raw Docker log data (Uint8Array with timestamps). */
    feed(data: Uint8Array): void;
    /** Clean up event store subscription and pending rAF. */
    destroy(): void;
}

export function createLogStore(opts: LogStoreOptions): LogStore {
    const { terminalType, containerName, stackName, onWarning } = opts;
    const entries: ShallowRef<LogEntry[]> = shallowRef([]);
    const decoder = new TextDecoder();
    const ansi = new AnsiUp();
    ansi.use_classes = false; // inline styles using default palette
    ansi.escape_html = true;

    let pending: LogEntry[] = [];
    let rafId: number | null = null;
    let destroyed = false;

    const eventStore = useEventStore();

    // ── Batch insert ────────────────────────────────────────────────────

    function scheduleBatchInsert() {
        if (rafId === null && !destroyed) {
            rafId = requestAnimationFrame(flushPending);
        }
    }

    function flushPending() {
        rafId = null;
        if (pending.length === 0) {
            return;
        }

        const batch = pending;
        pending = [];

        // Copy-on-write: create a new array so Vue's reactivity (and VList's
        // data prop comparison) detects the change. The copy is O(n) but only
        // happens once per animation frame.
        const arr = [...entries.value];
        for (const entry of batch) {
            const idx = findInsertIndex(arr, entry.nanos);
            arr.splice(idx, 0, entry);
        }
        entries.value = arr;
    }

    // ── Stale banner check ──────────────────────────────────────────────

    /**
     * Check if a banner's timestamp falls within the log range.
     * Only allow banners between (earliest log - 5s) and (latest log + 5s).
     * This prevents historical events from past stop/start cycles
     * (whose logs are no longer in the tail window) from showing as banners.
     */
    function isBannerInRange(nanos: number): boolean {
        const arr = entries.value;
        if (arr.length === 0) {
            // No logs yet — allow the banner (it's a live event, logs will follow)
            return true;
        }
        // Find earliest and latest log entry nanos
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
        if (earliest === Infinity) {
            return true; // No log entries, only banners — allow
        }
        // Allow banners within 5s of the log range
        return nanos >= earliest - 5_000_000_000 && nanos <= latest + 5_000_000_000;
    }

    // ── Event store subscription ────────────────────────────────────────

    const unsubscribe = eventStore.onInsert((event: DockerResourceEvent) => {
        if (destroyed) {
            return;
        }
        if (event.type !== "container") {
            return;
        }
        if (event.action !== "start" && event.action !== "die") {
            return;
        }
        if (terminalType === "combined") {
            if (event.stackName !== stackName) {
                return;
            }
        } else {
            if (event.name !== containerName && event.serviceName !== containerName) {
                return;
            }
        }

        // Live events are always allowed — they're happening now
        const banner: BannerEntry = {
            type: "banner",
            nanos: event.timeNano,
            action: event.action as "start" | "die",
            name: event.serviceName || event.name,
        };
        pending.push(banner);
        scheduleBatchInsert();
    });

    // Also check for historical events already in the store on creation.
    // Use a microtask so the first feed() has a chance to set the log range.
    queueMicrotask(() => {
        if (destroyed) {
            return;
        }
        const result = terminalType === "combined" && stackName
            ? eventStore.forStack(stackName)
            : containerName
                ? eventStore.forContainer(containerName)
                : { events: [], endIndex: 0 };

        for (const event of result.events) {
            if (event.action !== "start" && event.action !== "die") {
                continue;
            }
            if (!isBannerInRange(event.timeNano)) {
                continue;
            }
            // Check if this banner already exists (from onInsert)
            const exists = entries.value.some(
                e => e.type === "banner" && e.nanos === event.timeNano && e.action === event.action
            );
            if (exists) {
                continue;
            }
            pending.push({
                type: "banner",
                nanos: event.timeNano,
                action: event.action as "start" | "die",
                name: event.serviceName || event.name,
            });
        }
        if (pending.length > 0) {
            scheduleBatchInsert();
        }
    });

    // ── Feed ────────────────────────────────────────────────────────────

    function feed(data: Uint8Array) {
        if (destroyed) {
            return;
        }

        const text = decoder.decode(data, { stream: true });
        const lineTexts = text.split("\n");

        for (const lineText of lineTexts) {
            if (lineText.length === 0) {
                continue;
            }
            // Strip trailing \r (from normalize_newlines \r\n conversion)
            const clean = lineText.endsWith("\r") ? lineText.slice(0, -1) : lineText;
            if (clean.length === 0) {
                continue;
            }

            const nanos = parseTimestampNanos(clean);
            if (nanos !== null) {
                // Strip Docker timestamp prefix, convert ANSI to HTML
                const spaceIdx = clean.indexOf(" ");
                const raw = spaceIdx !== -1 ? clean.substring(spaceIdx + 1) : clean;
                const html = ansi.ansi_to_html(raw);
                pending.push({ type: "log", nanos, html });
            }
            // Non-timestamped data (e.g. cursor-show) is ignored in the
            // component model — xterm.js handled these as raw terminal
            // commands, but the Vue component doesn't need them.
        }

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

    return { entries, feed, destroy };
}
