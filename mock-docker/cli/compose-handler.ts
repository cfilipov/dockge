import { readFileSync } from "node:fs";
import { basename, resolve, dirname } from "node:path";
import { requestJSON } from "./socket-client.js";
import {
    renderProgress,
    composeUpTasks,
    composeStopTasks,
    composeDownTasks,
    composeRestartTasks,
    composePullTasks,
    composePauseTasks,
    composeUnpauseTasks,
} from "./tty-output.js";
import { parseCompose, findComposeFile } from "../src/compose-parser.js";
import type { ParsedCompose, ParsedService, ParsedPort } from "../src/compose-parser.js";

// ---------------------------------------------------------------------------
// Compose flag parsing
// ---------------------------------------------------------------------------

interface ComposeFlags {
    projectName: string;
    envFiles: string[];
    composeFile: string;
    subcmd: string;
    restArgs: string[];
}

function parseComposeFlags(args: string[]): ComposeFlags {
    const envFiles: string[] = [];
    let projectName = "";
    let composeFile = "";
    let idx = 0;

    while (idx < args.length) {
        if (args[idx] === "--env-file" && idx + 1 < args.length) {
            envFiles.push(args[idx + 1]);
            idx += 2;
            continue;
        }
        if ((args[idx] === "-p" || args[idx] === "--project-name") && idx + 1 < args.length) {
            projectName = args[idx + 1];
            idx += 2;
            continue;
        }
        // -f / --file with space-separated value
        if ((args[idx] === "-f" || args[idx] === "--file") && idx + 1 < args.length) {
            composeFile = resolve(process.cwd(), args[idx + 1]);
            idx += 2;
            continue;
        }
        // --file=PATH
        if (args[idx].startsWith("--file=")) {
            composeFile = resolve(process.cwd(), args[idx].slice("--file=".length));
            idx++;
            continue;
        }
        // -f=PATH
        if (args[idx].startsWith("-f=")) {
            composeFile = resolve(process.cwd(), args[idx].slice("-f=".length));
            idx++;
            continue;
        }
        break;
    }

    if (idx >= args.length) {
        process.stderr.write("mock-docker: no compose subcommand\n");
        process.exit(1);
    }

    if (!projectName) {
        projectName = basename(process.cwd());
    }

    return {
        projectName,
        envFiles,
        composeFile,
        subcmd: args[idx],
        restArgs: args.slice(idx + 1),
    };
}

function hasFlag(args: string[], flag: string): boolean {
    return args.includes(flag);
}

function findServiceArg(args: string[]): string {
    for (const a of args) {
        if (!a.startsWith("-")) return a;
    }
    return "";
}

// ---------------------------------------------------------------------------
// Compose file loading
// ---------------------------------------------------------------------------

function loadCompose(composeFilePath?: string): { parsed: ParsedCompose; services: string[] } {
    const composeFile = composeFilePath || findComposeFile(process.cwd());
    if (!composeFile) {
        process.stderr.write("no configuration file provided: not found\n");
        process.exit(1);
    }
    const content = readFileSync(composeFile, "utf-8");
    const parsed = parseCompose(content);
    const services = Object.keys(parsed.services);
    return { parsed, services };
}

function loadEnvFiles(envFiles: string[]): Record<string, string> {
    const env: Record<string, string> = {};
    for (const file of envFiles) {
        try {
            const content = readFileSync(resolve(file), "utf-8");
            for (const line of content.split("\n")) {
                const trimmed = line.trim();
                if (!trimmed || trimmed.startsWith("#")) continue;
                const eqIdx = trimmed.indexOf("=");
                if (eqIdx === -1) {
                    env[trimmed] = "";
                } else {
                    const key = trimmed.slice(0, eqIdx);
                    let val = trimmed.slice(eqIdx + 1);
                    // Strip surrounding quotes
                    if ((val.startsWith('"') && val.endsWith('"')) ||
                        (val.startsWith("'") && val.endsWith("'"))) {
                        val = val.slice(1, -1);
                    }
                    env[key] = val;
                }
            }
        } catch {
            // Ignore missing env files (matches real docker compose behavior)
        }
    }
    return env;
}

// ---------------------------------------------------------------------------
// Container config construction
// ---------------------------------------------------------------------------

function buildContainerConfig(
    project: string,
    serviceName: string,
    svc: ParsedService,
    envOverrides: Record<string, string>,
): {
    name: string;
    body: Record<string, unknown>;
} {
    const containerName = svc.containerName || `${project}-${serviceName}-1`;
    const image = svc.image || `${project}-${serviceName}`;

    // Merge environment: env file < compose env
    const env: Record<string, string> = { ...envOverrides, ...svc.environment };
    const envArr = Object.entries(env).map(([k, v]) => `${k}=${v}`);

    // Labels
    const labels: Record<string, string> = {
        "com.docker.compose.project": project,
        "com.docker.compose.service": serviceName,
        "com.docker.compose.container-number": "1",
        "com.docker.compose.version": "2.30.0",
        ...svc.labels,
    };

    // Exposed ports
    const exposedPorts: Record<string, Record<string, never>> = {};
    for (const p of svc.ports) {
        exposedPorts[`${p.target}/${p.protocol}`] = {};
    }
    for (const e of svc.expose) {
        const port = parseInt(e, 10);
        if (!isNaN(port)) {
            exposedPorts[`${port}/tcp`] = {};
        }
    }

    // Port bindings
    const portBindings: Record<string, Array<{ HostIp: string; HostPort: string }>> = {};
    for (const p of svc.ports) {
        if (p.published !== undefined) {
            const key = `${p.target}/${p.protocol}`;
            if (!portBindings[key]) portBindings[key] = [];
            portBindings[key].push({
                HostIp: p.hostIp || "0.0.0.0",
                HostPort: String(p.published),
            });
        }
    }

    // Binds
    const binds: string[] = [];
    for (const v of svc.volumes) {
        if (v.type === "bind") {
            const src = v.source.startsWith("/") ? v.source : resolve(process.cwd(), v.source);
            binds.push(`${src}:${v.target}${v.readOnly ? ":ro" : ""}`);
        } else if (v.type === "volume" && v.source) {
            binds.push(`${v.source}:${v.target}${v.readOnly ? ":ro" : ""}`);
        }
    }

    // Restart policy
    let restartPolicy = { Name: "no", MaximumRetryCount: 0 };
    if (svc.restart && svc.restart !== "no") {
        if (svc.restart.startsWith("on-failure")) {
            const parts = svc.restart.split(":");
            restartPolicy = {
                Name: "on-failure",
                MaximumRetryCount: parts[1] ? parseInt(parts[1], 10) : 0,
            };
        } else {
            restartPolicy = { Name: svc.restart, MaximumRetryCount: 0 };
        }
    }

    // Network mode
    const networkMode = svc.networkMode || `${project}_default`;

    // Command
    let cmd: string[] | undefined;
    if (svc.command) {
        cmd = typeof svc.command === "string"
            ? svc.command.split(/\s+/)
            : svc.command;
    }

    // Entrypoint
    let entrypoint: string[] | undefined;
    if (svc.entrypoint) {
        entrypoint = typeof svc.entrypoint === "string"
            ? [svc.entrypoint]
            : svc.entrypoint;
    }

    return {
        name: containerName,
        body: {
            Image: image,
            Cmd: cmd,
            Entrypoint: entrypoint,
            Env: envArr,
            Labels: labels,
            ExposedPorts: exposedPorts,
            HostConfig: {
                NetworkMode: networkMode,
                PortBindings: portBindings,
                Binds: binds.length > 0 ? binds : undefined,
                RestartPolicy: restartPolicy,
            },
        },
    };
}

// ---------------------------------------------------------------------------
// API helpers
// ---------------------------------------------------------------------------

async function createNetwork(
    socketPath: string,
    name: string,
    labels?: Record<string, string>,
): Promise<void> {
    const { statusCode } = await requestJSON(socketPath, "POST", "/networks/create", {
        Name: name,
        Driver: "bridge",
        Labels: labels || {},
    });
    // Ignore 409 = already exists
    if (statusCode !== 201 && statusCode !== 409) {
        process.stderr.write(`Warning: failed to create network ${name}\n`);
    }
}

async function createAndStartContainer(
    socketPath: string,
    name: string,
    config: Record<string, unknown>,
): Promise<string | null> {
    const createRes = await requestJSON<{ Id: string }>(
        socketPath,
        "POST",
        `/containers/create?name=${encodeURIComponent(name)}`,
        config,
    );
    if (createRes.statusCode !== 201) {
        process.stderr.write(`Warning: failed to create container ${name}\n`);
        return null;
    }
    const id = createRes.data.Id;

    const startRes = await requestJSON(socketPath, "POST", `/containers/${id}/start`);
    if (startRes.statusCode !== 204 && startRes.statusCode !== 304) {
        process.stderr.write(`Warning: failed to start container ${name}\n`);
    }

    return id;
}

async function listContainersByProject(
    socketPath: string,
    project: string,
): Promise<Array<{ Id: string; Names: string[]; Labels: Record<string, string>; Image: string; Command: string; Status: string }>> {
    const filters = JSON.stringify({
        label: [`com.docker.compose.project=${project}`],
    });
    const { data } = await requestJSON<Array<{
        Id: string;
        Names: string[];
        Labels: Record<string, string>;
        Image: string;
        Command: string;
        Status: string;
    }>>(socketPath, "GET", `/containers/json?all=1&filters=${encodeURIComponent(filters)}`);
    return Array.isArray(data) ? data : [];
}

async function stopContainer(socketPath: string, id: string): Promise<void> {
    await requestJSON(socketPath, "POST", `/containers/${id}/stop`);
}

async function removeContainer(socketPath: string, id: string): Promise<void> {
    await requestJSON(socketPath, "DELETE", `/containers/${id}?force=true`);
}

async function removeNetwork(socketPath: string, id: string): Promise<void> {
    await requestJSON(socketPath, "DELETE", `/networks/${id}`);
}

// ---------------------------------------------------------------------------
// Compose subcommands
// ---------------------------------------------------------------------------

async function composeUp(
    socketPath: string,
    project: string,
    parsed: ParsedCompose,
    envOverrides: Record<string, string>,
    restArgs: string[],
): Promise<void> {
    const svcArg = findServiceArg(restArgs);
    const forceRecreate = hasFlag(restArgs, "--force-recreate");
    const serviceNames = svcArg
        ? [svcArg]
        : Object.keys(parsed.services);
    const isWholeStack = !svcArg;

    // Show progress
    const tasks = composeUpTasks(project, serviceNames, isWholeStack, forceRecreate);
    await renderProgress("Running", tasks);

    // Create default network
    if (isWholeStack) {
        await createNetwork(socketPath, `${project}_default`, {
            "com.docker.compose.project": project,
            "com.docker.compose.network": "default",
        });
    }

    // Create project-defined networks
    for (const [netName, netDef] of Object.entries(parsed.networks)) {
        if (!netDef.external) {
            const fullName = netDef.name || `${project}_${netName}`;
            await createNetwork(socketPath, fullName);
        }
    }

    // If force-recreate, stop + remove existing containers first
    if (forceRecreate) {
        const existing = await listContainersByProject(socketPath, project);
        for (const ctr of existing) {
            const svcLabel = ctr.Labels["com.docker.compose.service"] || "";
            if (serviceNames.includes(svcLabel)) {
                await stopContainer(socketPath, ctr.Id);
                await removeContainer(socketPath, ctr.Id);
            }
        }
    }

    // Create and start containers
    for (const svcName of serviceNames) {
        const svc = parsed.services[svcName];
        if (!svc) continue;
        const { name, body } = buildContainerConfig(project, svcName, svc, envOverrides);
        await createAndStartContainer(socketPath, name, body);
    }
}

async function composeStop(
    socketPath: string,
    project: string,
    restArgs: string[],
    composeFilePath?: string,
): Promise<void> {
    const { services: allServices } = loadCompose(composeFilePath);
    const svcArg = findServiceArg(restArgs);
    const serviceNames = svcArg ? [svcArg] : allServices;

    const tasks = composeStopTasks(project, serviceNames);
    await renderProgress("Stopping", tasks);

    const containers = await listContainersByProject(socketPath, project);
    for (const ctr of containers) {
        const svcLabel = ctr.Labels["com.docker.compose.service"] || "";
        if (serviceNames.includes(svcLabel)) {
            await stopContainer(socketPath, ctr.Id);
        }
    }
}

async function composeDown(
    socketPath: string,
    project: string,
    parsed: ParsedCompose,
    restArgs: string[],
): Promise<void> {
    const removeVolumes = hasFlag(restArgs, "-v") || hasFlag(restArgs, "--volumes");
    const serviceNames = Object.keys(parsed.services);

    const tasks = composeDownTasks(project, serviceNames, removeVolumes);
    await renderProgress("Running", tasks);

    // Stop and remove containers
    const containers = await listContainersByProject(socketPath, project);
    for (const ctr of containers) {
        await stopContainer(socketPath, ctr.Id);
        await removeContainer(socketPath, ctr.Id);
    }

    // Remove volumes if requested
    if (removeVolumes) {
        for (const volName of Object.keys(parsed.volumes)) {
            const fullName = `${project}_${volName}`;
            await requestJSON(socketPath, "DELETE", `/volumes/${encodeURIComponent(fullName)}`);
        }
    }

    // Remove networks (ignore errors — may still be in use)
    const networks = await requestJSON<Array<{ Id: string; Name: string; Labels?: Record<string, string> }>>(
        socketPath, "GET", "/networks",
    );
    if (Array.isArray(networks.data)) {
        for (const net of networks.data) {
            const netProject = net.Labels?.["com.docker.compose.project"];
            if (netProject === project) {
                await removeNetwork(socketPath, net.Id);
            }
        }
    }
    // Also try removing the default network by name
    await requestJSON(socketPath, "DELETE", `/networks/${encodeURIComponent(`${project}_default`)}`);
}

async function composeRestart(
    socketPath: string,
    project: string,
    restArgs: string[],
    composeFilePath?: string,
): Promise<void> {
    const { services: allServices } = loadCompose(composeFilePath);
    const svcArg = findServiceArg(restArgs);
    const serviceNames = svcArg ? [svcArg] : allServices;

    const tasks = composeRestartTasks(project, serviceNames);
    await renderProgress("Restarting", tasks);

    const containers = await listContainersByProject(socketPath, project);
    for (const ctr of containers) {
        const svcLabel = ctr.Labels["com.docker.compose.service"] || "";
        if (serviceNames.includes(svcLabel)) {
            await requestJSON(socketPath, "POST", `/containers/${ctr.Id}/restart`);
        }
    }
}

async function composePull(restArgs: string[], composeFilePath?: string): Promise<void> {
    const { services: allServices } = loadCompose(composeFilePath);
    const svcArg = findServiceArg(restArgs);
    const serviceNames = svcArg ? [svcArg] : allServices;

    const tasks = composePullTasks(serviceNames);
    await renderProgress("Pulling", tasks);
    // No actual pull — this is a mock
}

async function composePause(
    socketPath: string,
    project: string,
    composeFilePath?: string,
): Promise<void> {
    const { services: allServices } = loadCompose(composeFilePath);
    const tasks = composePauseTasks(project, allServices);
    await renderProgress("Pausing", tasks);

    const containers = await listContainersByProject(socketPath, project);
    for (const ctr of containers) {
        await requestJSON(socketPath, "POST", `/containers/${ctr.Id}/pause`);
    }
}

async function composeUnpause(
    socketPath: string,
    project: string,
    composeFilePath?: string,
): Promise<void> {
    const { services: allServices } = loadCompose(composeFilePath);
    const tasks = composeUnpauseTasks(project, allServices);
    await renderProgress("Unpausing", tasks);

    const containers = await listContainersByProject(socketPath, project);
    for (const ctr of containers) {
        await requestJSON(socketPath, "POST", `/containers/${ctr.Id}/unpause`);
    }
}

async function composeConfig(composeFilePath?: string): Promise<void> {
    const composeFile = composeFilePath || findComposeFile(process.cwd());
    if (!composeFile) {
        process.stderr.write("no configuration file provided: not found\n");
        process.exit(1);
    }
    const content = readFileSync(composeFile, "utf-8");
    // Validate it has a services section
    if (!content.includes("services:")) {
        process.stderr.write("services must be a mapping\n");
        process.exit(1);
    }
    // Config validated — real docker compose config just outputs the resolved YAML
    // For mock purposes, success is enough
}

async function composeExec(restArgs: string[]): Promise<void> {
    // docker compose exec [flags] <service> <command> [args...]
    const rest: string[] = [];
    for (let i = 0; i < restArgs.length; i++) {
        const a = restArgs[i];
        if (a.startsWith("-")) {
            // Skip flags that take a value
            if (["-u", "--user", "-e", "--env", "-w", "--workdir"].includes(a)) {
                i++; // skip the value
            }
            continue;
        }
        rest.push(a);
    }
    if (rest.length < 2) {
        process.stderr.write("[mock-docker] compose exec: service and command required\n");
        process.exit(1);
    }
    const shell = rest[1];
    await execShell(shell);
}

async function composeLogs(
    socketPath: string,
    project: string,
    restArgs: string[],
    composeFilePath?: string,
): Promise<void> {
    const { services: allServices } = loadCompose(composeFilePath);

    const logColors = [
        "\x1b[36m", "\x1b[33m", "\x1b[32m", "\x1b[35m", "\x1b[34m",
        "\x1b[96m", "\x1b[93m", "\x1b[92m", "\x1b[95m", "\x1b[94m",
    ];

    let maxLen = 0;
    for (const svc of allServices) {
        if (svc.length > maxLen) maxLen = svc.length;
    }

    for (let i = 0; i < allServices.length; i++) {
        const svc = allServices[i];
        const color = logColors[i % logColors.length];
        const padded = svc.padEnd(maxLen);
        const prefix = `${color}${padded} | \x1b[0m`;
        const lines = await fetchServiceStartupLogs(socketPath, project, svc);
        for (const line of lines) {
            process.stdout.write(`${prefix}${line}\n`);
        }
    }

    // If -f/--follow, block until killed
    if (hasFlag(restArgs, "-f") || hasFlag(restArgs, "--follow")) {
        await new Promise(() => {}); // block forever — parent will kill us
    }
}

async function composePs(
    socketPath: string,
    project: string,
): Promise<void> {
    const containers = await listContainersByProject(socketPath, project);

    // Compute column widths
    const rows = containers.map((ctr) => ({
        name: ctr.Names?.[0]?.replace(/^\//, "") || "",
        image: ctr.Image || "",
        command: ctr.Command ? `"${ctr.Command}"` : "",
        service: ctr.Labels["com.docker.compose.service"] || "",
        status: ctr.Status || "",
    }));

    const cols = { name: 4, image: 5, command: 7, service: 7, status: 6 };
    for (const r of rows) {
        if (r.name.length > cols.name) cols.name = r.name.length;
        if (r.image.length > cols.image) cols.image = r.image.length;
        if (r.command.length > cols.command) cols.command = r.command.length;
        if (r.service.length > cols.service) cols.service = r.service.length;
        if (r.status.length > cols.status) cols.status = r.status.length;
    }

    const pad = (s: string, w: number) => s.padEnd(w + 3);
    console.log(
        pad("NAME", cols.name) + pad("IMAGE", cols.image) + pad("COMMAND", cols.command)
        + pad("SERVICE", cols.service) + "STATUS",
    );
    for (const r of rows) {
        console.log(
            pad(r.name, cols.name) + pad(r.image, cols.image) + pad(r.command, cols.command)
            + pad(r.service, cols.service) + r.status,
        );
    }
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

interface MockLogsResponse {
    startup: string[];
    heartbeat: string[];
    interval: string;
    shutdown: string[];
}

async function fetchServiceStartupLogs(
    socketPath: string,
    project: string,
    service: string,
): Promise<string[]> {
    try {
        const { statusCode, data } = await requestJSON<MockLogsResponse>(
            socketPath,
            "GET",
            `/_mock/logs/${project}/${service}`,
        );
        if (statusCode === 200 && data && Array.isArray(data.startup) && data.startup.length > 0) {
            return data.startup;
        }
    } catch {
        // Fall through to default
    }
    return [`[INFO] ${service} started`];
}

async function execShell(shell: string): Promise<void> {
    // In Node.js we can't do syscall.Exec like Go. Use child_process.spawn
    // with stdio: "inherit" and forward exit code.
    const { spawn } = await import("node:child_process");
    const child = spawn(shell, [], {
        stdio: "inherit",
        env: process.env,
    });
    child.on("error", () => {
        // Try fallback to sh
        const fallback = spawn("sh", [], {
            stdio: "inherit",
            env: process.env,
        });
        fallback.on("error", () => {
            process.stderr.write("[mock-docker] exec: no shell available\n");
            process.exit(1);
        });
        fallback.on("exit", (code) => {
            process.exit(code || 0);
        });
    });
    child.on("exit", (code) => {
        process.exit(code || 0);
    });
}

// ---------------------------------------------------------------------------
// Main handler
// ---------------------------------------------------------------------------

export async function handleCompose(
    socketPath: string,
    args: string[],
): Promise<void> {
    const { projectName, envFiles, composeFile, subcmd, restArgs } = parseComposeFlags(args);
    const envOverrides = loadEnvFiles(envFiles);
    const cf = composeFile || undefined;

    switch (subcmd) {
        case "up": {
            const { parsed } = loadCompose(cf);
            await composeUp(socketPath, projectName, parsed, envOverrides, restArgs);
            break;
        }
        case "stop":
            await composeStop(socketPath, projectName, restArgs, cf);
            break;
        case "down": {
            const { parsed } = loadCompose(cf);
            await composeDown(socketPath, projectName, parsed, restArgs);
            break;
        }
        case "restart":
            await composeRestart(socketPath, projectName, restArgs, cf);
            break;
        case "pull":
            await composePull(restArgs, cf);
            break;
        case "pause":
            await composePause(socketPath, projectName, cf);
            break;
        case "unpause":
            await composeUnpause(socketPath, projectName, cf);
            break;
        case "config":
            await composeConfig(cf);
            break;
        case "exec":
            await composeExec(restArgs);
            break;
        case "logs":
            await composeLogs(socketPath, projectName, restArgs, cf);
            break;
        case "ps":
            await composePs(socketPath, projectName);
            break;
        default:
            process.stderr.write(`[mock-docker] unsupported compose command: ${subcmd}\n`);
            break;
    }
}
