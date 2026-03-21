import { describe, test, expect, beforeAll } from "vitest";
import { resetMockState, connectClient, waitForContainerState } from "../src/helpers.js";

/**
 * Helper: wait for resourceEvent proving the Docker command completed,
 * then join the compose terminal and read the buffered output.
 */
async function getComposeTerminalOutput(
    client: Awaited<ReturnType<typeof connectClient>>,
    stackName: string,
): Promise<string> {
    await client.waitForEvent("resourceEvent");

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
    beforeAll(async () => {
        await resetMockState();
    });

    // stop runs first (containers start running), then start picks up from stopped state
    test("stopStack on unmanaged stack — uses -p flag", async () => {
        const client = await connectClient();
        try {
            await client.login();

            const stopResp = await client.sendAndReceive("stopStack", "10-unmanaged");
            expect(stopResp.ok).toBe(true);

            const output = await getComposeTerminalOutput(client, "10-unmanaged");

            expect(output).toContain("$ docker compose");
            expect(output).toContain("-p 10-unmanaged");
            expect(output).toContain("stop");
        } finally {
            client.close();
        }
    }, 30000);

    test("startStack on unmanaged stack — uses -p flag", async () => {
        const client = await connectClient();
        try {
            await client.login();

            const startResp = await client.sendAndReceive("startStack", "10-unmanaged");
            expect(startResp.ok).toBe(true);

            const output = await getComposeTerminalOutput(client, "10-unmanaged");

            expect(output).toContain("$ docker compose");
            expect(output).toContain("-p 10-unmanaged");
            expect(output).toContain("start");
        } finally {
            client.close();
        }
    }, 30000);

    test("restartStack on unmanaged stack — uses -p flag", async () => {
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

    // stopContainer runs first (portainer starts running), then startContainer picks up
    test("stopContainer on standalone container — uses docker stop", async () => {
        const client = await connectClient();
        try {
            await client.login();

            const stopResp = await client.sendAndReceive("stopContainer", "portainer");
            expect(stopResp.ok).toBe(true);

            await client.waitForEvent("resourceEvent");

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
        const client = await connectClient();
        try {
            await client.login();

            const startResp = await client.sendAndReceive("startContainer", "portainer");
            expect(startResp.ok).toBe(true);

            await client.waitForEvent("resourceEvent");

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
