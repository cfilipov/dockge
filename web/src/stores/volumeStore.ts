import { defineStore } from "pinia";
import { computed, reactive, ref } from "vue";
import { useContainerStore } from "./containerStore";

/** Matches the Go VolumeSummary type (with Labels). */
export interface VolumeSummary {
    name: string;
    driver: string;
    mountpoint: string;
    labels: Record<string, string>;
}

export interface VolumeWithStatus extends VolumeSummary {
    inUse: boolean;
    containerCount: number;
    stackName: string;
}

export const useVolumeStore = defineStore("volumes", () => {
    const volumeMap = reactive(new Map<string, VolumeSummary>());
    const loading = ref(true);

    /** Sorted array of volumes (backward-compatible). */
    const volumes = computed(() =>
        [...volumeMap.values()].sort((a, b) => a.name < b.name ? -1 : a.name > b.name ? 1 : 0)
    );

    /** Merge a map update. If data has replace=true, clears the store first. */
    function mergeVolumes(data: Record<string, VolumeSummary | null> | { replace: boolean; data: Record<string, VolumeSummary | null> }) {
        let entries: Record<string, VolumeSummary | null>;
        if (typeof data === "object" && data !== null && "replace" in data && typeof (data as any).replace === "boolean") {
            const wrapper = data as { replace: boolean; data: Record<string, VolumeSummary | null> };
            if (wrapper.replace) {
                volumeMap.clear();
            }
            entries = wrapper.data;
        } else {
            entries = data as Record<string, VolumeSummary | null>;
        }
        for (const [key, value] of Object.entries(entries)) {
            if (value === null) {
                volumeMap.delete(key);
            } else {
                volumeMap.set(key, value);
            }
        }
        loading.value = false;
    }

    /** Volumes enriched with in-use status from container store. */
    const volumesWithStatus = computed((): VolumeWithStatus[] => {
        const containerStore = useContainerStore();

        return volumes.value.map((v) => {
            const using = containerStore.byVolume(v.name);
            return {
                ...v,
                inUse: using.length > 0,
                containerCount: using.length,
                stackName: v.labels?.["com.docker.compose.project"] || "",
            };
        });
    });

    return {
        volumes,
        volumeMap,
        loading,
        mergeVolumes,
        volumesWithStatus,
    };
});
