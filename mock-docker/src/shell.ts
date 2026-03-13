import type { ContainerInspect } from "./types.js";
import type { Clock } from "./clock.js";
import { deterministicInt, hashToSeed, serviceSeed } from "./deterministic.js";
import { generateTop } from "./top.js";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface ShellSession {
    cwd: string;
    container: ContainerInspect;
    clock: Clock;
}

// ---------------------------------------------------------------------------
// Seed derivation (same pattern as top.ts)
// ---------------------------------------------------------------------------

function containerSeed(container: ContainerInspect): string {
    const labels = container.Config.Labels ?? {};
    const project = labels["com.docker.compose.project"];
    const service = labels["com.docker.compose.service"];
    if (project && service) return serviceSeed(project, service);
    return hashToSeed(["portge-mock-v1", "container", container.Id]);
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

export function createShellSession(container: ContainerInspect, clock: Clock): ShellSession {
    return {
        cwd: container.Config.WorkingDir || "/",
        container,
        clock,
    };
}

export function getPrompt(session: ShellSession): string {
    const user = session.container.Config.User || "root";
    const hostname = session.container.Config.Hostname || session.container.Id.slice(0, 12);
    const suffix = user === "root" ? "#" : "$";
    return `${user}@${hostname}:${session.cwd}${suffix} `;
}

/**
 * Process a shell command. Returns the output string, or null for `exit`.
 */
export function processCommand(session: ShellSession, input: string): string | null {
    const trimmed = input.trim();
    if (!trimmed) return "";

    const parts = splitCommand(trimmed);
    const cmd = parts[0];
    const args = parts.slice(1);

    switch (cmd) {
        case "exit": return null;
        case "echo": return args.join(" ");
        case "pwd": return session.cwd;
        case "whoami": return session.container.Config.User || "root";
        case "hostname": return session.container.Config.Hostname || session.container.Id.slice(0, 12);
        case "id": return formatId(session.container);
        case "uname": return formatUname(session.container, args);
        case "date": return session.clock.now().toString();
        case "uptime": return formatUptime(session);
        case "env": return formatEnv(session.container);
        case "printenv": return formatPrintenv(session.container, args);
        case "cat": return formatCat(session, args);
        case "ls": return formatLs(session, args);
        case "ps": return formatPs(session.container);
        case "cd": return handleCd(session, args);
        case "free": return formatFree(session.container);
        case "df": return formatDf(session.container);
        default: return `bash: ${cmd}: command not found`;
    }
}

// ---------------------------------------------------------------------------
// Command implementations
// ---------------------------------------------------------------------------

function splitCommand(input: string): string[] {
    const parts: string[] = [];
    let current = "";
    let inSingle = false;
    let inDouble = false;

    for (let i = 0; i < input.length; i++) {
        const ch = input[i];
        if (ch === "'" && !inDouble) {
            inSingle = !inSingle;
        } else if (ch === '"' && !inSingle) {
            inDouble = !inDouble;
        } else if (ch === " " && !inSingle && !inDouble) {
            if (current) { parts.push(current); current = ""; }
        } else {
            current += ch;
        }
    }
    if (current) parts.push(current);
    return parts;
}

function formatId(container: ContainerInspect): string {
    const user = container.Config.User || "root";
    if (user === "root") {
        return "uid=0(root) gid=0(root) groups=0(root)";
    }
    const seed = containerSeed(container);
    const uid = deterministicInt(seed + "uid", 1000, 65534);
    return `uid=${uid}(${user}) gid=${uid}(${user}) groups=${uid}(${user})`;
}

function formatUname(container: ContainerInspect, args: string[]): string {
    const hostname = container.Config.Hostname || container.Id.slice(0, 12);
    if (args.includes("-a") || args.includes("--all")) {
        return `Linux ${hostname} 5.15.0-mock #1 SMP x86_64 GNU/Linux`;
    }
    if (args.includes("-r") || args.includes("--kernel-release")) {
        return "5.15.0-mock";
    }
    if (args.includes("-n") || args.includes("--nodename")) {
        return hostname;
    }
    return "Linux";
}

function formatUptime(session: ShellSession): string {
    const startedAt = session.container.State.StartedAt;
    if (!startedAt || startedAt === "0001-01-01T00:00:00Z") {
        return " 00:00:00 up 0 min,  0 users,  load average: 0.00, 0.00, 0.00";
    }
    const startMs = new Date(startedAt).getTime();
    const nowMs = session.clock.now().getTime();
    const uptimeSec = Math.max(0, Math.floor((nowMs - startMs) / 1000));
    const hours = Math.floor(uptimeSec / 3600);
    const mins = Math.floor((uptimeSec % 3600) / 60);

    const now = session.clock.now();
    const timeStr = `${String(now.getUTCHours()).padStart(2, "0")}:${String(now.getUTCMinutes()).padStart(2, "0")}:${String(now.getUTCSeconds()).padStart(2, "0")}`;

    let uptimeStr: string;
    if (hours > 0) {
        uptimeStr = `${hours}:${String(mins).padStart(2, "0")}`;
    } else {
        uptimeStr = `${mins} min`;
    }

    return ` ${timeStr} up ${uptimeStr},  1 user,  load average: 0.08, 0.03, 0.01`;
}

function formatEnv(container: ContainerInspect): string {
    const env = container.Config.Env || [];
    return env.join("\n");
}

function formatPrintenv(container: ContainerInspect, args: string[]): string {
    const env = container.Config.Env || [];
    if (args.length === 0) return env.join("\n");

    const varName = args[0];
    for (const entry of env) {
        const eqIdx = entry.indexOf("=");
        if (eqIdx !== -1 && entry.slice(0, eqIdx) === varName) {
            return entry.slice(eqIdx + 1);
        }
    }
    return "";
}

function formatCat(session: ShellSession, args: string[]): string {
    if (args.length === 0) return "";
    const file = args[0];
    const container = session.container;
    const hostname = container.Config.Hostname || container.Id.slice(0, 12);

    switch (file) {
        case "/etc/hostname":
            return hostname;
        case "/etc/hosts": {
            const lines = ["127.0.0.1\tlocalhost", "::1\tlocalhost ip6-localhost ip6-loopback"];
            // Add container's own IP
            const networks = container.NetworkSettings.Networks;
            if (networks) {
                for (const ep of Object.values(networks)) {
                    if (ep.IPAddress) {
                        lines.push(`${ep.IPAddress}\t${hostname}`);
                        break;
                    }
                }
            }
            // Add extra hosts
            const extraHosts = container.HostConfig.ExtraHosts;
            if (extraHosts) {
                for (const h of extraHosts) {
                    const colonIdx = h.indexOf(":");
                    if (colonIdx !== -1) {
                        lines.push(`${h.slice(colonIdx + 1)}\t${h.slice(0, colonIdx)}`);
                    }
                }
            }
            return lines.join("\n");
        }
        case "/etc/resolv.conf": {
            const dns = container.HostConfig.Dns;
            if (dns && dns.length > 0) {
                return dns.map((d) => `nameserver ${d}`).join("\n");
            }
            return "nameserver 127.0.0.11\noptions ndots:0";
        }
        default:
            return `cat: ${file}: No such file or directory`;
    }
}

function formatLs(session: ShellSession, args: string[]): string {
    const longFormat = args.includes("-la") || args.includes("-l") || args.includes("-al");
    const pathArg = args.find((a) => !a.startsWith("-"));
    const targetPath = pathArg || session.cwd;

    const seed = containerSeed(session.container);
    const dirSeed = seed + "ls" + targetPath;

    // Generate deterministic directory contents based on path
    const entries = getDirectoryEntries(targetPath, dirSeed);

    if (longFormat) {
        const lines = [`total ${entries.length * 4}`];
        for (const entry of entries) {
            const size = deterministicInt(dirSeed + entry.name + "size", 0, 65536);
            const perm = entry.isDir ? "drwxr-xr-x" : "-rw-r--r--";
            const nlink = entry.isDir ? "2" : "1";
            lines.push(`${perm} ${nlink} root root ${String(size).padStart(5)} Jan  1 00:00 ${entry.name}`);
        }
        return lines.join("\n");
    }

    return entries.map((e) => e.name).join("  ");
}

interface DirEntry {
    name: string;
    isDir: boolean;
}

function getDirectoryEntries(path: string, seed: string): DirEntry[] {
    // Common filesystem paths with deterministic contents
    const known: Record<string, DirEntry[]> = {
        "/": [
            { name: "bin", isDir: true },
            { name: "dev", isDir: true },
            { name: "etc", isDir: true },
            { name: "home", isDir: true },
            { name: "lib", isDir: true },
            { name: "proc", isDir: true },
            { name: "root", isDir: true },
            { name: "sys", isDir: true },
            { name: "tmp", isDir: true },
            { name: "usr", isDir: true },
            { name: "var", isDir: true },
        ],
        "/etc": [
            { name: "hostname", isDir: false },
            { name: "hosts", isDir: false },
            { name: "resolv.conf", isDir: false },
            { name: "passwd", isDir: false },
            { name: "group", isDir: false },
        ],
        "/tmp": [],
        "/root": [],
    };

    if (path in known) return known[path];

    // Generate deterministic entries for unknown paths
    const count = deterministicInt(seed + "count", 1, 5);
    const names = ["data", "config", "logs", "cache", "run"];
    const entries: DirEntry[] = [];
    for (let i = 0; i < count; i++) {
        entries.push({ name: names[i % names.length], isDir: i < 2 });
    }
    return entries;
}

function formatPs(container: ContainerInspect): string {
    const top = generateTop(container);
    const lines: string[] = [];
    lines.push("USER         PID %CPU %MEM    VSZ   RSS TTY      STAT START   TIME COMMAND");
    for (const proc of top.Processes) {
        // proc: [UID, PID, PPID, C, STIME, TTY, TIME, CMD, VSZ, RSS, %MEM]
        const user = proc[0].padEnd(8);
        const pid = proc[1].padStart(5);
        const vsz = proc[8].padStart(6);
        const rss = proc[9].padStart(5);
        const mem = proc[10].padStart(4);
        const cmd = proc[7];
        lines.push(`${user} ${pid}  0.0 ${mem} ${vsz} ${rss} ?        Ss   00:00   ${proc[6]} ${cmd}`);
    }
    return lines.join("\n");
}

function handleCd(session: ShellSession, args: string[]): string {
    if (args.length === 0) {
        session.cwd = "/";
        return "";
    }
    const target = args[0];
    if (target === "..") {
        const parts = session.cwd.split("/").filter(Boolean);
        parts.pop();
        session.cwd = "/" + parts.join("/");
    } else if (target.startsWith("/")) {
        session.cwd = target;
    } else {
        const base = session.cwd === "/" ? "" : session.cwd;
        session.cwd = base + "/" + target;
    }
    return "";
}

function formatFree(container: ContainerInspect): string {
    const memLimit = container.HostConfig.Memory;
    const totalMb = memLimit ? Math.floor(memLimit / (1024 * 1024)) : 2048;
    const usedMb = Math.floor(totalMb * 0.35);
    const freeMb = totalMb - usedMb;
    const buffMb = Math.floor(totalMb * 0.20);
    const availMb = freeMb + buffMb;
    const swapTotal = Math.floor(totalMb / 2);
    const swapUsed = 0;
    const swapFree = swapTotal;

    const lines = [
        "              total        used        free      shared  buff/cache   available",
        `Mem:       ${String(totalMb).padStart(8)}    ${String(usedMb).padStart(8)}    ${String(freeMb).padStart(8)}           0    ${String(buffMb).padStart(8)}    ${String(availMb).padStart(8)}`,
        `Swap:      ${String(swapTotal).padStart(8)}    ${String(swapUsed).padStart(8)}    ${String(swapFree).padStart(8)}`,
    ];
    return lines.join("\n");
}

function formatDf(container: ContainerInspect): string {
    const seed = containerSeed(container);
    const usedG = deterministicInt(seed + "df-used", 1, 30);
    const totalG = 50;
    const availG = totalG - usedG;
    const usePct = Math.floor((usedG / totalG) * 100);

    const lines = [
        "Filesystem      Size  Used Avail Use% Mounted on",
        `overlay          ${totalG}G   ${String(usedG).padStart(2)}G   ${String(availG).padStart(2)}G  ${String(usePct).padStart(2)}% /`,
        "tmpfs            64M     0   64M   0% /dev",
        "shm              64M     0   64M   0% /dev/shm",
    ];
    return lines.join("\n");
}
