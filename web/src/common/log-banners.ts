/**
 * Client-side log buffer for Docker log terminals.
 *
 * Receives binary-framed log entries from the backend, buffers them over a
 * 200ms window, merges start/die banners from the event store, and flushes
 * sorted output to xterm.js.
 */

import type { Terminal } from "@xterm/xterm";
import { useEventStore } from "../stores/eventStore";
import type { DockerResourceEvent } from "../stores/containerStore";
import type { DockgeWebSocket } from "../composables/useSocket";

// ── ANSI banner formatting ──────────────────────────────────────────────────

/** Bold, black text on blue background (#74c2ff) */
const ANSI_BG_BLUE = "\x1b[1;38;2;0;0;0;48;2;116;194;255m";
/** Bold, black text on yellow background (#f8a306) */
const ANSI_BG_YELLOW = "\x1b[1;38;2;0;0;0;48;2;248;163;6m";
const ANSI_RESET = "\x1b[0m";

function startBanner(name: string): string {
    return `\t${ANSI_BG_BLUE} \u25b6 CONTAINER START \u2014 ${name} ${ANSI_RESET}\r\n\r\n`;
}

function stopBanner(name: string): string {
    return `\t${ANSI_BG_YELLOW} \u25fc CONTAINER STOP \u2014 ${name} ${ANSI_RESET}\r\n\r\n`;
}

// ── Log buffer ──────────────────────────────────────────────────────────────

interface TimestampedLine {
    nanos: number;
    text: string;
}

export interface LogBuffer {
    /** Feed raw binary log data into the buffer. */
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
 * Create a log buffer that receives binary-framed log entries, buffers them
 * over a 200ms window, merges start/die banners from the event store by
 * timestamp, and flushes to xterm.js.
 *
 * Binary entry format (per entry within the data payload):
 *   [timestamp_nanos: i64 BE, 8 bytes][message_length: u32 BE, 4 bytes][message: message_length bytes]
 */
export function createLogBuffer(opts: LogBufferOptions): LogBuffer {
    const { terminal, terminalType, containerName, stackName, socket } = opts;
    let buffer: TimestampedLine[] = [];
    let pendingBytes = new Uint8Array(0);
    let flushTimer: ReturnType<typeof setTimeout> | null = null;
    let lastFlushedNano = 0;
    let destroyed = false;

    const eventStore = useEventStore();

    function ensureTimer() {
        if (flushTimer === null && !destroyed) {
            flushTimer = setTimeout(flush, 200);
        }
    }

    function flush() {
        if (destroyed) return;
        flushTimer = null;

        const lines = buffer;
        buffer = [];

        // Determine query range
        const hasLines = lines.length > 0;
        const maxNano = hasLines ? lines[lines.length - 1].nanos : undefined;

        // Query events since last flush
        let matchingEvents: DockerResourceEvent[];
        if (terminalType === "combined" && stackName) {
            matchingEvents = eventStore.forStack(stackName, lastFlushedNano, maxNano);
        } else if (containerName) {
            matchingEvents = eventStore.forContainer(containerName, lastFlushedNano, maxNano);
        } else {
            matchingEvents = [];
        }

        // Convert start/die events to banner lines
        const banners: TimestampedLine[] = matchingEvents
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
    }

    function feed(data: Uint8Array) {
        if (destroyed) return;

        // Append to pending buffer
        const combined = new Uint8Array(pendingBytes.length + data.length);
        combined.set(pendingBytes);
        combined.set(data, pendingBytes.length);
        pendingBytes = combined;

        // Parse complete entries: [ts: i64 BE, 8][len: u32 BE, 4][msg: len bytes]
        while (pendingBytes.length >= 12) {
            const view = new DataView(pendingBytes.buffer, pendingBytes.byteOffset);
            const nanos = Number(view.getBigInt64(0));
            const msgLen = view.getUint32(8);
            if (pendingBytes.length < 12 + msgLen) break; // incomplete entry
            const msgBytes = pendingBytes.slice(12, 12 + msgLen);
            pendingBytes = pendingBytes.slice(12 + msgLen);

            const text = new TextDecoder().decode(msgBytes);
            buffer.push({ nanos, text });
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
