import { defineStore } from "pinia";
import { computed, ref } from "vue";
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
    const networks = ref<NetworkSummary[]>([]);
    const loading = ref(true);

    function setNetworks(data: NetworkSummary[]) {
        networks.value = data;
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
        loading,
        setNetworks,
        networksWithStatus,
    };
});
