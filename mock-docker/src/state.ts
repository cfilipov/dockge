import type {
    ContainerInspect,
    NetworkInspect,
    VolumeInspect,
    ImageInspect,
    ExecInspect,
} from "./types.js";

export class MockState {
    containers: Map<string, ContainerInspect>;
    networks: Map<string, NetworkInspect>;
    volumes: Map<string, VolumeInspect>;
    images: Map<string, ImageInspect>;
    execSessions: Map<string, ExecInspect>;

    constructor() {
        this.containers = new Map();
        this.networks = new Map();
        this.volumes = new Map();
        this.images = new Map();
        this.execSessions = new Map();
    }

    clear(): void {
        this.containers.clear();
        this.networks.clear();
        this.volumes.clear();
        this.images.clear();
        this.execSessions.clear();
    }
}
