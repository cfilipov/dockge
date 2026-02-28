<template>
    <div class="shadow-box mb-3">
        <ListHeader v-model:search-text="searchText" :filter="containerFilter" />

        <div ref="listRef" class="stack-list" :class="{ scrollbar: scrollbar }" :style="listStyle">
            <div v-if="filteredContainers.length === 0" class="text-center mt-3">
                {{ $t("noContainers") }}
            </div>

            <ContainerListItem
                v-for="(item, i) in filteredContainers"
                :key="i"
                :container="item"
            />
        </div>
    </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed } from "vue";
import ListHeader from "./ListHeader.vue";
import ContainerListItem from "./ContainerListItem.vue";
import { useActiveScroll } from "../composables/useActiveScroll";
import { useContainerStore } from "../stores/containerStore";
import { useStackStore } from "../stores/stackStore";
import { useUpdateStore } from "../stores/updateStore";
import { StackFilter, ContainerStatusInfo } from "../common/util-common";
import { useFilterParams } from "../composables/useFilterParams";

defineProps<{
    scrollbar?: boolean;
}>();

const containerStore = useContainerStore();
const stackStore = useStackStore();
const updateStore = useUpdateStore();

const searchText = ref("");
const containerFilter = reactive(new StackFilter());
useFilterParams(searchText, [
    { param: "status", category: containerFilter.status },
    { param: "attr", category: containerFilter.attributes },
]);
const listRef = ref<HTMLElement>();

const listStyle = computed(() => {
    return { height: "calc(100% - 60px)" };
});

function updateFilterOptions() {
    const statusOptions: Record<string, string> = {};
    for (const info of ContainerStatusInfo.ALL) {
        statusOptions[info.label] = info.label;
    }
    containerFilter.status.options = statusOptions;

    // Same attribute option as the stack list
    containerFilter.attributes.options = { imageUpdatesAvailable: "imageUpdatesAvailable" };
}

// Enrich containers with stack-level fields for ContainerListItem
const enrichedContainers = computed(() => {
    return containerStore.containers.map(c => {
        const stackEntry = stackStore.rawStacks.find(s => s.name === c.stackName);
        const composeImage = stackEntry?.images[c.serviceName];
        return {
            ...c,
            isManagedByDockge: !!stackEntry,
            recreateNecessary: !!(composeImage && c.image && c.image !== composeImage),
            imageUpdatesAvailable: updateStore.hasUpdate(`${c.stackName}/${c.serviceName}`),
        };
    });
});

const filteredContainers = computed(() => {
    let result = [...enrichedContainers.value];

    // Populate filter options
    updateFilterOptions();

    // Search text filter
    if (searchText.value !== "") {
        const lowered = searchText.value.toLowerCase();
        result = result.filter((c: any) =>
            c.name.toLowerCase().includes(lowered) ||
            c.serviceName.toLowerCase().includes(lowered) ||
            (c.stackName || "").toLowerCase().includes(lowered)
        );
    }

    // Status filter
    if (containerFilter.status.isFilterSelected()) {
        result = result.filter((c: any) => {
            const label = ContainerStatusInfo.from(c).label;
            return containerFilter.status.selected.has(label);
        });
    }

    // Attribute filter
    if (containerFilter.attributes.isFilterSelected()) {
        result = result.filter((c: any) => {
            for (const attribute of containerFilter.attributes.selected) {
                if (c[attribute] === true) {
                    return true;
                }
            }
            return false;
        });
    }

    // Sort: running first, then exited, then others; alphabetical within same state
    const stateOrder: Record<string, number> = {
        running: 0,
        paused: 1,
        exited: 2,
        dead: 3,
        created: 4,
    };

    result.sort((a: any, b: any) => {
        const aOrder = stateOrder[a.state] ?? 5;
        const bOrder = stateOrder[b.state] ?? 5;
        if (aOrder !== bOrder) return aOrder - bOrder;
        return a.name.localeCompare(b.name);
    });

    return result;
});

const { scrollToActive } = useActiveScroll(listRef, filteredContainers);

defineExpose({ scrollToActive });
</script>

<style lang="scss" scoped>
@import "../styles/list-common";
</style>
