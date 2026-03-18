import { defineStore } from "pinia";
import { computed, reactive, ref, shallowRef } from "vue";
import { useContainerStore } from "./containerStore";
import type { DockerResourceEvent } from "./containerStore";

/** Matches the Go NetworkSummary type. */
export interface NetworkSummary {
    name: string;
    id: string;
    driver: string;
    scope: string;
    internal: boolean;
    attachable: boolean;
    ingress: boolean;
    labels: Record<string, string>;
}

export interface NetworkWithStatus extends NetworkSummary {
    inUse: boolean;
    containerCount: number;
    stackName: string;
}

export const useNetworkStore = defineStore("networks", () => {
    const networkMap = reactive(new Map<string, NetworkSummary>());
    const loading = ref(true);
    const lastEvent = shallowRef<DockerResourceEvent | null>(null);

    /** Sorted array of networks (backward-compatible). */
    const networks = computed(() =>
        [...networkMap.values()].sort((a, b) => a.name < b.name ? -1 : a.name > b.name ? 1 : 0)
    );

    /** Merge a map update with field-level merge for existing entries. */
    function mergeNetworks(data: Record<string, Partial<NetworkSummary> | null>) {
        for (const [key, value] of Object.entries(data)) {
            if (value === null) {
                networkMap.delete(key);
            } else {
                const existing = networkMap.get(key);
                if (existing) {
                    networkMap.set(key, { ...existing, ...value } as NetworkSummary);
                } else {
                    networkMap.set(key, value as NetworkSummary);
                }
            }
        }
        loading.value = false;
    }

    /** Set the last event (called from the resourceEvent channel listener). */
    function setLastEvent(evt: DockerResourceEvent) {
        lastEvent.value = evt;
    }

    /** Networks enriched with in-use status from container store. */
    const networksWithStatus = computed((): NetworkWithStatus[] => {
        const containerStore = useContainerStore();

        return networks.value.map((n) => {
            const using = containerStore.byNetwork(n.name);
            return {
                ...n,
                inUse: using.length > 0,
                containerCount: using.length,
                stackName: n.labels?.["com.docker.compose.project"] || "",
            };
        });
    });

    return {
        networks,
        networkMap,
        loading,
        lastEvent,
        mergeNetworks,
        setLastEvent,
        networksWithStatus,
    };
});
