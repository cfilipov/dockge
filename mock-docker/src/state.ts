import { EventEmitter as NodeEventEmitter } from "node:events";
import type {
    ContainerInspect,
    NetworkInspect,
    VolumeInspect,
    ImageInspect,
    ExecInspect,
} from "./types.js";
import type { DockerEvent } from "./list-types.js";
import type { LogTemplates } from "./log-templates.js";

export interface LogEntry {
    ts: number;   // milliseconds since epoch
    line: string;
}

const LOG_BUFFER_CAP = 100;

export class MockState {
    containers: Map<string, ContainerInspect>;
    networks: Map<string, NetworkInspect>;
    volumes: Map<string, VolumeInspect>;
    images: Map<string, ImageInspect>;
    execSessions: Map<string, ExecInspect>;
    logTemplates: LogTemplates | null;
    /** Image refs that have updates available (from global .mock.yaml). */
    updateImages: Set<string>;
    private statsCounters: Map<string, number>;
    /** Per-container log buffer (capped at LOG_BUFFER_CAP). */
    logBuffers: Map<string, LogEntry[]>;
    /** Emits "log" events with container ID when new lines are appended. */
    logEmitter: NodeEventEmitter;
    /** Active heartbeat intervals per container ID. */
    heartbeatIntervals: Map<string, ReturnType<typeof setInterval>>;
    /** Deterministic event history — built during init, appended on mutations. */
    eventHistory: DockerEvent[];

    constructor() {
        this.containers = new Map();
        this.networks = new Map();
        this.volumes = new Map();
        this.images = new Map();
        this.execSessions = new Map();
        this.logTemplates = null;
        this.updateImages = new Set();
        this.statsCounters = new Map();
        this.logBuffers = new Map();
        this.logEmitter = new NodeEventEmitter();
        this.logEmitter.setMaxListeners(200);
        this.heartbeatIntervals = new Map();
        this.eventHistory = [];
    }

    /** Returns the next stats counter for a container, incrementing it for future calls. */
    nextStatsCounter(containerId: string): number {
        const current = this.statsCounters.get(containerId) ?? 0;
        this.statsCounters.set(containerId, current + 1);
        return current;
    }

    clear(): void {
        this.containers.clear();
        this.networks.clear();
        this.volumes.clear();
        this.images.clear();
        this.execSessions.clear();
        this.statsCounters.clear();
        this.logBuffers.clear();
        this.logEmitter.removeAllListeners();
        for (const interval of this.heartbeatIntervals.values()) {
            clearInterval(interval);
        }
        this.heartbeatIntervals.clear();
        this.eventHistory = [];
        // logTemplates is intentionally NOT cleared — it's loaded from source, not runtime state
    }
}

/** Append a log line to a container's buffer and emit notification. */
export function appendLog(state: MockState, containerId: string, ts: number, line: string): void {
    let buf = state.logBuffers.get(containerId);
    if (!buf) {
        buf = [];
        state.logBuffers.set(containerId, buf);
    }
    buf.push({ ts, line });
    if (buf.length > LOG_BUFFER_CAP) {
        buf.splice(0, buf.length - LOG_BUFFER_CAP);
    }
    state.logEmitter.emit("log", containerId);
}
