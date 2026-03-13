// List endpoint response types and filter types for Docker Engine API v1.47.
// These are the shapes returned by /containers/json, /images/json, etc.

import type { EndpointSettings, MountPoint, OCIDescriptor, ImageManifestSummary } from "./types.js";

export interface PortInfo {
    PrivatePort: number;
    PublicPort?: number;
    Type: string;
    IP?: string;
}

export interface ContainerListEntry {
    Id: string;
    Names: string[];
    Image: string;
    ImageID: string;
    Command: string;
    Created: number;
    Ports: PortInfo[];
    SizeRw?: number;
    SizeRootFs?: number;
    Labels: Record<string, string>;
    State: string;
    Status: string;
    HostConfig: { NetworkMode: string };
    NetworkSettings: { Networks: Record<string, EndpointSettings> };
    Mounts: MountPoint[];
    ImageManifestDescriptor?: OCIDescriptor;
    Health?: { Status: string; FailingStreak: number };
}

export interface ImageListEntry {
    Id: string;
    ParentId: string;
    RepoTags: string[];
    RepoDigests: string[];
    Created: number;
    Size: number;
    SharedSize: number;
    Labels: Record<string, string>;
    Containers: number;
    Manifests?: ImageManifestSummary[];
    Descriptor?: OCIDescriptor;
}

export interface DockerEvent {
    Type: string;
    Action: string;
    Actor: { ID: string; Attributes: Record<string, string> };
    time: number;       // unix epoch seconds
    timeNano: number;   // unix epoch nanoseconds
}

export type ParsedFilters = Map<string, string[]>;
