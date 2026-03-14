import type {
    ContainerInspect,
    NetworkInspect,
    VolumeInspect,
    ImageInspect,
    ExecInspect,
} from "./types.js";
import type { LogTemplates } from "./log-templates.js";

export class MockState {
    containers: Map<string, ContainerInspect>;
    networks: Map<string, NetworkInspect>;
    volumes: Map<string, VolumeInspect>;
    images: Map<string, ImageInspect>;
    execSessions: Map<string, ExecInspect>;
    logTemplates: LogTemplates | null;

    constructor() {
        this.containers = new Map();
        this.networks = new Map();
        this.volumes = new Map();
        this.images = new Map();
        this.execSessions = new Map();
        this.logTemplates = null;
    }

    clear(): void {
        this.containers.clear();
        this.networks.clear();
        this.volumes.clear();
        this.images.clear();
        this.execSessions.clear();
        // logTemplates is intentionally NOT cleared — it's loaded from source, not runtime state
    }
}
