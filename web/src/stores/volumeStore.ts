import { defineStore } from "pinia";
import { computed, reactive, ref, shallowRef } from "vue";
import { useContainerStore } from "./containerStore";
import type { DockerResourceEvent } from "./containerStore";

/** Matches the Go VolumeSummary type. */
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
    const lastEvent = shallowRef<DockerResourceEvent | null>(null);

    /** Sorted array of volumes (backward-compatible). */
    const volumes = computed(() =>
        [...volumeMap.values()].sort((a, b) => a.name < b.name ? -1 : a.name > b.name ? 1 : 0)
    );

    /** Merge a map update with field-level merge for existing entries. */
    function mergeVolumes(data: Record<string, Partial<VolumeSummary> | null>) {
        for (const [key, value] of Object.entries(data)) {
            if (value === null) {
                volumeMap.delete(key);
            } else {
                const existing = volumeMap.get(key);
                if (existing) {
                    volumeMap.set(key, { ...existing, ...value } as VolumeSummary);
                } else {
                    volumeMap.set(key, value as VolumeSummary);
                }
            }
        }
        loading.value = false;
    }

    /** Set the last event (called from the resourceEvent channel listener). */
    function setLastEvent(evt: DockerResourceEvent) {
        lastEvent.value = evt;
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
        lastEvent,
        mergeVolumes,
        setLastEvent,
        volumesWithStatus,
    };
});
