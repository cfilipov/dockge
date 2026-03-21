import { spawn, ChildProcess } from "child_process";
import { existsSync, mkdirSync, rmSync } from "fs";
import path from "path";

const PORT = 5053;
const PROJECT_ROOT = path.resolve(import.meta.dirname, "../..");
const RUN_DIR = path.join(PROJECT_ROOT, ".run", `ws-test-${PORT}`);
const DATA_DIR = path.join(RUN_DIR, "data");
const STACKS_DIR = path.join(RUN_DIR, "stacks");

let mockDaemon: ChildProcess | undefined;
let dockgeServer: ChildProcess | undefined;

function waitForFile(filePath: string, timeoutMs = 5000): Promise<void> {
    return new Promise((resolve, reject) => {
        const start = Date.now();
        const check = () => {
            if (existsSync(filePath)) {
                resolve();
                return;
            }
            if (Date.now() - start > timeoutMs) {
                reject(new Error(`Timeout waiting for ${filePath}`));
                return;
            }
            setTimeout(check, 100);
        };
        check();
    });
}

function waitForHealthz(url: string, timeoutMs = 10000): Promise<void> {
    return new Promise((resolve, reject) => {
        const start = Date.now();
        const check = async () => {
            try {
                const resp = await fetch(url);
                if (resp.ok) {
                    resolve();
                    return;
                }
            } catch {
                // Server not ready yet
            }
            if (Date.now() - start > timeoutMs) {
                reject(new Error(`Timeout waiting for ${url}`));
                return;
            }
            setTimeout(check, 200);
        };
        check();
    });
}

export async function setup(): Promise<void> {
    // Support external backend
    if (process.env.TEST_BACKEND === "external") {
        return;
    }

    // Clean and create run directory
    rmSync(RUN_DIR, { recursive: true, force: true });
    mkdirSync(DATA_DIR, { recursive: true });
    mkdirSync(STACKS_DIR, { recursive: true });

    const mockSock = path.join(RUN_DIR, "docker.sock");
    const mockDaemonBin = path.join(PROJECT_ROOT, "bin", "mock-daemon");
    const dockgeBin = process.env.DOCKGE_BIN
        ? path.resolve(process.env.DOCKGE_BIN)
        : path.join(PROJECT_ROOT, "bin", "dockge");
    const stacksSource = path.join(PROJECT_ROOT, "ws-test", "stacks");
    const imagesJSON = path.join(PROJECT_ROOT, "mock-docker", "scripts", "images.json");

    if (!existsSync(mockDaemonBin)) {
        throw new Error(`mock-daemon binary not found at ${mockDaemonBin}. Run: task build:mock-docker-daemon`);
    }
    if (!existsSync(dockgeBin)) {
        throw new Error(`dockge binary not found at ${dockgeBin}. Run: task build-go (or set DOCKGE_BIN)`);
    }

    // Start mock daemon
    mockDaemon = spawn(
        mockDaemonBin,
        [
            "--socket", mockSock,
            "--stacks-source", stacksSource,
            "--stacks-dir", STACKS_DIR,
            "--images-json", imagesJSON,
        ],
        {
            stdio: ["ignore", "pipe", "pipe"],
        },
    );
    mockDaemon.stdout?.on("data", (d: Buffer) => {
        if (process.env.DEBUG) process.stdout.write(`[mock-daemon] ${d}`);
    });
    mockDaemon.stderr?.on("data", (d: Buffer) => {
        if (process.env.DEBUG) process.stderr.write(`[mock-daemon] ${d}`);
    });

    await waitForFile(mockSock);

    // Start dockge server
    const dockgeArgs = [
        "--dev",
        "--port", String(PORT),
        "--data-dir", DATA_DIR,
        "--stacks-dir", STACKS_DIR,
    ];
    if (process.env.DOCKGE_NO_AUTH === "1") {
        dockgeArgs.push("--no-auth");
    }
    dockgeServer = spawn(
        dockgeBin,
        dockgeArgs,
        {
            env: {
                ...process.env,
                DOCKER_HOST: `unix://${mockSock}`,
                PATH: `${path.join(PROJECT_ROOT, "bin")}:${process.env.PATH}`,
            },
            stdio: ["ignore", "pipe", "pipe"],
        },
    );
    dockgeServer.stdout?.on("data", (d: Buffer) => {
        if (process.env.DEBUG) process.stdout.write(`[dockge] ${d}`);
    });
    dockgeServer.stderr?.on("data", (d: Buffer) => {
        if (process.env.DEBUG) process.stderr.write(`[dockge] ${d}`);
    });

    await waitForHealthz(`http://localhost:${PORT}/healthz`);
}

export async function teardown(): Promise<void> {
    if (process.env.TEST_BACKEND === "external") {
        return;
    }

    if (dockgeServer) {
        dockgeServer.kill("SIGTERM");
        dockgeServer = undefined;
    }
    if (mockDaemon) {
        mockDaemon.kill("SIGTERM");
        mockDaemon = undefined;
    }

    // Small delay to let processes exit
    await new Promise((r) => setTimeout(r, 500));

    rmSync(RUN_DIR, { recursive: true, force: true });
}
