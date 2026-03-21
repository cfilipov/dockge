import { describe, test, expect } from "vitest";
import { resetMockState, connectClient, waitForContainerState } from "../src/helpers.js";

/**
 * Helper: trigger a stack action, then join the compose terminal and read
 * the buffered command echo. Returns the terminal output string.
 * The command may succeed or fail (mock CLI may not support -p flag),
 * but the echo line is written before execution starts.
 */
async function getComposeTerminalOutput(
    client: Awaited<ReturnType<typeof connectClient>>,
    stackName: string,
): Promise<string> {
    // Wait for the command to complete (or fail) — give it time
    await new Promise((r) => setTimeout(r, 3000));

    const joinResp = await client.sendAndReceive("terminalJoin", {
        type: "compose",
        stack: stackName,
    });
    expect(joinResp.ok).toBe(true);
    const sessionId = joinResp.sessionId as number;

    let output = "";
    for (let i = 0; i < 50; i++) {
        try {
            const data = await client.waitForBinary(2000);
            if (data.length < 2) continue;
            const gotSession = (data[0] << 8) | data[1];
            if (gotSession !== sessionId) continue;
            output += data.subarray(2).toString("utf-8");
            if (output.includes("[Done]") || output.includes("[Error]")) break;
        } catch {
            break;
        }
    }
    return output;
}

describe("unmanaged stacks and standalone containers", () => {
    test("stopStack on unmanaged stack — uses -p flag", async () => {
        await resetMockState();

        const client = await connectClient();
        try {
            await client.login();

            const stopResp = await client.sendAndReceive("stopStack", "10-unmanaged");
            expect(stopResp.ok).toBe(true);

            const output = await getComposeTerminalOutput(client, "10-unmanaged");

            // The echoed command must use -p flag for unmanaged stacks
            expect(output).toContain("$ docker compose");
            expect(output).toContain("-p 10-unmanaged");
            expect(output).toContain("stop");
        } finally {
            client.close();
        }
    }, 30000);

    test("startStack on unmanaged stack — uses -p flag", async () => {
        await resetMockState();

        const client = await connectClient();
        try {
            await client.login();

            const startResp = await client.sendAndReceive("startStack", "10-unmanaged");
            expect(startResp.ok).toBe(true);

            const output = await getComposeTerminalOutput(client, "10-unmanaged");

            expect(output).toContain("$ docker compose");
            expect(output).toContain("-p 10-unmanaged");
            expect(output).toContain("up");
        } finally {
            client.close();
        }
    }, 30000);

    test("restartStack on unmanaged stack — uses -p flag", async () => {
        await resetMockState();

        const client = await connectClient();
        try {
            await client.login();

            const restartResp = await client.sendAndReceive("restartStack", "10-unmanaged");
            expect(restartResp.ok).toBe(true);

            const output = await getComposeTerminalOutput(client, "10-unmanaged");

            expect(output).toContain("$ docker compose");
            expect(output).toContain("-p 10-unmanaged");
            expect(output).toContain("restart");
        } finally {
            client.close();
        }
    }, 30000);

    test("stopContainer on standalone container — uses docker stop", async () => {
        await resetMockState();

        const client = await connectClient();
        try {
            await client.login();

            const stopResp = await client.sendAndReceive("stopContainer", "portainer");
            expect(stopResp.ok).toBe(true);

            // Wait for command to complete, then read terminal buffer
            await new Promise((r) => setTimeout(r, 3000));

            const joinResp = await client.sendAndReceive("terminalJoin", {
                type: "container-action",
                container: "portainer",
            });
            expect(joinResp.ok).toBe(true);
            const sessionId = joinResp.sessionId as number;

            let output = "";
            for (let i = 0; i < 50; i++) {
                try {
                    const data = await client.waitForBinary(2000);
                    if (data.length < 2) continue;
                    const gotSession = (data[0] << 8) | data[1];
                    if (gotSession !== sessionId) continue;
                    output += data.subarray(2).toString("utf-8");
                    if (output.includes("[Done]") || output.includes("[Error]")) break;
                } catch {
                    break;
                }
            }

            expect(output).toContain("$ docker stop portainer");
        } finally {
            client.close();
        }
    }, 30000);

    test("startContainer on standalone container — uses docker start", async () => {
        await resetMockState();

        const client = await connectClient();
        try {
            await client.login();

            // Stop first
            await client.sendAndReceive("stopContainer", "portainer");
            await new Promise((r) => setTimeout(r, 2000));

            const startResp = await client.sendAndReceive("startContainer", "portainer");
            expect(startResp.ok).toBe(true);

            // Wait for command to complete, then read terminal buffer
            await new Promise((r) => setTimeout(r, 3000));

            const joinResp = await client.sendAndReceive("terminalJoin", {
                type: "container-action",
                container: "portainer",
            });
            expect(joinResp.ok).toBe(true);
            const sessionId = joinResp.sessionId as number;

            let output = "";
            for (let i = 0; i < 50; i++) {
                try {
                    const data = await client.waitForBinary(2000);
                    if (data.length < 2) continue;
                    const gotSession = (data[0] << 8) | data[1];
                    if (gotSession !== sessionId) continue;
                    output += data.subarray(2).toString("utf-8");
                    if (output.includes("[Done]") || output.includes("[Error]")) break;
                } catch {
                    break;
                }
            }

            expect(output).toContain("$ docker start portainer");
        } finally {
            client.close();
        }
    }, 30000);
});
