import { requestJSON, requestRaw, requestStream } from "./socket-client.js";

// ---------------------------------------------------------------------------
// docker subcommand handlers
// ---------------------------------------------------------------------------

async function containerAction(
    socketPath: string,
    action: string,
    args: string[],
): Promise<void> {
    if (args.length === 0) {
        process.stderr.write(`[mock-docker] ${action}: container name required\n`);
        process.exit(1);
    }
    const containerName = args[0];

    const { statusCode, data } = await requestJSON<{ message?: string }>(
        socketPath,
        "POST",
        `/containers/${encodeURIComponent(containerName)}/${action}`,
    );
    if (statusCode === 304) {
        // Container already in the desired state
        const msg = action === "stop" || action === "restart"
            ? `container ${containerName} is not running`
            : `container ${containerName} is already started`;
        process.stderr.write(`Error response from daemon: ${msg}\n`);
        process.exitCode = 1;
        return;
    }
    if (statusCode >= 400) {
        const msg = (data && typeof data === "object" && data.message) || `${action} failed`;
        process.stderr.write(`Error response from daemon: ${msg}\n`);
        process.exitCode = 1;
        return;
    }
    console.log(containerName);
}

async function dockerExec(
    socketPath: string,
    args: string[],
): Promise<void> {
    // docker exec [-it] <container> <command> [args...]
    const rest: string[] = [];
    for (let i = 0; i < args.length; i++) {
        const a = args[i];
        if (a === "-i" || a === "-t" || a === "-it" || a === "-ti") continue;
        if (a.startsWith("-") && !a.startsWith("--")) continue;
        rest.push(a);
    }
    if (rest.length < 2) {
        process.stderr.write("[mock-docker] exec: container and command required\n");
        process.exit(1);
    }
    const container = rest[0];
    const command = rest[1];
    const cmdArgs = rest.slice(2);

    // Step 1: Create exec instance
    const createRes = await requestJSON<{ Id: string }>(
        socketPath,
        "POST",
        `/containers/${encodeURIComponent(container)}/exec`,
        { Cmd: [command, ...cmdArgs], AttachStdout: true, AttachStderr: true, Tty: false },
    );
    if (createRes.statusCode >= 400) {
        process.stderr.write(`Error: No such container: ${container}\n`);
        process.exitCode = 1;
        return;
    }
    const execId = createRes.data?.Id;
    if (!execId) {
        process.stderr.write("[mock-docker] exec: failed to create exec instance\n");
        process.exitCode = 1;
        return;
    }

    // Step 2: Start exec and read output
    const { statusCode, body } = await requestRaw(
        socketPath,
        "POST",
        `/exec/${execId}/start`,
        { Detach: false, Tty: false },
    );
    if (statusCode >= 400) {
        process.stderr.write("[mock-docker] exec: start failed\n");
        process.exitCode = 1;
        return;
    }

    // Check for Docker multiplexed stream framing
    const looksMultiplexed = body.length >= 8
        && body[0] <= 2
        && body[1] === 0 && body[2] === 0 && body[3] === 0;

    if (looksMultiplexed) {
        let offset = 0;
        while (offset + 8 <= body.length) {
            const streamType = body[offset];
            const payloadLen = body.readUInt32BE(offset + 4);
            offset += 8;
            if (offset + payloadLen > body.length) break;
            const payload = body.subarray(offset, offset + payloadLen);
            if (streamType === 2) {
                process.stderr.write(payload);
            } else {
                process.stdout.write(payload);
            }
            offset += payloadLen;
        }
    } else if (body.length > 0) {
        process.stdout.write(body);
    }
}

async function dockerPs(
    socketPath: string,
    args: string[],
): Promise<void> {
    const all = args.includes("--all") || args.includes("-a");
    const formatIdx = args.indexOf("--format");
    const format = formatIdx !== -1 && args[formatIdx + 1] ? args[formatIdx + 1] : "";

    // Parse --filter flags
    const filterParts: string[] = [];
    for (let i = 0; i < args.length; i++) {
        if ((args[i] === "--filter" || args[i] === "-f") && args[i + 1]) {
            filterParts.push(args[i + 1]);
            i++;
        }
    }

    let query = all ? "all=1" : "";
    if (filterParts.length > 0) {
        const filters: Record<string, string[]> = {};
        for (const fp of filterParts) {
            const eqIdx = fp.indexOf("=");
            if (eqIdx !== -1) {
                const key = fp.slice(0, eqIdx);
                const val = fp.slice(eqIdx + 1);
                if (!filters[key]) filters[key] = [];
                filters[key].push(val);
            }
        }
        const sep = query ? "&" : "";
        query += `${sep}filters=${encodeURIComponent(JSON.stringify(filters))}`;
    }

    const { data } = await requestJSON<Array<{
        Id: string;
        Names: string[];
        Image: string;
        State: string;
        Status: string;
    }>>(socketPath, "GET", `/containers/json?${query}`);

    if (!Array.isArray(data)) return;

    if (format === "{{.ID}}") {
        for (const c of data) console.log(c.Id.slice(0, 12));
        return;
    }

    // Default table output
    console.log("CONTAINER ID   IMAGE          STATUS         NAMES");
    for (const c of data) {
        const id = c.Id.slice(0, 12);
        const image = (c.Image || "").slice(0, 14).padEnd(14);
        const status = (c.Status || c.State || "").padEnd(14);
        const names = (c.Names || []).map((n: string) => n.replace(/^\//, "")).join(",");
        console.log(`${id}   ${image} ${status} ${names}`);
    }
}

async function dockerRm(
    socketPath: string,
    args: string[],
): Promise<void> {
    const force = args.includes("-f") || args.includes("--force");
    const containers = args.filter((a) => !a.startsWith("-"));

    for (const name of containers) {
        const query = force ? "?force=true" : "";
        const { statusCode, data } = await requestJSON<{ message?: string }>(
            socketPath,
            "DELETE",
            `/containers/${encodeURIComponent(name)}${query}`,
        );
        if (statusCode >= 400) {
            const msg = (data && typeof data === "object" && data.message) || "remove failed";
            process.stderr.write(`Error response from daemon: ${msg}\n`);
            process.exitCode = 1;
        } else {
            console.log(name);
        }
    }
}

async function dockerKill(
    socketPath: string,
    args: string[],
): Promise<void> {
    let signal = "SIGKILL";
    const containers: string[] = [];
    for (let i = 0; i < args.length; i++) {
        if (args[i] === "--signal" || args[i] === "-s") {
            signal = args[++i] || "SIGKILL";
        } else if (!args[i].startsWith("-")) {
            containers.push(args[i]);
        }
    }

    for (const name of containers) {
        const { statusCode } = await requestJSON(
            socketPath,
            "POST",
            `/containers/${encodeURIComponent(name)}/kill?signal=${signal}`,
        );
        if (statusCode >= 400) {
            process.stderr.write(`Error: No such container: ${name}\n`);
            process.exitCode = 1;
        } else {
            console.log(name);
        }
    }
}

async function dockerPause(
    socketPath: string,
    args: string[],
): Promise<void> {
    for (const name of args.filter((a) => !a.startsWith("-"))) {
        const { statusCode } = await requestJSON(
            socketPath,
            "POST",
            `/containers/${encodeURIComponent(name)}/pause`,
        );
        if (statusCode >= 400) {
            process.stderr.write(`Error: No such container: ${name}\n`);
            process.exitCode = 1;
        } else {
            console.log(name);
        }
    }
}

async function dockerUnpause(
    socketPath: string,
    args: string[],
): Promise<void> {
    for (const name of args.filter((a) => !a.startsWith("-"))) {
        const { statusCode } = await requestJSON(
            socketPath,
            "POST",
            `/containers/${encodeURIComponent(name)}/unpause`,
        );
        if (statusCode >= 400) {
            process.stderr.write(`Error: No such container: ${name}\n`);
            process.exitCode = 1;
        } else {
            console.log(name);
        }
    }
}

async function dockerInspect(
    socketPath: string,
    args: string[],
): Promise<void> {
    const resources = args.filter((a) => !a.startsWith("-"));
    if (resources.length === 0) {
        process.stderr.write("Error: requires at least 1 argument\n");
        process.exit(1);
    }

    const results: unknown[] = [];
    for (const name of resources) {
        // Try containers first
        let { statusCode, data } = await requestJSON(
            socketPath,
            "GET",
            `/containers/${encodeURIComponent(name)}/json`,
        );
        if (statusCode === 200) {
            results.push(data);
            continue;
        }

        // Try networks
        ({ statusCode, data } = await requestJSON(
            socketPath,
            "GET",
            `/networks/${encodeURIComponent(name)}`,
        ));
        if (statusCode === 200) {
            results.push(data);
            continue;
        }

        // Try volumes
        ({ statusCode, data } = await requestJSON(
            socketPath,
            "GET",
            `/volumes/${encodeURIComponent(name)}`,
        ));
        if (statusCode === 200) {
            results.push(data);
            continue;
        }

        process.stderr.write(`Error: No such object: ${name}\n`);
        process.exitCode = 1;
    }

    if (results.length > 0) {
        console.log(JSON.stringify(results, null, 4));
    }
}

async function dockerLogs(
    socketPath: string,
    args: string[],
): Promise<void> {
    // Parse flags with value consumption for --tail, --since, --until
    let follow = false;
    let timestamps = false;
    let tail: string | undefined;
    let since: string | undefined;
    let until: string | undefined;
    const positional: string[] = [];

    const valueFlagNames = new Set(["--tail", "--since", "--until"]);

    for (let i = 0; i < args.length; i++) {
        const a = args[i];
        if (a === "-f" || a === "--follow") {
            follow = true;
        } else if (a === "-t" || a === "--timestamps") {
            timestamps = true;
        } else if (a.startsWith("--tail=")) {
            tail = a.slice("--tail=".length);
        } else if (a.startsWith("--since=")) {
            since = a.slice("--since=".length);
        } else if (a.startsWith("--until=")) {
            until = a.slice("--until=".length);
        } else if (valueFlagNames.has(a)) {
            // Consume next arg as the value
            i++;
            const val = args[i];
            if (a === "--tail") tail = val;
            else if (a === "--since") since = val;
            else if (a === "--until") until = val;
        } else if (a === "-n" && args[i + 1]) {
            // Short form for --tail
            tail = args[++i];
        } else if (!a.startsWith("-")) {
            positional.push(a);
        }
    }

    if (positional.length === 0) {
        process.stderr.write("Error: requires at least 1 argument\n");
        process.exit(1);
    }

    const name = positional[0];

    // Build query string
    const params: string[] = ["stdout=1", "stderr=1"];
    if (follow) params.push("follow=1");
    if (timestamps) params.push("timestamps=1");
    if (tail !== undefined) params.push(`tail=${encodeURIComponent(tail)}`);
    if (since !== undefined) params.push(`since=${encodeURIComponent(since)}`);
    if (until !== undefined) params.push(`until=${encodeURIComponent(until)}`);
    const query = params.join("&");

    const { statusCode, headers, body } = await requestRaw(
        socketPath,
        "GET",
        `/containers/${encodeURIComponent(name)}/logs?${query}`,
    );
    if (statusCode >= 400) {
        process.stderr.write(`Error: No such container: ${name}\n`);
        process.exitCode = 1;
        return;
    }

    // Detect multiplexed stream: check content-type header OR auto-detect
    // from the data (byte 0 is stream type 0-2, bytes 1-3 are zeros).
    // Auto-detection is needed because some Docker API implementations
    // may return raw-stream content-type with multiplexed framing.
    const contentType = typeof headers["content-type"] === "string"
        ? headers["content-type"]
        : "";
    const looksMultiplexed = body.length >= 8
        && body[0] <= 2
        && body[1] === 0 && body[2] === 0 && body[3] === 0;
    const isMultiplexed = contentType.includes("application/vnd.docker.multiplexed-stream")
        || looksMultiplexed;

    if (isMultiplexed) {
        // Parse 8-byte framed multiplexed stream
        let offset = 0;
        while (offset + 8 <= body.length) {
            const streamType = body[offset]; // 1=stdout, 2=stderr
            const payloadLen = body.readUInt32BE(offset + 4);
            offset += 8;
            if (offset + payloadLen > body.length) break;
            const payload = body.subarray(offset, offset + payloadLen);
            if (streamType === 2) {
                process.stderr.write(payload);
            } else {
                process.stdout.write(payload);
            }
            offset += payloadLen;
        }
    } else {
        // Raw stream or plain text — write as-is
        process.stdout.write(body);
    }
}

async function dockerTop(
    socketPath: string,
    args: string[],
): Promise<void> {
    const containers = args.filter((a) => !a.startsWith("-"));
    if (containers.length === 0) {
        process.stderr.write("Error: requires at least 1 argument\n");
        process.exit(1);
    }
    const { statusCode, data } = await requestJSON<{
        Titles: string[];
        Processes: string[][];
    }>(socketPath, "GET", `/containers/${encodeURIComponent(containers[0])}/top`);
    if (statusCode >= 400 || !data) {
        process.stderr.write(`Error: No such container: ${containers[0]}\n`);
        process.exitCode = 1;
        return;
    }
    if (data.Titles) console.log(data.Titles.join("\t"));
    if (data.Processes) {
        for (const row of data.Processes) {
            console.log(row.join("\t"));
        }
    }
}

async function dockerStats(
    socketPath: string,
    args: string[],
): Promise<void> {
    const containers = args.filter((a) => !a.startsWith("-"));
    const oneShot = args.includes("--no-stream");

    if (containers.length === 0) {
        // List all running containers
        const { data } = await requestJSON<Array<{ Id: string; Names: string[] }>>(
            socketPath,
            "GET",
            "/containers/json",
        );
        if (Array.isArray(data)) {
            for (const c of data) {
                containers.push(c.Id.slice(0, 12));
            }
        }
    }

    console.log("CONTAINER ID   NAME           CPU %     MEM USAGE / LIMIT");
    for (const name of containers) {
        const { data } = await requestJSON<{
            id: string;
            name: string;
            cpu_stats: {
                cpu_usage: { total_usage: number };
                system_cpu_usage: number;
                online_cpus: number;
            };
            precpu_stats: {
                cpu_usage: { total_usage: number };
                system_cpu_usage: number;
            };
            memory_stats: { usage: number; limit: number };
        }>(socketPath, "GET", `/containers/${encodeURIComponent(name)}/stats?stream=false`);
        if (data && typeof data === "object") {
            const id = ((data.id as string) || name).slice(0, 12);
            const cname = ((data.name as string) || "").replace(/^\//, "").padEnd(14);

            // Compute CPU %
            const cpuDelta = (data.cpu_stats?.cpu_usage?.total_usage ?? 0)
                - (data.precpu_stats?.cpu_usage?.total_usage ?? 0);
            const systemDelta = (data.cpu_stats?.system_cpu_usage ?? 0)
                - (data.precpu_stats?.system_cpu_usage ?? 0);
            const numCpus = data.cpu_stats?.online_cpus || 1;
            const cpuPercent = systemDelta > 0
                ? (cpuDelta / systemDelta) * numCpus * 100
                : 0;

            // Format memory
            const memUsage = data.memory_stats?.usage ?? 0;
            const memLimit = data.memory_stats?.limit ?? 0;

            console.log(`${id}   ${cname} ${cpuPercent.toFixed(2)}%     ${formatMemory(memUsage)} / ${formatMemory(memLimit)}`);
        }
    }
}

async function dockerNetworkLs(
    socketPath: string,
): Promise<void> {
    const { data } = await requestJSON<Array<{
        Id: string;
        Name: string;
        Driver: string;
        Scope: string;
    }>>(socketPath, "GET", "/networks");

    console.log("NETWORK ID     NAME                   DRIVER    SCOPE");
    if (Array.isArray(data)) {
        for (const n of data) {
            const id = n.Id.slice(0, 12);
            const name = n.Name.padEnd(22);
            const driver = (n.Driver || "bridge").padEnd(9);
            console.log(`${id}   ${name} ${driver} ${n.Scope || "local"}`);
        }
    }
}

async function dockerNetworkCreate(
    socketPath: string,
    args: string[],
): Promise<void> {
    const names = args.filter((a) => !a.startsWith("-"));
    if (names.length === 0) {
        process.stderr.write("Error: network name is required\n");
        process.exit(1);
    }
    const { statusCode, data } = await requestJSON<{ Id: string }>(
        socketPath,
        "POST",
        "/networks/create",
        { Name: names[0] },
    );
    if (statusCode >= 400) {
        process.stderr.write(`Error creating network\n`);
        process.exitCode = 1;
    } else {
        console.log(data?.Id?.slice(0, 12) || names[0]);
    }
}

async function dockerNetworkRm(
    socketPath: string,
    args: string[],
): Promise<void> {
    for (const name of args.filter((a) => !a.startsWith("-"))) {
        const { statusCode } = await requestJSON(
            socketPath,
            "DELETE",
            `/networks/${encodeURIComponent(name)}`,
        );
        if (statusCode >= 400) {
            process.stderr.write(`Error: No such network: ${name}\n`);
            process.exitCode = 1;
        } else {
            console.log(name);
        }
    }
}

async function dockerNetworkConnect(
    socketPath: string,
    args: string[],
): Promise<void> {
    const positional = args.filter((a) => !a.startsWith("-"));
    if (positional.length < 2) {
        process.stderr.write("Error: requires 2 arguments (network, container)\n");
        process.exit(1);
    }
    const { statusCode } = await requestJSON(
        socketPath,
        "POST",
        `/networks/${encodeURIComponent(positional[0])}/connect`,
        { Container: positional[1] },
    );
    if (statusCode >= 400) {
        process.stderr.write(`Error connecting to network\n`);
        process.exitCode = 1;
    }
}

async function dockerNetworkDisconnect(
    socketPath: string,
    args: string[],
): Promise<void> {
    const positional = args.filter((a) => !a.startsWith("-"));
    if (positional.length < 2) {
        process.stderr.write("Error: requires 2 arguments (network, container)\n");
        process.exit(1);
    }
    const { statusCode } = await requestJSON(
        socketPath,
        "POST",
        `/networks/${encodeURIComponent(positional[0])}/disconnect`,
        { Container: positional[1] },
    );
    if (statusCode >= 400) {
        process.stderr.write(`Error disconnecting from network\n`);
        process.exitCode = 1;
    }
}

async function dockerNetwork(
    socketPath: string,
    args: string[],
): Promise<void> {
    if (args.length === 0) {
        process.stderr.write("Usage: docker network COMMAND\n");
        return;
    }
    switch (args[0]) {
        case "ls":
            await dockerNetworkLs(socketPath);
            break;
        case "create":
            await dockerNetworkCreate(socketPath, args.slice(1));
            break;
        case "rm":
        case "remove":
            await dockerNetworkRm(socketPath, args.slice(1));
            break;
        case "connect":
            await dockerNetworkConnect(socketPath, args.slice(1));
            break;
        case "disconnect":
            await dockerNetworkDisconnect(socketPath, args.slice(1));
            break;
        default:
            process.stderr.write(`[mock-docker] unsupported network command: ${args[0]}\n`);
            break;
    }
}

async function dockerVolumeLs(
    socketPath: string,
): Promise<void> {
    const { data } = await requestJSON<{
        Volumes: Array<{ Name: string; Driver: string; Mountpoint: string }>;
    }>(socketPath, "GET", "/volumes");

    console.log("DRIVER    VOLUME NAME");
    if (data && Array.isArray(data.Volumes)) {
        for (const v of data.Volumes) {
            console.log(`${(v.Driver || "local").padEnd(9)} ${v.Name}`);
        }
    }
}

async function dockerVolumeCreate(
    socketPath: string,
    args: string[],
): Promise<void> {
    const names = args.filter((a) => !a.startsWith("-"));
    const name = names[0] || "";
    const { statusCode, data } = await requestJSON<{ Name: string }>(
        socketPath,
        "POST",
        "/volumes/create",
        { Name: name },
    );
    if (statusCode >= 400) {
        process.stderr.write("Error creating volume\n");
        process.exitCode = 1;
    } else {
        console.log(data?.Name || name);
    }
}

async function dockerVolumeRm(
    socketPath: string,
    args: string[],
): Promise<void> {
    for (const name of args.filter((a) => !a.startsWith("-"))) {
        const { statusCode } = await requestJSON(
            socketPath,
            "DELETE",
            `/volumes/${encodeURIComponent(name)}`,
        );
        if (statusCode >= 400) {
            process.stderr.write(`Error: No such volume: ${name}\n`);
            process.exitCode = 1;
        } else {
            console.log(name);
        }
    }
}

async function dockerVolume(
    socketPath: string,
    args: string[],
): Promise<void> {
    if (args.length === 0) {
        process.stderr.write("Usage: docker volume COMMAND\n");
        return;
    }
    switch (args[0]) {
        case "ls":
            await dockerVolumeLs(socketPath);
            break;
        case "create":
            await dockerVolumeCreate(socketPath, args.slice(1));
            break;
        case "rm":
        case "remove":
            await dockerVolumeRm(socketPath, args.slice(1));
            break;
        default:
            process.stderr.write(`[mock-docker] unsupported volume command: ${args[0]}\n`);
            break;
    }
}

async function dockerImages(
    socketPath: string,
): Promise<void> {
    const { data } = await requestJSON<Array<{
        Id: string;
        RepoTags: string[];
        Size: number;
    }>>(socketPath, "GET", "/images/json");

    console.log("REPOSITORY          TAG       IMAGE ID       SIZE");
    if (Array.isArray(data)) {
        for (const img of data) {
            const tag = (img.RepoTags?.[0] || "<none>:<none>").split(":");
            const repo = (tag[0] || "<none>").padEnd(19);
            const tagStr = (tag[1] || "<none>").padEnd(9);
            const id = (img.Id || "").replace("sha256:", "").slice(0, 12);
            const size = formatSize(img.Size || 0);
            console.log(`${repo} ${tagStr} ${id}   ${size}`);
        }
    }
}

async function dockerImagePrune(
    socketPath: string,
    args: string[],
): Promise<void> {
    console.log("Total reclaimed space: 0B");
}

async function dockerImage(
    socketPath: string,
    args: string[],
): Promise<void> {
    if (args.length === 0) {
        process.stderr.write("Usage: docker image COMMAND\n");
        return;
    }
    switch (args[0]) {
        case "prune":
            await dockerImagePrune(socketPath, args.slice(1));
            break;
        case "ls":
            await dockerImages(socketPath);
            break;
        default:
            process.stderr.write(`[mock-docker] unsupported image command: ${args[0]}\n`);
            break;
    }
}

async function dockerVersion(
    socketPath: string,
): Promise<void> {
    const { statusCode, data } = await requestJSON<{
        Platform: { Name: string };
        Components: Array<{ Name: string; Version: string; Details?: Record<string, string> }>;
        Version: string;
        ApiVersion: string;
        MinAPIVersion: string;
        GitCommit: string;
        GoVersion: string;
        Os: string;
        Arch: string;
    }>(socketPath, "GET", "/version");

    if (statusCode >= 400 || !data || typeof data !== "object") {
        process.stderr.write("Error: failed to get docker version\n");
        process.exitCode = 1;
        return;
    }

    console.log("Client:");
    console.log(` Version:    mock-docker`);
    console.log(` API version: ${data.ApiVersion ?? ""}`);
    console.log("");
    console.log("Server:");
    console.log(` Version:      ${data.Version ?? ""}`);
    console.log(` API version:  ${data.ApiVersion ?? ""}`);
    console.log(` Min API version: ${data.MinAPIVersion ?? ""}`);
    console.log(` Git commit:   ${data.GitCommit ?? ""}`);
    console.log(` Go version:   ${data.GoVersion ?? ""}`);
    console.log(` Os/Arch:      ${data.Os ?? ""}/${data.Arch ?? ""}`);
}

async function dockerInfo(
    socketPath: string,
): Promise<void> {
    const { statusCode, data } = await requestJSON<{
        Containers: number;
        ContainersRunning: number;
        ContainersPaused: number;
        ContainersStopped: number;
        Images: number;
        ServerVersion: string;
        Driver: string;
        DockerRootDir: string;
        Name: string;
        OperatingSystem: string;
        OSType: string;
        Architecture: string;
    }>(socketPath, "GET", "/info");

    if (statusCode >= 400 || !data || typeof data !== "object") {
        process.stderr.write("Error: failed to get docker info\n");
        process.exitCode = 1;
        return;
    }

    console.log(`Containers: ${data.Containers ?? 0}`);
    console.log(` Running: ${data.ContainersRunning ?? 0}`);
    console.log(` Paused: ${data.ContainersPaused ?? 0}`);
    console.log(` Stopped: ${data.ContainersStopped ?? 0}`);
    console.log(`Images: ${data.Images ?? 0}`);
    console.log(`Server Version: ${data.ServerVersion ?? ""}`);
    console.log(`Storage Driver: ${data.Driver ?? ""}`);
    console.log(`Docker Root Dir: ${data.DockerRootDir ?? ""}`);
    console.log(`Name: ${data.Name ?? ""}`);
    console.log(`Operating System: ${data.OperatingSystem ?? ""}`);
    console.log(`OSType: ${data.OSType ?? ""}`);
    console.log(`Architecture: ${data.Architecture ?? ""}`);
}

async function dockerEvents(
    socketPath: string,
    args: string[],
): Promise<void> {
    // Parse --filter, --since, --until flags
    const filterParts: string[] = [];
    let since: string | undefined;
    let until: string | undefined;

    for (let i = 0; i < args.length; i++) {
        const a = args[i];
        if ((a === "--filter" || a === "-f") && args[i + 1]) {
            filterParts.push(args[++i]);
        } else if (a.startsWith("--since=")) {
            since = a.slice("--since=".length);
        } else if (a === "--since" && args[i + 1]) {
            since = args[++i];
        } else if (a.startsWith("--until=")) {
            until = a.slice("--until=".length);
        } else if (a === "--until" && args[i + 1]) {
            until = args[++i];
        }
    }

    const params: string[] = [];
    if (filterParts.length > 0) {
        const filters: Record<string, string[]> = {};
        for (const fp of filterParts) {
            const eqIdx = fp.indexOf("=");
            if (eqIdx !== -1) {
                const key = fp.slice(0, eqIdx);
                const val = fp.slice(eqIdx + 1);
                if (!filters[key]) filters[key] = [];
                filters[key].push(val);
            }
        }
        params.push(`filters=${encodeURIComponent(JSON.stringify(filters))}`);
    }
    if (since) params.push(`since=${encodeURIComponent(since)}`);
    if (until) params.push(`until=${encodeURIComponent(until)}`);

    const query = params.length > 0 ? `?${params.join("&")}` : "";

    await requestStream(socketPath, "GET", `/events${query}`, (line) => {
        console.log(line);
    });
}

function formatSize(bytes: number): string {
    if (bytes < 1024) return `${bytes}B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)}kB`;
    if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)}MB`;
    return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)}GB`;
}

function formatMemory(bytes: number): string {
    if (bytes === 0) return "0B";
    const mib = 1048576;
    const gib = 1073741824;
    if (bytes < mib) return `${(bytes / 1024).toFixed(1)}KiB`;
    if (bytes < gib) return `${(bytes / mib).toFixed(1)}MiB`;
    return `${(bytes / gib).toFixed(1)}GiB`;
}

// ---------------------------------------------------------------------------
// Main handler
// ---------------------------------------------------------------------------

export async function handleDocker(
    socketPath: string,
    args: string[],
): Promise<void> {
    if (args.length === 0) {
        process.stderr.write("mock-docker: no command specified\n");
        process.exit(1);
    }

    switch (args[0]) {
        case "start":
            await containerAction(socketPath, "start", args.slice(1));
            break;
        case "stop":
            await containerAction(socketPath, "stop", args.slice(1));
            break;
        case "restart":
            await containerAction(socketPath, "restart", args.slice(1));
            break;
        case "exec":
            await dockerExec(socketPath, args.slice(1));
            break;
        case "ps":
            await dockerPs(socketPath, args.slice(1));
            break;
        case "rm":
            await dockerRm(socketPath, args.slice(1));
            break;
        case "kill":
            await dockerKill(socketPath, args.slice(1));
            break;
        case "pause":
            await dockerPause(socketPath, args.slice(1));
            break;
        case "unpause":
            await dockerUnpause(socketPath, args.slice(1));
            break;
        case "inspect":
            await dockerInspect(socketPath, args.slice(1));
            break;
        case "logs":
            await dockerLogs(socketPath, args.slice(1));
            break;
        case "top":
            await dockerTop(socketPath, args.slice(1));
            break;
        case "stats":
            await dockerStats(socketPath, args.slice(1));
            break;
        case "network":
            await dockerNetwork(socketPath, args.slice(1));
            break;
        case "volume":
            await dockerVolume(socketPath, args.slice(1));
            break;
        case "images":
            await dockerImages(socketPath);
            break;
        case "image":
            await dockerImage(socketPath, args.slice(1));
            break;
        case "version":
            await dockerVersion(socketPath);
            break;
        case "info":
            await dockerInfo(socketPath);
            break;
        case "events":
            await dockerEvents(socketPath, args.slice(1));
            break;
        default:
            process.stderr.write(`[mock-docker] unsupported command: ${args[0]}\n`);
            break;
    }
}
