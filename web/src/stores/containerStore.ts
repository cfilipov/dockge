import { defineStore } from "pinia";
import { computed, reactive, ref } from "vue";

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

    /** Sorted array of containers (backward-compatible with old array ref).
     *  Uses < comparison to match Go's lexicographic sort order. */
    const containers = computed(() =>
        [...containerMap.values()].sort((a, b) => a.name < b.name ? -1 : a.name > b.name ? 1 : 0)
    );

    /** Merge a map update. If data has replace=true, clears the store first. */
    function mergeContainers(data: Record<string, ContainerBroadcast | null> | { replace: boolean; data: Record<string, ContainerBroadcast | null> }) {
        let entries: Record<string, ContainerBroadcast | null>;
        if (typeof data === "object" && data !== null && "replace" in data && typeof (data as any).replace === "boolean") {
            const wrapper = data as { replace: boolean; data: Record<string, ContainerBroadcast | null> };
            if (wrapper.replace) {
                containerMap.clear();
            }
            entries = wrapper.data;
        } else {
            entries = data as Record<string, ContainerBroadcast | null>;
        }
        for (const [key, value] of Object.entries(entries)) {
            if (value === null) {
                containerMap.delete(key);
            } else {
                containerMap.set(key, value);
            }
        }
        loading.value = false;
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
        mergeContainers,
        byStack,
        byNetwork,
        byImage,
        byVolume,
    };
});
