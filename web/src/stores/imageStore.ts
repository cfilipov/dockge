import { defineStore } from "pinia";
import { computed, reactive, ref } from "vue";
import { useContainerStore } from "./containerStore";

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

    /** Sorted array of images (backward-compatible). Images keyed by ID. */
    const images = computed(() =>
        [...imageMap.values()].sort((a, b) => a.id < b.id ? -1 : a.id > b.id ? 1 : 0)
    );

    /** Merge a map update. Null values delete the key; non-null values upsert. */
    function mergeImages(data: Record<string, ImageSummary | null>) {
        for (const [key, value] of Object.entries(data)) {
            if (value === null) {
                imageMap.delete(key);
            } else {
                imageMap.set(key, value);
            }
        }
        loading.value = false;
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
        mergeImages,
        imagesWithStatus,
        dangling,
    };
});
