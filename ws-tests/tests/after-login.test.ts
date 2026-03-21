import { describe, test, expect, beforeAll } from "vitest";
import WebSocket from "ws";
import { resetMockState, connectClient } from "../src/helpers.js";

const BASE_URL = process.env.TEST_WS_URL ?? "ws://localhost:5053/ws";
const isNoAuth = !!process.env.DOCKGE_NO_AUTH;

/** Open a raw WebSocket and drain the initial "info" event. */
async function openRawWs(): Promise<WebSocket> {
    const ws = new WebSocket(BASE_URL);
    await new Promise<void>((resolve, reject) => {
        ws.once("open", resolve);
        ws.once("error", reject);
    });
    // Drain the info event (sent on connect, before auth)
    await new Promise<void>((resolve) => {
        ws.once("message", () => resolve());
    });
    return ws;
}

describe("after-login", () => {
    beforeAll(async () => {
        await resetMockState();
    });

    // --- Login event ordering ---
    // The ack must arrive before the initial data sends so the frontend has
    // loggedIn=true and the dashboard mounted before data events arrive.

    test("login ack arrives before initial data events", async () => {
        if (isNoAuth) {
            // No-auth mode: server sends info + 6 data events unprompted on
            // connect. Open a raw WS and register the handler immediately so
            // nothing is lost.
            const ws = new WebSocket(BASE_URL);
            await new Promise<void>((resolve, reject) => {
                ws.once("open", resolve);
                ws.once("error", reject);
            });

            const received: Array<{ type: "event"; name: string }> = [];
            const done = new Promise<void>((resolve) => {
                const expectedEvents = new Set([
                    "stacks", "containers", "networks", "images", "volumes", "updates",
                ]);
                const seen = new Set<string>();

                ws.on("message", (raw: Buffer) => {
                    const msg = JSON.parse(raw.toString());
                    if (typeof msg.event === "string") {
                        received.push({ type: "event", name: msg.event });
                        if (expectedEvents.has(msg.event)) {
                            seen.add(msg.event);
                        }
                        if (seen.size === expectedEvents.size) {
                            resolve();
                        }
                    }
                });
            });

            await done;
            ws.close();

            // info must be the first event
            expect(received[0].name).toBe("info");

            // All six data events must be present
            for (const name of ["stacks", "containers", "networks", "images", "volumes", "updates"]) {
                const idx = received.findIndex(r => r.name === name);
                expect(idx, `"${name}" event should arrive in no-auth mode`).toBeGreaterThanOrEqual(0);
            }
            return;
        }

        // Auth mode: login ack must arrive before data events
        const ws = await openRawWs();
        const received: Array<{ type: "ack" | "event"; name: string }> = [];

        const done = new Promise<void>((resolve) => {
            const expectedEvents = new Set([
                "stacks", "containers", "networks", "images", "volumes", "updates",
            ]);
            const seen = new Set<string>();

            ws.on("message", (raw: Buffer) => {
                const msg = JSON.parse(raw.toString());

                if (typeof msg.id === "number") {
                    received.push({ type: "ack", name: "login" });
                } else if (typeof msg.event === "string") {
                    received.push({ type: "event", name: msg.event });
                    if (expectedEvents.has(msg.event)) {
                        seen.add(msg.event);
                    }
                    if (seen.size === expectedEvents.size) {
                        resolve();
                    }
                }
            });
        });

        ws.send(JSON.stringify({ id: 1, event: "login", args: ["admin", "testpass123", "", ""] }));

        await done;
        ws.close();

        // The first entry must be the login ack
        expect(received[0]).toEqual({ type: "ack", name: "login" });

        // All six data events must come after the ack
        const ackIndex = received.findIndex(r => r.type === "ack" && r.name === "login");
        for (const name of ["stacks", "containers", "networks", "images", "volumes", "updates"]) {
            const idx = received.findIndex(r => r.type === "event" && r.name === name);
            expect(idx, `"${name}" event should arrive after login ack`).toBeGreaterThan(ackIndex);
        }
    });

    test("loginByToken ack arrives before initial data events", async () => {
        if (isNoAuth) {
            // No-auth mode: data events arrive on connect unprompted.
            // Same verification as the login test above.
            const ws = new WebSocket(BASE_URL);
            await new Promise<void>((resolve, reject) => {
                ws.once("open", resolve);
                ws.once("error", reject);
            });

            const received: Array<{ type: "event"; name: string }> = [];
            const done = new Promise<void>((resolve) => {
                const expected = new Set(["stacks", "containers", "networks", "images", "volumes", "updates"]);
                const seen = new Set<string>();
                ws.on("message", (raw: Buffer) => {
                    const msg = JSON.parse(raw.toString());
                    if (typeof msg.event === "string" && expected.has(msg.event)) {
                        received.push({ type: "event", name: msg.event });
                        seen.add(msg.event);
                        if (seen.size === expected.size) resolve();
                    }
                });
            });
            await done;
            ws.close();
            expect(received.length).toBe(6);
            return;
        }

        // Auth mode: original test
        // Get a token first
        const client = await connectClient();
        const token = await client.login();
        client.close();

        const ws = await openRawWs();

        const received: Array<{ type: "ack" | "event"; name: string }> = [];

        const done = new Promise<void>((resolve) => {
            const seen = new Set<string>();
            const expectedEvents = new Set([
                "stacks", "containers", "networks", "images", "volumes", "updates",
            ]);

            ws.on("message", (raw: Buffer) => {
                const msg = JSON.parse(raw.toString());
                if (typeof msg.id === "number") {
                    received.push({ type: "ack", name: "loginByToken" });
                } else if (typeof msg.event === "string") {
                    received.push({ type: "event", name: msg.event });
                    if (expectedEvents.has(msg.event)) {
                        seen.add(msg.event);
                    }
                    if (seen.size === expectedEvents.size) {
                        resolve();
                    }
                }
            });
        });

        ws.send(JSON.stringify({ id: 1, event: "loginByToken", args: [token] }));

        await done;
        ws.close();

        expect(received[0]).toEqual({ type: "ack", name: "loginByToken" });
    });

    // --- Container data shape ---
    // The initial containers send must include all fields the frontend expects:
    // name, containerId, serviceName, stackName, state, health, image, imageId,
    // networks, mounts, ports.

    test("containers data has correct fields", async () => {
        const client = await connectClient();
        try {
            await client.login();
            const containers = await client.waitForEvent("containers");

            expect(Object.keys(containers).length).toBeGreaterThan(0);

            // Pick the first container
            const first = Object.values(containers)[0] as Record<string, unknown>;

            // Required string fields
            for (const field of ["name", "containerId", "serviceName", "stackName", "state", "health", "image", "imageId"]) {
                expect(first, `missing field "${field}"`).toHaveProperty(field);
                expect(typeof first[field], `"${field}" should be string`).toBe("string");
            }

            // Required structured fields
            expect(first).toHaveProperty("networks");
            expect(typeof first.networks).toBe("object");
            expect(first.networks).not.toBeNull();

            expect(first).toHaveProperty("mounts");
            expect(Array.isArray(first.mounts)).toBe(true);

            expect(first).toHaveProperty("ports");
            expect(Array.isArray(first.ports)).toBe(true);
        } finally {
            client.close();
        }
    });

    test("containers network entries have ipv4, ipv6, mac", async () => {
        const client = await connectClient();
        try {
            await client.login();
            const containers = await client.waitForEvent("containers");

            // Find a container that has at least one network
            const withNet = Object.values(containers).find((c: any) =>
                c.networks && Object.keys(c.networks).length > 0
            ) as Record<string, unknown> | undefined;

            expect(withNet, "should have at least one container with networks").toBeTruthy();

            const networks = withNet!.networks as Record<string, Record<string, unknown>>;
            const firstNet = Object.values(networks)[0];

            for (const field of ["ipv4", "ipv6", "mac"]) {
                expect(firstNet, `missing network field "${field}"`).toHaveProperty(field);
                expect(typeof firstNet[field], `"${field}" should be string`).toBe("string");
            }
        } finally {
            client.close();
        }
    });

    test("containers ports field is an array with correct entry shape", async () => {
        const client = await connectClient();
        try {
            await client.login();
            const containers = await client.waitForEvent("containers");

            // Every container must have ports as an array
            for (const [name, c] of Object.entries(containers)) {
                const container = c as Record<string, unknown>;
                expect(Array.isArray(container.ports), `${name}.ports should be an array`).toBe(true);

                // If ports are present, verify the entry shape
                const ports = container.ports as Array<Record<string, unknown>>;
                for (const port of ports) {
                    expect(port).toHaveProperty("hostPort");
                    expect(port).toHaveProperty("containerPort");
                    expect(port).toHaveProperty("protocol");
                    expect(typeof port.hostPort).toBe("number");
                    expect(typeof port.containerPort).toBe("number");
                    expect(typeof port.protocol).toBe("string");
                }
            }
        } finally {
            client.close();
        }
    });

    test("containers health field reflects healthcheck status", async () => {
        const client = await connectClient();
        try {
            await client.login();
            const containers = await client.waitForEvent("containers");

            // All containers should have health as a string (may be empty)
            for (const [name, c] of Object.entries(containers)) {
                const container = c as Record<string, unknown>;
                expect(typeof container.health, `${name}.health should be string`).toBe("string");
                // health must be one of: "", "healthy", "unhealthy", "starting"
                expect(
                    ["", "healthy", "unhealthy", "starting"].includes(container.health as string),
                    `${name}.health="${container.health}" is not a valid health value`,
                ).toBe(true);
            }
        } finally {
            client.close();
        }
    });

    // --- Volumes deserialization ---
    // The mock daemon must include Options:{} in volume JSON so bollard can
    // deserialize it. If Options is missing, the volumes event would never arrive.

    test("volumes event arrives without deserialization error", async () => {
        // This test verifies that the volumes event is sent at all.
        // If the mock daemon omits the Options field from volume JSON,
        // bollard fails to deserialize and the event never arrives.
        const client = await connectClient();
        try {
            await client.login();
            const volumes = await client.waitForEvent("volumes");

            // The event should arrive (not timeout). The map may be empty
            // if no stacks define volumes, but the event itself must exist.
            expect(volumes).toBeDefined();
            expect(typeof volumes).toBe("object");

            // If any volumes exist, verify field shape
            for (const [name, v] of Object.entries(volumes)) {
                const vol = v as Record<string, unknown>;
                expect(vol, `${name} should have "name"`).toHaveProperty("name");
                expect(vol, `${name} should have "driver"`).toHaveProperty("driver");
                expect(vol, `${name} should have "mountpoint"`).toHaveProperty("mountpoint");
                expect(typeof vol.name).toBe("string");
                expect(typeof vol.driver).toBe("string");
            }
        } finally {
            client.close();
        }
    });

    // --- Updates channel format ---
    // The updates channel sends a raw array, not wrapped in {"items": ...}.
    // The test client auto-unwraps items for objects, so we test with raw WS
    // to verify the wire format.

    test("updates event is a raw array, not wrapped in items", async () => {
        const ws = new WebSocket(BASE_URL);
        await new Promise<void>((resolve, reject) => {
            ws.once("open", resolve);
            ws.once("error", reject);
        });

        const updatesData = new Promise<unknown>((resolve) => {
            ws.on("message", (raw: Buffer) => {
                const msg = JSON.parse(raw.toString());
                if (msg.event === "updates") {
                    resolve(msg.data);
                }
            });
        });

        ws.send(JSON.stringify({ id: 1, event: "login", args: ["admin", "testpass123", "", ""] }));

        const data = await updatesData;
        ws.close();

        // Must be an array, not an object with items key
        expect(Array.isArray(data), "updates data should be an array").toBe(true);
    });
});
