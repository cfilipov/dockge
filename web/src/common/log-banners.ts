/**
 * Client-side log buffer for Docker log terminals.
 *
 * Buffers incoming log text over a 200ms window, parses Docker timestamps,
 * merges start/die banners from the event store, and flushes to xterm.js.
 *
 * The backend sends raw Docker log data with timestamps prepended
 * (e.g., "2025-01-15T00:00:00.000Z alpine container started\r\n").
 * This module parses and strips those timestamps for banner interleaving,
 * displaying only the service's own log message.
 */

import type { Terminal } from "@xterm/xterm";
import { useEventStore } from "../stores/eventStore";
import type { DockerResourceEvent } from "../stores/containerStore";
import type { EventQueryResult } from "../stores/eventStore";
import type { DockgeWebSocket } from "../composables/useSocket";

// ── ANSI banner formatting ──────────────────────────────────────────────────

/** Bold, black text on blue background (#74c2ff) */
const ANSI_BG_BLUE = "\x1b[1;38;2;0;0;0;48;2;116;194;255m";
/** Bold, black text on yellow background (#f8a306) */
const ANSI_BG_YELLOW = "\x1b[1;38;2;0;0;0;48;2;248;163;6m";
const ANSI_RESET = "\x1b[0m";

function startBanner(name: string): string {
    return `${ANSI_BG_BLUE}[CONTAINER START] ${name}${ANSI_RESET}\r\n`;
}

function stopBanner(name: string): string {
    return `${ANSI_BG_YELLOW}[CONTAINER STOP] ${name}${ANSI_RESET}\r\n`;
}

// ── Timestamp parsing ───────────────────────────────────────────────────────

/**
 * Parse an RFC3339Nano timestamp from the beginning of a log line.
 * Docker log lines with timestamps=true start with: "2025-01-15T00:00:05.300000000Z rest"
 * Returns nanoseconds since epoch, or null if not parseable.
 */
export function parseTimestampNanos(line: string): number | null {
    // Timestamps are at most ~35 chars, look for the first space
    const spaceIdx = line.indexOf(" ", 0);
    if (spaceIdx === -1 || spaceIdx > 35) return null;

    const ts = line.substring(0, spaceIdx);
    if (ts.length < 19 || !/^\d{4}-/.test(ts)) return null;

    const year = parseInt(ts.substring(0, 4), 10);
    const month = parseInt(ts.substring(5, 7), 10);
    const day = parseInt(ts.substring(8, 10), 10);
    const hour = parseInt(ts.substring(11, 13), 10);
    const min = parseInt(ts.substring(14, 16), 10);
    const sec = parseInt(ts.substring(17, 19), 10);

    if (isNaN(year) || isNaN(month) || isNaN(day) || isNaN(hour) || isNaN(min) || isNaN(sec)) {
        return null;
    }

    // Convert to Unix seconds using the same algorithm as the backend
    const y = month <= 2 ? year - 1 : year;
    const era = Math.floor(y >= 0 ? y : y - 399) / 400 | 0;
    const yoe = y - era * 400;
    const m = month;
    const doy = Math.floor((153 * (m > 2 ? m - 3 : m + 9) + 2) / 5) + day - 1;
    const doe = yoe * 365 + Math.floor(yoe / 4) - Math.floor(yoe / 100) + doy;
    const days = era * 146097 + doe - 719468;
    const secs = days * 86400 + hour * 3600 + min * 60 + sec;

    // Parse fractional seconds
    let nanos = 0;
    if (ts.length > 19 && ts[19] === ".") {
        let fracEnd = 20;
        while (fracEnd < ts.length && ts[fracEnd] >= "0" && ts[fracEnd] <= "9") {
            fracEnd++;
        }
        const fracStr = ts.substring(20, Math.min(fracEnd, 29)); // up to 9 digits
        if (fracStr.length > 0) {
            nanos = parseInt(fracStr.padEnd(9, "0"), 10);
        }
    }

    // JavaScript can represent integers up to 2^53, which is enough for
    // nanosecond timestamps through year 2255.
    return secs * 1_000_000_000 + nanos;
}

// ── Log buffer ──────────────────────────────────────────────────────────────

interface TimestampedLine {
    nanos: number;
    text: string;
}

export interface LogBuffer {
    /** Feed raw log data (binary) into the buffer. */
    feed(data: Uint8Array): void;
    /** Stop the buffer and flush remaining data. */
    destroy(): void;
}

export interface LogBufferOptions {
    /** Terminal to write to */
    terminal: Terminal;
    /** Type of log terminal */
    terminalType: "container-log" | "container-log-by-name" | "combined";
    /** Container name (for single-container types) */
    containerName?: string;
    /** Stack name (for combined type) */
    stackName?: string;
    /** WebSocket for sending clientWarning messages */
    socket?: DockgeWebSocket;
}

/**
 * Create a log buffer that receives raw Docker log data (text with timestamps),
 * buffers over a 200ms window, parses timestamps, strips them for display,
 * merges start/die banners from the event store, and flushes to xterm.js.
 */
export function createLogBuffer(opts: LogBufferOptions): LogBuffer {
    const { terminal, terminalType, containerName, stackName, socket } = opts;
    const decoder = new TextDecoder();
    let buffer: TimestampedLine[] = [];
    let rawBuffer: Uint8Array[] = [];
    let flushTimer: ReturnType<typeof setTimeout> | null = null;
    let lastFlushedNano = 0;
    let lastEventIndex = 0;
    let destroyed = false;

    const eventStore = useEventStore();

    function ensureTimer() {
        if (flushTimer === null && !destroyed) {
            flushTimer = setTimeout(flush, 200);
        }
    }

    /** Query events from the store using the tracked index for efficient scanning. */
    function queryEvents(since: number, until?: number): EventQueryResult {
        if (terminalType === "combined" && stackName) {
            return eventStore.forStack(stackName, since, until, lastEventIndex);
        } else if (containerName) {
            return eventStore.forContainer(containerName, since, until, lastEventIndex);
        }
        return { events: [], endIndex: lastEventIndex };
    }

    function flush() {
        if (destroyed) return;
        flushTimer = null;

        const lines = buffer;
        buffer = [];
        const raw = rawBuffer;
        rawBuffer = [];

        // Write non-timestamped data directly (e.g. cursor-show sequences)
        for (const chunk of raw) {
            terminal.write(chunk);
        }

        // On the first flush with log lines, initialize the watermark to just
        // before the oldest log line. This prevents historical events (whose
        // corresponding logs were pushed out by Docker's tail limit) from
        // generating banners with no surrounding log context.
        const hasLines = lines.length > 0;
        if (lastFlushedNano === 0 && hasLines) {
            lastFlushedNano = lines[0].nanos - 5_000_000_000; // 5s before first log
        }

        // Determine query range
        const maxNano = hasLines ? lines[lines.length - 1].nanos : undefined;

        // Query events since last flush
        const queryResult = queryEvents(lastFlushedNano, maxNano);
        lastEventIndex = queryResult.endIndex;

        // Convert start/die events to banner lines
        const banners: TimestampedLine[] = queryResult.events
            .filter(e => e.action === "start" || e.action === "die")
            .map(e => ({
                nanos: e.timeNano,
                text: e.action === "start"
                    ? startBanner(e.serviceName || e.name)
                    : stopBanner(e.serviceName || e.name),
            }));

        // Merge and sort (log lines may be out of order for combined logs)
        const merged = [...lines, ...banners].sort((a, b) => a.nanos - b.nanos);

        // Late-arrival detection
        for (const line of merged) {
            if (line.nanos < lastFlushedNano && socket) {
                const gap = lastFlushedNano - line.nanos;
                socket.emit("clientWarning", `late log line: ts=${line.nanos}, lastFlushed=${lastFlushedNano}, gap=${gap}ns`);
            }
        }

        // Write to terminal, advance watermark
        for (const line of merged) {
            terminal.write(line.text);
        }
        if (merged.length > 0) {
            const maxTs = merged[merged.length - 1].nanos;
            if (maxTs > lastFlushedNano) {
                lastFlushedNano = maxTs;
            }
        }

        // Post-flush: check if there are still start/die events beyond the watermark.
        // This handles the case where a die event's timestamp slightly exceeds the
        // last log line's timestamp within the same flush window.
        // Do NOT advance lastEventIndex here — the follow-up flush needs to find
        // these events via its own main query.
        const remaining = queryEvents(lastFlushedNano);
        if (remaining.events.some(e => e.action === "start" || e.action === "die")) {
            ensureTimer();
        }
    }

    function feed(data: Uint8Array) {
        if (destroyed) return;

        // Decode text and split into lines, parsing Docker timestamps
        const text = decoder.decode(data, { stream: true });
        const lineTexts = text.split("\n");

        for (const lineText of lineTexts) {
            if (lineText.length === 0) continue;
            // Strip trailing \r (from normalize_newlines \r\n conversion)
            const clean = lineText.endsWith("\r") ? lineText.slice(0, -1) : lineText;
            if (clean.length === 0) continue;

            const nanos = parseTimestampNanos(clean);
            if (nanos !== null) {
                // Strip Docker timestamp prefix, keep the rest as display text
                const spaceIdx = clean.indexOf(" ");
                const content = spaceIdx !== -1 ? clean.substring(spaceIdx + 1) : clean;
                buffer.push({ nanos, text: content + "\r\n" });
            } else {
                // Non-timestamped data (e.g. cursor-show) — buffer as raw
                rawBuffer.push(new TextEncoder().encode(clean));
            }
        }

        ensureTimer();
    }

    // Subscribe to event store for start/die events that should trigger a flush
    const unsubscribe = eventStore.onInsert((event: DockerResourceEvent) => {
        if (destroyed) return;
        if (event.type !== "container") return;
        if (event.action !== "start" && event.action !== "die") return;
        if (terminalType === "combined") {
            if (event.stackName !== stackName) return;
        } else {
            if (event.name !== containerName && event.serviceName !== containerName) return;
        }
        // Just ensure the timer is running — flush will query the store
        ensureTimer();
    });

    function destroy() {
        destroyed = true;
        unsubscribe();
        if (flushTimer !== null) {
            clearTimeout(flushTimer);
            flushTimer = null;
        }
        // Final flush
        flush();
    }

    return { feed, destroy };
}
