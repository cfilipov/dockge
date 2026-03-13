import { describe, it, expect, beforeEach } from "vitest";
import { FixedClock } from "../src/clock.js";
import type { ContainerInspect } from "../src/types.js";
import { createShellSession, processCommand, getPrompt } from "../src/shell.js";
import type { ShellSession } from "../src/shell.js";

// ---------------------------------------------------------------------------
// Test fixture
// ---------------------------------------------------------------------------

function makeContainer(overrides: Partial<ContainerInspect> = {}): ContainerInspect {
    return {
        Id: "abc123def456abc123def456abc123def456abc123def456abc123def456abcd",
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
        Image: "sha256:imageaabbcc",
        Name: "/test-container",
        HostConfig: {
            NetworkMode: "bridge",
            RestartPolicy: { Name: "", MaximumRetryCount: 0 },
        },
        Mounts: [],
        Config: {
            Hostname: "testhost",
            Image: "alpine:latest",
            Env: ["PATH=/usr/local/bin:/usr/bin:/bin", "HOME=/root", "TERM=xterm"],
            Labels: {
                "com.docker.compose.project": "testproject",
                "com.docker.compose.service": "web",
            },
        },
        NetworkSettings: {
            Networks: {
                bridge: {
                    NetworkID: "net001",
                    EndpointID: "ep001",
                    Gateway: "172.17.0.1",
                    IPAddress: "172.17.0.2",
                    IPPrefixLen: 16,
                },
            },
        },
        ...overrides,
    };
}

let clock: FixedClock;
let container: ContainerInspect;
let session: ShellSession;

beforeEach(() => {
    clock = new FixedClock(new Date("2025-01-15T12:00:00Z"));
    container = makeContainer();
    session = createShellSession(container, clock);
});

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("shell", () => {
    describe("echo", () => {
        it("echoes arguments", () => {
            expect(processCommand(session, "echo hello world")).toBe("hello world");
        });

        it("echoes empty string with no args", () => {
            expect(processCommand(session, "echo")).toBe("");
        });
    });

    describe("whoami", () => {
        it("returns root by default", () => {
            expect(processCommand(session, "whoami")).toBe("root");
        });

        it("returns container user when set", () => {
            const c = makeContainer({ Config: { ...container.Config, User: "appuser" } });
            const s = createShellSession(c, clock);
            expect(processCommand(s, "whoami")).toBe("appuser");
        });
    });

    describe("hostname", () => {
        it("returns container hostname", () => {
            expect(processCommand(session, "hostname")).toBe("testhost");
        });
    });

    describe("pwd and cd", () => {
        it("starts at WorkingDir or /", () => {
            expect(processCommand(session, "pwd")).toBe("/");
        });

        it("starts at WorkingDir when set", () => {
            const c = makeContainer({ Config: { ...container.Config, WorkingDir: "/app" } });
            const s = createShellSession(c, clock);
            expect(processCommand(s, "pwd")).toBe("/app");
        });

        it("cd changes directory", () => {
            processCommand(session, "cd /tmp");
            expect(processCommand(session, "pwd")).toBe("/tmp");
        });

        it("cd .. goes up", () => {
            processCommand(session, "cd /usr/local");
            processCommand(session, "cd ..");
            expect(processCommand(session, "pwd")).toBe("/usr");
        });

        it("cd with no args goes to /", () => {
            processCommand(session, "cd /tmp");
            processCommand(session, "cd");
            expect(processCommand(session, "pwd")).toBe("/");
        });

        it("relative cd appends to cwd", () => {
            processCommand(session, "cd /usr");
            processCommand(session, "cd local");
            expect(processCommand(session, "pwd")).toBe("/usr/local");
        });
    });

    describe("env and printenv", () => {
        it("env returns all env vars", () => {
            const result = processCommand(session, "env")!;
            expect(result).toContain("PATH=/usr/local/bin:/usr/bin:/bin");
            expect(result).toContain("HOME=/root");
            expect(result).toContain("TERM=xterm");
        });

        it("printenv returns all env vars", () => {
            const result = processCommand(session, "printenv")!;
            expect(result).toContain("PATH=");
        });

        it("printenv VAR returns specific value", () => {
            expect(processCommand(session, "printenv PATH")).toBe("/usr/local/bin:/usr/bin:/bin");
        });

        it("printenv unknown returns empty", () => {
            expect(processCommand(session, "printenv NONEXISTENT")).toBe("");
        });
    });

    describe("cat", () => {
        it("cat /etc/hostname returns hostname", () => {
            expect(processCommand(session, "cat /etc/hostname")).toBe("testhost");
        });

        it("cat /etc/hosts includes localhost and container IP", () => {
            const result = processCommand(session, "cat /etc/hosts")!;
            expect(result).toContain("127.0.0.1\tlocalhost");
            expect(result).toContain("172.17.0.2\ttesthost");
        });

        it("cat /etc/resolv.conf returns DNS config", () => {
            const result = processCommand(session, "cat /etc/resolv.conf")!;
            expect(result).toContain("nameserver 127.0.0.11");
        });

        it("cat /etc/resolv.conf uses custom DNS", () => {
            const c = makeContainer({
                HostConfig: { ...container.HostConfig, Dns: ["8.8.8.8", "8.8.4.4"] },
            });
            const s = createShellSession(c, clock);
            const result = processCommand(s, "cat /etc/resolv.conf")!;
            expect(result).toContain("nameserver 8.8.8.8");
            expect(result).toContain("nameserver 8.8.4.4");
        });

        it("cat unknown file returns error", () => {
            expect(processCommand(session, "cat /nonexistent")).toBe("cat: /nonexistent: No such file or directory");
        });
    });

    describe("uname", () => {
        it("uname returns Linux", () => {
            expect(processCommand(session, "uname")).toBe("Linux");
        });

        it("uname -a includes hostname", () => {
            const result = processCommand(session, "uname -a")!;
            expect(result).toContain("Linux");
            expect(result).toContain("testhost");
            expect(result).toContain("5.15.0-mock");
        });
    });

    describe("ps", () => {
        it("ps aux returns process list", () => {
            const result = processCommand(session, "ps aux")!;
            expect(result).toContain("USER");
            expect(result).toContain("PID");
            expect(result).toContain("COMMAND");
        });
    });

    describe("id", () => {
        it("returns root uid/gid", () => {
            expect(processCommand(session, "id")).toBe("uid=0(root) gid=0(root) groups=0(root)");
        });

        it("returns custom user uid/gid", () => {
            const c = makeContainer({ Config: { ...container.Config, User: "app" } });
            const s = createShellSession(c, clock);
            const result = processCommand(s, "id")!;
            expect(result).toMatch(/^uid=\d+\(app\) gid=\d+\(app\) groups=\d+\(app\)$/);
        });
    });

    describe("date", () => {
        it("returns a date string", () => {
            const result = processCommand(session, "date")!;
            expect(result).toBeTruthy();
            // Should contain 2025 since our clock is set to 2025
            expect(result).toContain("2025");
        });
    });

    describe("uptime", () => {
        it("returns uptime string", () => {
            const result = processCommand(session, "uptime")!;
            expect(result).toContain("up");
            expect(result).toContain("load average");
        });
    });

    describe("free", () => {
        it("shows memory info", () => {
            const result = processCommand(session, "free -m")!;
            expect(result).toContain("Mem:");
            expect(result).toContain("Swap:");
        });
    });

    describe("df", () => {
        it("shows disk info", () => {
            const result = processCommand(session, "df -h")!;
            expect(result).toContain("Filesystem");
            expect(result).toContain("Mounted on");
            expect(result).toContain("overlay");
        });
    });

    describe("ls", () => {
        it("returns directory listing", () => {
            const result = processCommand(session, "ls")!;
            expect(result).toContain("bin");
            expect(result).toContain("etc");
            expect(result).toContain("usr");
        });

        it("ls -la returns detailed listing", () => {
            const result = processCommand(session, "ls -la")!;
            expect(result).toContain("total");
            expect(result).toContain("drwxr-xr-x");
            expect(result).toContain("root");
        });

        it("ls /etc lists etc directory", () => {
            const result = processCommand(session, "ls /etc")!;
            expect(result).toContain("hostname");
            expect(result).toContain("hosts");
        });
    });

    describe("exit", () => {
        it("returns null", () => {
            expect(processCommand(session, "exit")).toBeNull();
        });
    });

    describe("unknown command", () => {
        it("returns command not found", () => {
            expect(processCommand(session, "foobar")).toBe("bash: foobar: command not found");
        });
    });

    describe("empty input", () => {
        it("returns empty string", () => {
            expect(processCommand(session, "")).toBe("");
            expect(processCommand(session, "   ")).toBe("");
        });
    });

    describe("determinism", () => {
        it("same container produces same output for all commands", () => {
            const s1 = createShellSession(makeContainer(), clock);
            const s2 = createShellSession(makeContainer(), clock);

            expect(processCommand(s1, "ls")).toBe(processCommand(s2, "ls"));
            expect(processCommand(s1, "ls -la")).toBe(processCommand(s2, "ls -la"));
            expect(processCommand(s1, "df -h")).toBe(processCommand(s2, "df -h"));
            expect(processCommand(s1, "free -m")).toBe(processCommand(s2, "free -m"));
            expect(processCommand(s1, "ps aux")).toBe(processCommand(s2, "ps aux"));
            expect(processCommand(s1, "id")).toBe(processCommand(s2, "id"));
        });
    });

    describe("getPrompt", () => {
        it("returns root prompt with #", () => {
            expect(getPrompt(session)).toBe("root@testhost:/# ");
        });

        it("returns user prompt with $", () => {
            const c = makeContainer({ Config: { ...container.Config, User: "app" } });
            const s = createShellSession(c, clock);
            expect(getPrompt(s)).toBe("app@testhost:/$ ");
        });

        it("updates after cd", () => {
            processCommand(session, "cd /tmp");
            expect(getPrompt(session)).toBe("root@testhost:/tmp# ");
        });
    });
});
