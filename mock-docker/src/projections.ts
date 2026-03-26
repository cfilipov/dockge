// Projection functions: transform Inspect objects into List entry objects.

import type { ContainerInspect, ContainerState, ImageInspect } from "./types.js";
import type { ContainerListEntry, ImageListEntry, PortInfo } from "./list-types.js";
import type { Clock } from "./clock.js";

/**
 * Format a millisecond duration as a human-readable relative time string.
 * Picks the single largest unit: days, hours, minutes, or seconds.
 */
function formatDuration(ms: number): string {
    const seconds = Math.floor(ms / 1000);
    if (seconds < 1) return "Less than a second";
    if (seconds < 60) {
        return seconds === 1 ? "1 second" : `${seconds} seconds`;
    }
    const minutes = Math.floor(seconds / 60);
    if (minutes < 60) {
        return minutes === 1 ? "1 minute" : `${minutes} minutes`;
    }
    const hours = Math.floor(minutes / 60);
    if (hours < 24) {
        return hours === 1 ? "1 hour" : `${hours} hours`;
    }
    const days = Math.floor(hours / 24);
    return days === 1 ? "1 day" : `${days} days`;
}

/**
 * Compute the human-readable Status string (e.g. "Up 2 hours (healthy)").
 */
export function computeStatusString(state: ContainerState, clock: Clock): string {
    const now = clock.now().getTime();

    if (state.Status === "running" || state.Status === "paused") {
        const startedMs = new Date(state.StartedAt).getTime();
        const duration = formatDuration(now - startedMs);

        if (state.Status === "paused") {
            return `Up ${duration} (Paused)`;
        }

        if (state.Health) {
            const h = state.Health.Status.toLowerCase();
            if (h === "healthy") return `Up ${duration} (healthy)`;
            if (h === "unhealthy") return `Up ${duration} (unhealthy)`;
        }
        return `Up ${duration}`;
    }

    if (state.Status === "exited") {
        const finishedMs = new Date(state.FinishedAt).getTime();
        const duration = formatDuration(now - finishedMs);
        return `Exited (${state.ExitCode}) ${duration} ago`;
    }

    if (state.Status === "created") {
        return "Created";
    }

    return state.Status;
}

/**
 * Flatten NetworkSettings.Ports into a PortInfo array.
 */
function flattenPorts(ports: Record<string, Array<{ HostIp: string; HostPort: string }> | null> | undefined): PortInfo[] {
    if (!ports) return [];
    const result: PortInfo[] = [];
    for (const [key, bindings] of Object.entries(ports)) {
        const [portStr, proto] = key.split("/");
        const privatePort = parseInt(portStr, 10);
        const type = proto || "tcp";

        if (!bindings || bindings.length === 0) {
            result.push({ PrivatePort: privatePort, Type: type });
        } else {
            for (const b of bindings) {
                const entry: PortInfo = { PrivatePort: privatePort, Type: type };
                if (b.HostPort) {
                    const pubPort = parseInt(b.HostPort, 10);
                    if (!isNaN(pubPort) && pubPort >= 0 && pubPort <= 65535) {
                        entry.PublicPort = pubPort;
                    }
                }
                if (b.HostIp) {
                    entry.IP = b.HostIp;
                }
                result.push(entry);
            }
        }
    }
    return result;
}

/**
 * Project a full ContainerInspect into a ContainerListEntry.
 */
export function projectToContainerListEntry(container: ContainerInspect, clock: Clock, includeSize: boolean = false): ContainerListEntry {
    const command = container.Path + (container.Args.length > 0 ? " " + container.Args.join(" ") : "");

    const entry: ContainerListEntry = {
        Id: container.Id,
        Names: [container.Name],
        Image: container.Config.Image ?? "",
        ImageID: container.Image,
        Command: command,
        Created: Math.floor(new Date(container.Created).getTime() / 1000),
        Ports: flattenPorts(container.NetworkSettings.Ports),
        Labels: container.Config.Labels ?? {},
        State: container.State.Status,
        Status: computeStatusString(container.State, clock),
        HostConfig: { NetworkMode: container.HostConfig.NetworkMode || "default" },
        NetworkSettings: { Networks: container.NetworkSettings.Networks ?? {} },
        Mounts: container.Mounts ?? [],
    };

    if (includeSize) {
        entry.SizeRw = container.SizeRw;
        entry.SizeRootFs = container.SizeRootFs;
    }

    if (container.State.Health) {
        entry.Health = {
            Status: container.State.Health.Status,
            FailingStreak: container.State.Health.FailingStreak,
        };
    }

    return entry;
}

/**
 * Project a full ImageInspect into an ImageListEntry.
 */
export function projectToImageListEntry(image: ImageInspect, containers: Map<string, ContainerInspect>): ImageListEntry {
    let containerCount = 0;
    for (const c of containers.values()) {
        if (c.Image === image.Id) {
            containerCount++;
        }
    }

    return {
        Id: image.Id,
        ParentId: "",
        RepoTags: image.RepoTags ?? [],
        RepoDigests: image.RepoDigests ?? [],
        Created: Math.floor(new Date(image.Created).getTime() / 1000),
        Size: image.Size,
        SharedSize: -1,
        Labels: image.Config?.Labels ?? {},
        Containers: containerCount,
    };
}
