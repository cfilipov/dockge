<template>
    <div class="shadow-box mb-3">
        <ListHeader v-model:search-text="searchText" :filter="volumeFilter" show-category-headers />

        <div ref="listRef" class="stack-list" :class="{ scrollbar: scrollbar }" :style="listStyle">
            <div v-if="filteredVolumes.length === 0" class="text-center mt-3">
                {{ $t("noContainers") }}
            </div>

            <VolumeListItem
                v-for="(item, i) in filteredVolumes"
                :key="i"
                :volume="item"
            />
        </div>
    </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from "vue";
import { useActiveScroll } from "../composables/useActiveScroll";
import ListHeader from "./ListHeader.vue";
import VolumeListItem from "./VolumeListItem.vue";
import { useSocket } from "../composables/useSocket";
import { StackFilterCategory } from "../common/util-common";
import { useFilterParams } from "../composables/useFilterParams";

defineProps<{
    scrollbar?: boolean;
}>();

const { emitAgent } = useSocket();

const searchText = ref("");
const volumeList = ref<Record<string, any>[]>([]);
const listRef = ref<HTMLElement>();

class VolumeFilter {
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

const volumeFilter = reactive(new VolumeFilter());
useFilterParams(searchText, [
    { param: "status", category: volumeFilter.status },
]);

const listStyle = computed(() => {
    return { height: "calc(100% - 60px)" };
});

function updateFilterOptions() {
    volumeFilter.status.options = {
        volumeInUse: "volumeInUse",
        volumeUnused: "volumeUnused",
    };
}

const filteredVolumes = computed(() => {
    let result = [...volumeList.value];

    updateFilterOptions();

    // Search filter
    if (searchText.value !== "") {
        const lowered = searchText.value.toLowerCase();
        result = result.filter((vol: any) => {
            return vol.name && vol.name.toLowerCase().includes(lowered);
        });
    }

    // Status filter (In Use / Unused)
    if (volumeFilter.status.isFilterSelected()) {
        result = result.filter((vol: any) => {
            const inUse = (vol.containers ?? 0) > 0;
            if (volumeFilter.status.selected.has("volumeInUse") && inUse) return true;
            if (volumeFilter.status.selected.has("volumeUnused") && !inUse) return true;
            return false;
        });
    }

    // Sort alphabetically by name
    result.sort((a: any, b: any) => {
        return (a.name || "").localeCompare(b.name || "");
    });

    return result;
});

function fetchVolumes() {
    emitAgent("", "getDockerVolumeList", (res: any) => {
        if (res.ok && res.dockerVolumeList) {
            volumeList.value = res.dockerVolumeList;
        }
    });
}

const { scrollToActive } = useActiveScroll(listRef, filteredVolumes);

defineExpose({ scrollToActive });

onMounted(() => {
    fetchVolumes();
});
</script>

<style lang="scss" scoped>
@import "../styles/list-common";
</style>
