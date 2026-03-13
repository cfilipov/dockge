import { describe, it, expect, beforeEach } from "vitest";
import { MockState } from "../src/state.js";
import { FixedClock } from "../src/clock.js";
import { EventEmitter } from "../src/events.js";
import type { ContainerInspect, ImageInspect } from "../src/types.js";
import { execCreate, execInspect } from "../src/mutations.js";

// ---------------------------------------------------------------------------
// Test fixture
// ---------------------------------------------------------------------------

interface TestEnv {
    state: MockState;
    clock: FixedClock;
    emitter: EventEmitter;
    runningContainer: ContainerInspect;
    stoppedContainer: ContainerInspect;
}

function makeTestState(): TestEnv {
    const state = new MockState();
    const clock = new FixedClock(new Date("2025-01-01T00:00:00Z"));
    const emitter = new EventEmitter();

    const image: ImageInspect = {
        Id: "sha256:aabbccddee00112233445566778899aabbccddee00112233445566778899aabb",
        RepoTags: ["alpine:latest"],
        RepoDigests: [],
        Created: "2025-01-01T00:00:00Z",
        Architecture: "amd64",
        Os: "linux",
        Size: 10_000_000,
        RootFS: { Type: "layers" },
    };
    state.images.set(image.Id, image);

    const runningContainer: ContainerInspect = {
        Id: "ctr_running_001aabbccddee00112233445566778899aabbccddee0011223344",
        Created: "2025-01-01T00:00:00Z",
        Path: "/bin/sh",
        Args: [],
        State: {
            Status: "running",
            Running: true,
            Paused: false,
            Restarting: false,
            OOMKilled: false,
            Dead: false,
            Pid: 12345,
            ExitCode: 0,
            Error: "",
            StartedAt: "2025-01-01T00:00:00Z",
            FinishedAt: "0001-01-01T00:00:00Z",
        },
        Image: image.Id,
        Name: "/test-runner",
        HostConfig: {
            NetworkMode: "bridge",
            RestartPolicy: { Name: "", MaximumRetryCount: 0 },
        },
        Mounts: [],
        Config: {
            Image: "alpine:latest",
            Hostname: "testhost",
            Labels: {},
        },
        NetworkSettings: { Networks: {} },
    };
    state.containers.set(runningContainer.Id, runningContainer);

    const stoppedContainer: ContainerInspect = {
        Id: "ctr_stopped_001aabbccddee00112233445566778899aabbccddee001122334455",
        Created: "2025-01-01T00:00:00Z",
        Path: "/bin/sh",
        Args: [],
        State: {
            Status: "exited",
            Running: false,
            Paused: false,
            Restarting: false,
            OOMKilled: false,
            Dead: false,
            Pid: 0,
            ExitCode: 0,
            Error: "",
            StartedAt: "2024-12-31T23:00:00Z",
            FinishedAt: "2024-12-31T23:30:00Z",
        },
        Image: image.Id,
        Name: "/test-stopped",
        HostConfig: {
            NetworkMode: "bridge",
            RestartPolicy: { Name: "", MaximumRetryCount: 0 },
        },
        Mounts: [],
        Config: {
            Image: "alpine:latest",
            Labels: {},
        },
        NetworkSettings: { Networks: {} },
    };
    state.containers.set(stoppedContainer.Id, stoppedContainer);

    return { state, clock, emitter, runningContainer, stoppedContainer };
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("exec mutations", () => {
    let env: TestEnv;

    beforeEach(() => {
        env = makeTestState();
    });

    describe("execCreate", () => {
        it("returns an exec ID", () => {
            const result = execCreate(env.state, env.runningContainer.Id, {
                Cmd: ["/bin/sh"],
                AttachStdin: true,
                AttachStdout: true,
                AttachStderr: true,
                Tty: true,
            }, env.clock);

            expect("ok" in result).toBe(true);
            if ("ok" in result) {
                expect(result.ok.Id).toBeTruthy();
                expect(typeof result.ok.Id).toBe("string");
                expect(result.ok.Id.length).toBe(64);
            }
        });

        it("adds exec ID to container ExecIDs", () => {
            const result = execCreate(env.state, env.runningContainer.Id, {
                Cmd: ["/bin/sh"],
                AttachStdin: true,
                AttachStdout: true,
                AttachStderr: true,
                Tty: false,
            }, env.clock);

            expect("ok" in result).toBe(true);
            if ("ok" in result) {
                const c = env.state.containers.get(env.runningContainer.Id)!;
                expect(c.ExecIDs).toContain(result.ok.Id);
            }
        });

        it("stores exec session in state", () => {
            const result = execCreate(env.state, env.runningContainer.Id, {
                Cmd: ["echo", "hello"],
                AttachStdin: false,
                AttachStdout: true,
                AttachStderr: true,
                Tty: false,
            }, env.clock);

            expect("ok" in result).toBe(true);
            if ("ok" in result) {
                const exec = env.state.execSessions.get(result.ok.Id);
                expect(exec).toBeTruthy();
                expect(exec!.ContainerID).toBe(env.runningContainer.Id);
                expect(exec!.Running).toBe(false);
                expect(exec!.ProcessConfig?.entrypoint).toBe("echo");
                expect(exec!.ProcessConfig?.arguments).toEqual(["hello"]);
            }
        });

        it("fails on stopped container", () => {
            const result = execCreate(env.state, env.stoppedContainer.Id, {
                Cmd: ["/bin/sh"],
                AttachStdin: true,
                AttachStdout: true,
                AttachStderr: true,
                Tty: false,
            }, env.clock);

            expect("error" in result).toBe(true);
            if ("error" in result) {
                expect(result.statusCode).toBe(409);
            }
        });

        it("fails on unknown container", () => {
            const result = execCreate(env.state, "nonexistent", {
                Cmd: ["/bin/sh"],
                AttachStdin: true,
                AttachStdout: true,
                AttachStderr: true,
                Tty: false,
            }, env.clock);

            expect("error" in result).toBe(true);
            if ("error" in result) {
                expect(result.statusCode).toBe(404);
            }
        });
    });

    describe("execInspect", () => {
        it("returns exec session info", () => {
            const createResult = execCreate(env.state, env.runningContainer.Id, {
                Cmd: ["/bin/sh"],
                AttachStdin: true,
                AttachStdout: true,
                AttachStderr: true,
                Tty: true,
            }, env.clock);

            expect("ok" in createResult).toBe(true);
            if ("ok" in createResult) {
                const result = execInspect(env.state, createResult.ok.Id);
                expect("ok" in result).toBe(true);
                if ("ok" in result) {
                    expect(result.ok.ID).toBe(createResult.ok.Id);
                    expect(result.ok.ContainerID).toBe(env.runningContainer.Id);
                    expect(result.ok.Running).toBe(false);
                    expect(result.ok.ExitCode).toBe(0);
                    expect(result.ok.OpenStdin).toBe(true);
                    expect(result.ok.OpenStdout).toBe(true);
                    expect(result.ok.ProcessConfig?.tty).toBe(true);
                }
            }
        });

        it("returns 404 for unknown exec ID", () => {
            const result = execInspect(env.state, "nonexistent-exec-id");
            expect("error" in result).toBe(true);
            if ("error" in result) {
                expect(result.statusCode).toBe(404);
            }
        });
    });
});
