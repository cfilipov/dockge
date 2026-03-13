import type { ContainerInspect } from "./types.js";
import { deterministicInt, serviceSeed, hashToSeed } from "./deterministic.js";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface TopResponse {
    Titles: string[];
    Processes: string[][];
}

// ---------------------------------------------------------------------------
// Seed derivation
// ---------------------------------------------------------------------------

function containerSeed(container: ContainerInspect): string {
    const labels = container.Config.Labels ?? {};
    const project = labels["com.docker.compose.project"];
    const service = labels["com.docker.compose.service"];
    if (project && service) return serviceSeed(project, service);
    return hashToSeed(["portge-mock-v1", "container", container.Id]);
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const TITLES = ["UID", "PID", "PPID", "C", "STIME", "TTY", "TIME", "CMD", "VSZ", "RSS", "%MEM"];

const WORKER_NAMES = ["worker", "handler", "scheduler", "gc", "logger", "monitor"];

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

export function generateTop(container: ContainerInspect): TopResponse {
    const seed = containerSeed(container);
    const user = container.Config.User || "root";
    const pid1Cmd = buildPid1Command(container);
    const workerCount = deterministicInt(seed + "nprocs", 2, 8);

    const processes: string[][] = [];

    // PID 1: main process
    const pid1Vsz = deterministicInt(seed + "p1vsz", 10000, 500000);
    const pid1Rss = deterministicInt(seed + "p1rss", 5000, 100000);
    const pid1Mem = (pid1Rss / 1024 / 1024 * 100).toFixed(1);
    processes.push([
        user,
        "1",
        "0",
        "0",
        "00:00",
        "?",
        "00:00:01",
        pid1Cmd,
        String(pid1Vsz),
        String(pid1Rss),
        pid1Mem,
    ]);

    // Worker processes
    for (let i = 0; i < workerCount; i++) {
        const workerSeed = seed + "worker" + String(i);
        const pid = deterministicInt(workerSeed + "pid", 10, 999);
        const workerName = WORKER_NAMES[i % WORKER_NAMES.length];
        const vsz = deterministicInt(workerSeed + "vsz", 5000, 200000);
        const rss = deterministicInt(workerSeed + "rss", 2000, 50000);
        const mem = (rss / 1024 / 1024 * 100).toFixed(1);
        const cpuTime = `00:00:${String(deterministicInt(workerSeed + "time", 0, 59)).padStart(2, "0")}`;
        processes.push([
            user,
            String(pid),
            "1",
            "0",
            "00:00",
            "?",
            cpuTime,
            workerName,
            String(vsz),
            String(rss),
            mem,
        ]);
    }

    return { Titles: TITLES, Processes: processes };
}

function buildPid1Command(container: ContainerInspect): string {
    const parts: string[] = [];
    if (container.Path) parts.push(container.Path);
    if (container.Args && container.Args.length > 0) {
        parts.push(...container.Args);
    }
    return parts.join(" ") || "/bin/sh";
}
