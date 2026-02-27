<template>
    <div class="shadow-box mb-3">
        <ListHeader v-model:search-text="searchText" :filter="imageFilter" show-category-headers />

        <div ref="listRef" class="stack-list" :class="{ scrollbar: scrollbar }" :style="listStyle">
            <div v-if="filteredImages.length === 0" class="text-center mt-3">
                {{ $t("noContainers") }}
            </div>

            <ImageListItem
                v-for="(item, i) in filteredImages"
                :key="i"
                :image="item"
            />
        </div>
    </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from "vue";
import { useActiveScroll } from "../composables/useActiveScroll";
import ListHeader from "./ListHeader.vue";
import ImageListItem from "./ImageListItem.vue";
import { useSocket } from "../composables/useSocket";
import { StackFilterCategory } from "../common/util-common";
import { useFilterParams } from "../composables/useFilterParams";

defineProps<{
    scrollbar?: boolean;
}>();

const { emitAgent } = useSocket();

const searchText = ref("");
const imageList = ref<Record<string, any>[]>([]);
const listRef = ref<HTMLElement>();

class ImageFilter {
    status = new StackFilterCategory("status");

    get categories() {
        return [this.status];
    }

    isFilterSelected(): boolean {
        return this.categories.some(c => c.isFilterSelected());
    }

    clear() {
        this.categories.forEach(c => c.clear());
    }
}

const imageFilter = reactive(new ImageFilter());
useFilterParams(searchText, [
    { param: "status", category: imageFilter.status },
]);

const listStyle = computed(() => {
    return { height: "calc(100% - 60px)" };
});

function updateFilterOptions() {
    imageFilter.status.options = {
        imageInUse: "imageInUse",
        imageUnused: "imageUnused",
        imageDangling: "imageDangling",
    };
}

const filteredImages = computed(() => {
    let result = [...imageList.value];

    updateFilterOptions();

    // Search filter — also match truncated sha256 for dangling images
    if (searchText.value !== "") {
        const lowered = searchText.value.toLowerCase();
        result = result.filter((img: any) => {
            const tags = img.repoTags || [];
            if (tags.some((t: string) => t.toLowerCase().includes(lowered))) return true;
            if (img.id && img.id.toLowerCase().includes(lowered)) return true;
            return false;
        });
    }

    // Status filter (In Use / Unused / Dangling)
    if (imageFilter.status.isFilterSelected()) {
        result = result.filter((img: any) => {
            const dangling = img.dangling === true;
            const inUse = (img.containers ?? 0) > 0;
            if (imageFilter.status.selected.has("imageDangling") && dangling) return true;
            if (imageFilter.status.selected.has("imageInUse") && !dangling && inUse) return true;
            if (imageFilter.status.selected.has("imageUnused") && !dangling && !inUse) return true;
            return false;
        });
    }

    // Sort: tagged images alphabetically by first tag, dangling at end by ID
    result.sort((a: any, b: any) => {
        const aDangling = a.dangling === true;
        const bDangling = b.dangling === true;
        if (aDangling !== bDangling) return aDangling ? 1 : -1;
        const nameA = a.repoTags?.[0] || a.id || "";
        const nameB = b.repoTags?.[0] || b.id || "";
        return nameA.localeCompare(nameB);
    });

    return result;
});

function fetchImages() {
    emitAgent("", "getDockerImageList", (res: any) => {
        if (res.ok && res.dockerImageList) {
            imageList.value = res.dockerImageList;
        }
    });
}

const { scrollToActive } = useActiveScroll(listRef, filteredImages);

defineExpose({ scrollToActive });

onMounted(() => {
    fetchImages();
});
</script>

<style lang="scss" scoped>
@import "../styles/list-common";
</style>
