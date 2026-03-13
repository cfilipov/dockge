import type { Route } from "../server.js";
import { sendJSON, sendError } from "../server.js";
import { execInspect } from "../mutations.js";
import type { MockState } from "../state.js";
import type { ExecInspect } from "../types.js";
import { createShellSession, processCommand, getPrompt } from "../shell.js";
import { frameOutput } from "../stream.js";

const SHELL_COMMANDS = new Set(["/bin/sh", "/bin/bash", "sh", "bash"]);

function isShellCmd(cmd: string[]): boolean {
    return cmd.length === 1 && SHELL_COMMANDS.has(cmd[0]);
}

export const execRoutes: Route[] = [
    {
        method: "POST",
        pattern: "/exec/:id/start",
        handler: async ({ req, res, params, state, clock }) => {
            const exec = state.execSessions.get(params.id);
            if (!exec) {
                sendError(res, 404, `No such exec instance: ${params.id}`);
                return;
            }

            const container = state.containers.get(exec.ContainerID);
            if (!container) {
                sendError(res, 404, `No such container: ${exec.ContainerID}`);
                return;
            }

            // Parse config from the exec session (set at create time).
            // The start request body may contain { Detach, Tty } overrides,
            // but we must not eagerly consume the stream — for interactive
            // shells, stdin data follows the JSON config on the same
            // connection.  Instead, we parse the JSON prefix incrementally
            // and hand any leftover bytes to the shell's line reader.

            const tty = exec.ProcessConfig?.tty ?? false;
            const cmd = [exec.ProcessConfig?.entrypoint || "/bin/sh", ...(exec.ProcessConfig?.arguments || [])];

            exec.Running = true;
            exec.Pid = container.State.Pid + 1;

            const contentType = tty
                ? "application/vnd.docker.raw-stream"
                : "application/vnd.docker.multiplexed-stream";
            res.writeHead(200, { "Content-Type": contentType });

            const writeOutput = (text: string) => {
                if (tty) {
                    res.write(text + "\n");
                } else {
                    res.write(frameOutput(text));
                }
            };

            const writePrompt = (prompt: string) => {
                if (tty) {
                    res.write(prompt);
                } else {
                    const payload = Buffer.from(prompt, "utf-8");
                    const header = Buffer.alloc(8);
                    header[0] = 1;
                    header.writeUInt32BE(payload.length, 4);
                    res.write(Buffer.concat([header, payload]));
                }
            };

            const finishExec = () => {
                exec.Running = false;
                exec.ExitCode = 0;
                exec.Pid = 0;
                cleanupExec(state, exec);
            };

            if (isShellCmd(cmd)) {
                // Interactive shell session
                const session = createShellSession(container, clock);
                writePrompt(getPrompt(session));

                // Incremental stdin parser: accumulates bytes, extracts
                // the JSON config prefix (if any), then processes shell
                // input line by line.
                let buffer = "";
                let jsonParsed = false;

                const processLines = () => {
                    let newlineIdx: number;
                    while ((newlineIdx = buffer.indexOf("\n")) !== -1) {
                        const line = buffer.slice(0, newlineIdx);
                        buffer = buffer.slice(newlineIdx + 1);

                        const output = processCommand(session, line);
                        if (output === null) {
                            finishExec();
                            res.end();
                            return;
                        }
                        if (output) writeOutput(output);
                        writePrompt(getPrompt(session));
                    }
                };

                req.on("data", (chunk: Buffer) => {
                    buffer += chunk.toString();

                    // The Docker client sends a small JSON body
                    // ({ Detach, Tty }) before streaming stdin.  Skip
                    // over it so the shell only sees actual commands.
                    if (!jsonParsed) {
                        const trimmed = buffer.trimStart();
                        if (trimmed.startsWith("{")) {
                            const closeIdx = trimmed.indexOf("}");
                            if (closeIdx === -1) return; // wait for more data
                            // Discard the JSON object, keep the rest
                            const jsonEnd = buffer.indexOf("}") + 1;
                            buffer = buffer.slice(jsonEnd);
                        }
                        jsonParsed = true;
                    }

                    processLines();
                });

                req.on("end", () => {
                    // Process any remaining buffered input
                    if (buffer.trim()) {
                        const output = processCommand(session, buffer.trim());
                        if (output !== null && output) writeOutput(output);
                    }
                    finishExec();
                    res.end();
                });

                req.on("close", () => {
                    finishExec();
                });
            } else {
                // One-shot command: consume body first, then execute.
                const bodyChunks: Buffer[] = [];
                await new Promise<void>((resolve) => {
                    req.on("data", (chunk: Buffer) => bodyChunks.push(chunk));
                    req.on("end", () => resolve());
                });

                const session = createShellSession(container, clock);
                const fullCmd = cmd.join(" ");
                const output = processCommand(session, fullCmd);
                if (output !== null && output !== "") {
                    writeOutput(output);
                }
                finishExec();
                res.end();
            }
        },
    },
    {
        method: "GET",
        pattern: "/exec/:id/json",
        handler: async ({ res, params, state }) => {
            const result = execInspect(state, params.id);
            if ("error" in result) {
                sendError(res, result.statusCode, result.error);
                return;
            }
            sendJSON(res, 200, result.ok);
        },
    },
];

function cleanupExec(state: MockState, exec: ExecInspect): void {
    const container = state.containers.get(exec.ContainerID);
    if (container?.ExecIDs) {
        const idx = container.ExecIDs.indexOf(exec.ID);
        if (idx !== -1) container.ExecIDs.splice(idx, 1);
    }
}
