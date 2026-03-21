import { describe, test, expect, beforeAll } from "vitest";
import { resetMockState, connectClient, waitForContainerState } from "../src/helpers.js";

describe("global-env", () => {
    beforeAll(async () => {
        await resetMockState();
    });

    test("startStack includes --env-file when global.env exists", async () => {
        const cmd = await connectClient();
        const obs = await connectClient();
        try {
            await cmd.login();
            await obs.login();
            await obs.waitForEvent("containers"); // drain AfterLogin

            // Set globalENV so that global.env is written to disk
            const setResp = await cmd.sendAndReceive("setSettings", {
                globalENV: "MY_VAR=hello",
            }, "");
            expect(setResp.ok).toBe(true);

            // Trigger startStack (background spawn recreates compose terminal)
            const startResp = await cmd.sendAndReceive("startStack", "test-stack");
            expect(startResp.ok).toBe(true);

            // Wait for the command to complete (container reaches running state)
            await waitForContainerState(obs, "test-stack-web-1", "running");

            // NOW join compose terminal to read buffered output (including command echo)
            const joinResp = await cmd.sendAndReceive("terminalJoin", {
                type: "compose",
                stack: "test-stack",
            });
            expect(joinResp.ok).toBe(true);
            const sessionId = joinResp.sessionId as number;

            // Read buffer replay
            let output = "";
            for (let i = 0; i < 50; i++) {
                try {
                    const data = await cmd.waitForBinary(3000);
                    if (data.length < 2) continue;
                    const gotSession = (data[0] << 8) | data[1];
                    if (gotSession !== sessionId) continue;
                    output += data.subarray(2).toString("utf-8");
                    if (output.includes("[Done]") || output.includes("[Error]")) break;
                } catch {
                    break; // timeout = no more data
                }
            }

            // The command echo line must include --env-file pointing to global.env
            expect(output).toContain("$ docker compose");
            expect(output).toContain("--env-file");
            expect(output).toContain("global.env");
        } finally {
            cmd.close();
            obs.close();
        }
    });

    test("startStack omits --env-file when no global.env", async () => {
        const cmd = await connectClient();
        const obs = await connectClient();
        try {
            await cmd.login();
            await obs.login();
            await obs.waitForEvent("containers"); // drain AfterLogin

            // Reset globalENV to empty (removes global.env from disk)
            const setResp = await cmd.sendAndReceive("setSettings", {
                globalENV: "",
            }, "");
            expect(setResp.ok).toBe(true);

            // Trigger startStack
            const startResp = await cmd.sendAndReceive("startStack", "test-stack");
            expect(startResp.ok).toBe(true);

            // Wait for the command to complete (container reaches running state)
            await waitForContainerState(obs, "test-stack-web-1", "running");

            // NOW join compose terminal to read buffered output
            const joinResp = await cmd.sendAndReceive("terminalJoin", {
                type: "compose",
                stack: "test-stack",
            });
            expect(joinResp.ok).toBe(true);
            const sessionId = joinResp.sessionId as number;

            // Read buffer replay
            let output = "";
            for (let i = 0; i < 50; i++) {
                try {
                    const data = await cmd.waitForBinary(3000);
                    if (data.length < 2) continue;
                    const gotSession = (data[0] << 8) | data[1];
                    if (gotSession !== sessionId) continue;
                    output += data.subarray(2).toString("utf-8");
                    if (output.includes("[Done]") || output.includes("[Error]")) break;
                } catch {
                    break; // timeout = no more data
                }
            }

            // The command echo line must NOT include --env-file
            const cmdLine = output.split("\n").find(l => l.includes("$ docker compose"));
            expect(cmdLine).toBeDefined();
            expect(cmdLine).not.toContain("--env-file");
        } finally {
            cmd.close();
            obs.close();
        }
    });
});
