import { defineStore } from "pinia";
import { computed, reactive, ref, shallowRef } from "vue";
import { useContainerStore } from "./containerStore";
import type { DockerResourceEvent } from "./containerStore";

/** Matches the Go ImageSummary type. */
export interface ImageSummary {
    id: string;
    repoTags: string[];
    size: string;
    created: string;
    dangling: boolean;
}

export interface ImageWithStatus extends ImageSummary {
    containerCount: number;
    inUse: boolean;
}

export const useImageStore = defineStore("images", () => {
    const imageMap = reactive(new Map<string, ImageSummary>());
    const loading = ref(true);
    const lastEvent = shallowRef<DockerResourceEvent | null>(null);

    /** Sorted array of images (backward-compatible). Images keyed by ID. */
    const images = computed(() =>
        [...imageMap.values()].sort((a, b) => a.id < b.id ? -1 : a.id > b.id ? 1 : 0)
    );

    /** Merge a map update with field-level merge for existing entries. */
    function mergeImages(data: Record<string, Partial<ImageSummary> | null>) {
        for (const [key, value] of Object.entries(data)) {
            if (value === null) {
                imageMap.delete(key);
            } else {
                const existing = imageMap.get(key);
                if (existing) {
                    imageMap.set(key, { ...existing, ...value } as ImageSummary);
                } else {
                    imageMap.set(key, value as ImageSummary);
                }
            }
        }
        loading.value = false;
    }

    /** Set the last event (called from the resourceEvent channel listener). */
    function setLastEvent(evt: DockerResourceEvent) {
        lastEvent.value = evt;
    }

    /** Images enriched with container count from container store. */
    const imagesWithStatus = computed((): ImageWithStatus[] => {
        const containerStore = useContainerStore();

        return images.value.map((img) => {
            const using = containerStore.byImage(img.id);
            return {
                ...img,
                containerCount: using.length,
                inUse: using.length > 0,
            };
        });
    });

    /** Dangling images (no tags). */
    const dangling = computed(() =>
        images.value.filter((img) => img.dangling)
    );

    return {
        images,
        imageMap,
        loading,
        lastEvent,
        mergeImages,
        setLastEvent,
        imagesWithStatus,
        dangling,
    };
});
