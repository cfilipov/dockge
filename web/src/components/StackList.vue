<template>
    <div class="shadow-box mb-3">
        <ListHeader v-model:search-text="searchText" :filter="stackFilter" />

        <div ref="stackListRef" class="stack-list" :class="{ scrollbar: scrollbar }" :style="stackListStyle">
            <div v-if="flatStackList.length === 0" class="text-center mt-3">
                <router-link to="/stacks/new">{{ $t("addFirstStackMsg") }}</router-link>
            </div>

            <StackListItem
                v-for="item in flatStackList"
                :key="item.name"
                :stack="item"
                :isSelectMode="selectMode"
                :isSelected="isSelected"
                :select="select"
                :deselect="deselect"
            />
        </div>
    </div>

    <Confirm ref="confirmPauseRef" :yes-text="$t('Yes')" :no-text="$t('No')" @yes="pauseSelected">
        {{ $t("pauseStackMsg") }}
    </Confirm>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch } from "vue";
import { useActiveScroll } from "../composables/useActiveScroll";
import Confirm from "../components/Confirm.vue";
import ListHeader from "./ListHeader.vue";
import StackListItem from "../components/StackListItem.vue";
import { useSocket } from "../composables/useSocket";
import { useStackStore } from "../stores/stackStore";
import { CREATED_FILE, CREATED_STACK, EXITED, RUNNING, RUNNING_AND_EXITED, UNHEALTHY, UNKNOWN, StackFilter, StackStatusInfo } from "../common/util-common";
import { useFilterParams } from "../composables/useFilterParams";

defineProps<{
    scrollbar?: boolean;
}>();

const stackStore = useStackStore();
const { getSocket } = useSocket();

const searchText = ref("");
const selectMode = ref(false);
const selectAll = ref(false);
const disableSelectAllWatcher = ref(false);
const selectedStacks = ref<Record<string, boolean>>({});
const stackFilter = reactive(new StackFilter());
useFilterParams(searchText, [
    { param: "status", category: stackFilter.status },
    { param: "attr", category: stackFilter.attributes },
]);
const stackListRef = ref<HTMLElement>();
const confirmPauseRef = ref<InstanceType<typeof Confirm>>();

const combinedStacks = computed((): any[] => [...stackStore.allStacks]);

// Keep filter options in sync with the stack list (outside computed to avoid side effects)
watch(combinedStacks, (list) => {
    updateFilterOptions(list);
}, { immediate: true });

const filteredStacks = computed(() => {
    let result = [...combinedStacks.value];

    // filter
    result = result.filter(stack => {
        // search text
        let searchTextMatch = true;
        if (searchText.value !== "") {
            const lowered = searchText.value.toLowerCase();
            searchTextMatch =
                stack.name.toLowerCase().includes(lowered) ||
                (stack.tags && stack.tags.find((tag: any) =>
                    tag.name.toLowerCase().includes(lowered) ||
                    tag.value?.toLowerCase().includes(lowered)
                ));
        }

        // status filter
        let statusMatch = true;
        if (stackFilter.status.isFilterSelected()) {
            const statusLabel = StackStatusInfo.get(stack.status).label;
            statusMatch = stackFilter.status.selected.has(statusLabel);
        }

        // attribute filter
        let attributeMatch = true;
        if (stackFilter.attributes.isFilterSelected()) {
            attributeMatch = false;
            for (const attribute of stackFilter.attributes.selected) {
                if (attribute === "unmanaged") {
                    if (!stack.isManagedByDockge) {
                        attributeMatch = true;
                    }
                } else if (stack[attribute] === true) {
                    attributeMatch = true;
                }
            }
        }

        return searchTextMatch && statusMatch && attributeMatch;
    });

    // sort
    result.sort((m1: any, m2: any) => {
        if (m1.isManagedByDockge && !m2.isManagedByDockge) return -1;
        if (!m1.isManagedByDockge && m2.isManagedByDockge) return 1;

        if (m1.status !== m2.status) {
            if (m2.status === UNHEALTHY) return 1;
            if (m1.status === UNHEALTHY) return -1;
            if (m2.status === RUNNING) return 1;
            if (m1.status === RUNNING) return -1;
            if (m2.status === RUNNING_AND_EXITED) return 1;
            if (m1.status === RUNNING_AND_EXITED) return -1;
            if (m2.status === EXITED) return 1;
            if (m1.status === EXITED) return -1;
            if (m2.status === CREATED_STACK) return 1;
            if (m1.status === CREATED_STACK) return -1;
            if (m2.status === CREATED_FILE) return 1;
            if (m1.status === CREATED_FILE) return -1;
            if (m2.status === UNKNOWN) return 1;
            if (m1.status === UNKNOWN) return -1;
        }
        return m1.name.localeCompare(m2.name);
    });

    return result;
});

const flatStackList = computed(() => filteredStacks.value);

const stackListStyle = computed(() => {
    let listHeaderHeight = 60;
    if (selectMode.value) listHeaderHeight += 42;
    return { height: `calc(100% - ${listHeaderHeight}px)` };
});

function updateFilterOptions(_stacks: any[]) {
    // Build status options from StackStatusInfo
    const statusOptions: Record<string, string> = {};
    for (const info of StackStatusInfo.ALL) {
        statusOptions[info.label] = info.label;
    }
    stackFilter.status.options = statusOptions;

    // Attribute filter options
    stackFilter.attributes.options = {
        imageUpdatesAvailable: "imageUpdatesAvailable",
        unmanaged: "unmanaged",
    };
}

function deselect(id: string) {
    delete selectedStacks.value[id];
}

function select(id: string) {
    selectedStacks.value[id] = true;
}

function isSelected(id: string) {
    return id in selectedStacks.value;
}

function cancelSelectMode() {
    selectMode.value = false;
    selectedStacks.value = {};
}

function pauseDialog() {
    confirmPauseRef.value?.show();
}

function pauseSelected() {
    Object.keys(selectedStacks.value)
        .filter(id => combinedStacks.value.find((s: any) => s.name === id)?.started)
        .forEach(id => getSocket().emit("pauseStack", id, () => {}));
    cancelSelectMode();
}

function resumeSelected() {
    Object.keys(selectedStacks.value)
        .filter(id => !combinedStacks.value.find((s: any) => s.name === id)?.started)
        .forEach(id => getSocket().emit("resumeStack", id, () => {}));
    cancelSelectMode();
}

watch(searchText, () => {
    for (let stack of flatStackList.value) {
        if (!selectedStacks.value[stack.id]) {
            if (selectAll.value) {
                disableSelectAllWatcher.value = true;
                selectAll.value = false;
            }
            break;
        }
    }
});

watch(selectAll, () => {
    if (!disableSelectAllWatcher.value) {
        selectedStacks.value = {};
        if (selectAll.value) {
            flatStackList.value.forEach((item: any) => {
                selectedStacks.value[item.id] = true;
            });
        }
    } else {
        disableSelectAllWatcher.value = false;
    }
});

watch(selectMode, () => {
    if (!selectMode.value) {
        selectAll.value = false;
        selectedStacks.value = {};
    }
});

const { scrollToActive } = useActiveScroll(stackListRef, flatStackList);

defineExpose({ scrollToActive });
</script>

<style lang="scss" scoped>
@import "../styles/vars.scss";
@import "../styles/list-common";

.small-padding {
    padding-left: 5px !important;
    padding-right: 5px !important;
}

.stack-item {
    width: 100%;
}

.tags {
    margin-top: 4px;
    padding-left: 67px;
    display: flex;
    flex-wrap: wrap;
    gap: 0;
}

.bottom-style {
    padding-left: 67px;
    margin-top: 5px;
}

.selection-controls {
    margin-top: 5px;
    display: flex;
    align-items: center;
    gap: 10px;
}

.agent-select {
    cursor: pointer;
    font-size: 14px;
    font-weight: 500;
    color: $dark-font-color3;
    padding-left: 10px;
    padding-right: 10px;
    display: flex;
    align-items: center;
    user-select: none;
}
</style>
