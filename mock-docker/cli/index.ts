import { handleCompose } from "./compose-handler.js";
import { handleDocker } from "./docker-handler.js";

// ---------------------------------------------------------------------------
// Parse DOCKER_HOST → Unix socket path
// ---------------------------------------------------------------------------

function parseDockerHost(): string {
    const host = process.env.DOCKER_HOST || "unix:///var/run/docker.sock";
    // Strip unix:// prefix
    if (host.startsWith("unix://")) {
        return host.slice(7);
    }
    return host;
}

// ---------------------------------------------------------------------------
// Entry point
// ---------------------------------------------------------------------------

const args = process.argv.slice(2);
if (args.length === 0) {
    process.stderr.write("mock-docker: no command specified\n");
    process.exit(1);
}

const socketPath = parseDockerHost();

try {
    if (args[0] === "compose") {
        await handleCompose(socketPath, args.slice(1));
    } else {
        await handleDocker(socketPath, args);
    }
} catch (err) {
    if (err && typeof err === "object" && "code" in err && (err as NodeJS.ErrnoException).code === "ENOENT") {
        process.stderr.write(`mock-docker: cannot connect to Docker daemon at ${process.env.DOCKER_HOST}\n`);
        process.exit(125);
    }
    if (err && typeof err === "object" && "code" in err && (err as NodeJS.ErrnoException).code === "ECONNREFUSED") {
        process.stderr.write(`mock-docker: daemon not responding at ${process.env.DOCKER_HOST}\n`);
        process.exit(125);
    }
    throw err;
}
