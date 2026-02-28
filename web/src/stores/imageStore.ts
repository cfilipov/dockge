import { defineStore } from "pinia";
import { computed, ref } from "vue";
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
    const images = ref<ImageSummary[]>([]);
    const loading = ref(true);

    function setImages(data: ImageSummary[]) {
        images.value = data;
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
        loading,
        setImages,
        imagesWithStatus,
        dangling,
    };
});
