import { defineStore } from "pinia";
import { computed, reactive, ref } from "vue";
import { useContainerStore } from "./containerStore";

/** Matches the Go NetworkSummary type (with Labels). */
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

    /** Sorted array of networks (backward-compatible). */
    const networks = computed(() =>
        [...networkMap.values()].sort((a, b) => a.name < b.name ? -1 : a.name > b.name ? 1 : 0)
    );

    /** Merge a map update. If data has replace=true, clears the store first. */
    function mergeNetworks(data: Record<string, NetworkSummary | null> | { replace: boolean; data: Record<string, NetworkSummary | null> }) {
        let entries: Record<string, NetworkSummary | null>;
        if (typeof data === "object" && data !== null && "replace" in data && typeof (data as any).replace === "boolean") {
            const wrapper = data as { replace: boolean; data: Record<string, NetworkSummary | null> };
            if (wrapper.replace) {
                networkMap.clear();
            }
            entries = wrapper.data;
        } else {
            entries = data as Record<string, NetworkSummary | null>;
        }
        for (const [key, value] of Object.entries(entries)) {
            if (value === null) {
                networkMap.delete(key);
            } else {
                networkMap.set(key, value);
            }
        }
        loading.value = false;
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
        mergeNetworks,
        networksWithStatus,
    };
});
