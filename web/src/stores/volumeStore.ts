import { defineStore } from "pinia";
import { computed, ref } from "vue";
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
    const volumes = ref<VolumeSummary[]>([]);
    const loading = ref(true);

    function setVolumes(data: VolumeSummary[]) {
        volumes.value = data;
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
        loading,
        setVolumes,
        volumesWithStatus,
    };
});
