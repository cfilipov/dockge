import { defineStore } from "pinia";
import { computed, reactive, ref, shallowRef } from "vue";

/** Event metadata from Docker event broadcasts. */
export interface DockerResourceEvent {
    type: "container" | "network" | "image" | "volume";
    action: string;
    id: string;
    name: string;
    stackName: string;
    serviceName: string;
    containerId: string;
    timeNano: number;
}

/** Matches the Go ContainerBroadcast type. */
export interface ContainerBroadcast {
    name: string;
    containerId: string;
    serviceName: string;
    stackName: string;
    state: string;
    health: string;
    image: string;
    imageId: string;
    networks: Record<string, { ipv4: string; ipv6: string; mac: string }>;
    mounts: { name: string; type: string }[];
    ports: { hostPort: number; containerPort: number; protocol: string }[];
}

export const useContainerStore = defineStore("containers", () => {
    const containerMap = reactive(new Map<string, ContainerBroadcast>());
    const loading = ref(true);
    const lastEvent = shallowRef<DockerResourceEvent | null>(null);

    /** Sorted array of containers (backward-compatible with old array ref).
     *  Uses < comparison to match Go's lexicographic sort order. */
    const containers = computed(() =>
        [...containerMap.values()].sort((a, b) => a.name < b.name ? -1 : a.name > b.name ? 1 : 0)
    );

    /** Merge a map update with field-level merge for existing entries.
     *  Null values delete the key; partial objects merge into existing; full objects replace. */
    function mergeContainers(data: Record<string, Partial<ContainerBroadcast> | null>) {
        for (const [key, value] of Object.entries(data)) {
            if (value === null) {
                containerMap.delete(key);
            } else {
                const existing = containerMap.get(key);
                if (existing) {
                    containerMap.set(key, { ...existing, ...value } as ContainerBroadcast);
                } else {
                    containerMap.set(key, value as ContainerBroadcast);
                }
            }
        }
        loading.value = false;
    }

    /** Set the last event (called from the resourceEvent channel listener). */
    function setLastEvent(evt: DockerResourceEvent) {
        lastEvent.value = evt;
    }

    /** Containers belonging to a specific compose project (stack). */
    function byStack(stackName: string): ContainerBroadcast[] {
        const result: ContainerBroadcast[] = [];
        for (const c of containerMap.values()) {
            if (c.stackName === stackName) result.push(c);
        }
        return result;
    }

    /** Containers connected to a specific network. */
    function byNetwork(networkName: string): ContainerBroadcast[] {
        const result: ContainerBroadcast[] = [];
        for (const c of containerMap.values()) {
            if (networkName in (c.networks || {})) result.push(c);
        }
        return result;
    }

    /** Containers using a specific image (by image ID). */
    function byImage(imageId: string): ContainerBroadcast[] {
        const result: ContainerBroadcast[] = [];
        for (const c of containerMap.values()) {
            if (c.imageId === imageId) result.push(c);
        }
        return result;
    }

    /** Containers using a specific volume (by mount name). */
    function byVolume(volumeName: string): ContainerBroadcast[] {
        const result: ContainerBroadcast[] = [];
        for (const c of containerMap.values()) {
            if ((c.mounts || []).some((m) => m.name === volumeName)) result.push(c);
        }
        return result;
    }

    return {
        containers,
        containerMap,
        loading,
        lastEvent,
        mergeContainers,
        setLastEvent,
        byStack,
        byNetwork,
        byImage,
        byVolume,
    };
});
