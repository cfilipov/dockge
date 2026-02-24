<template>
    <div class="shadow-box mb-3" :style="boxStyle">
        <div class="list-header">
            <div class="header-top">
                <div class="d-flex flex-grow-1 align-items-center">
                    <a v-if="searchText == ''" class="search-icon">
                        <font-awesome-icon icon="search" />
                    </a>
                    <a v-if="searchText != ''" class="search-icon" style="cursor: pointer" @click="clearSearchText">
                        <font-awesome-icon icon="times" />
                    </a>
                    <input v-model="searchText" class="form-control search-input" autocomplete="off" />
                </div>

                <BDropdown variant="link" placement="bottom-end" menu-class="filter-dropdown" toggle-class="filter-icon-container" no-caret>
                    <template #button-content>
                        <font-awesome-icon class="filter-icon" :class="{ 'filter-icon-active': stackFilter.isFilterSelected() }" icon="filter" />
                    </template>

                    <BDropdownItemButton :disabled="!stackFilter.isFilterSelected()" button-class="filter-dropdown-clear" @click="stackFilter.clear()">
                        <font-awesome-icon class="ms-1 me-2" icon="times" />{{ $t("clearFilter") }}
                    </BDropdownItemButton>

                    <BDropdownDivider></BDropdownDivider>

                    <template v-for="category in stackFilter.categories" :key="category.label">
                        <BDropdownGroup v-if="category.hasOptions()" :header="$t(category.label)">
                            <BDropdownForm v-for="(value, key) in category.options" :key="key" form-class="filter-option" @change="category.toggleSelected(value)" @click.stop>
                                <BFormCheckbox :checked="category.selected.has(value)">{{ $t(key) }}</BFormCheckbox>
                            </BDropdownForm>
                        </BDropdownGroup>
                    </template>
                </BDropdown>
            </div>
        </div>

        <div ref="stackListRef" class="stack-list" :class="{ scrollbar: scrollbar }" :style="stackListStyle">
            <div v-if="flatStackList.length === 0" class="text-center mt-3">
                <router-link to="/stacks/new">{{ $t("addFirstStackMsg") }}</router-link>
            </div>

            <div class="stack-list-inner" v-for="(agent, index) in agentStackList" :key="index">
                <div v-if="agentCount > 1"
                     class="p-2 agent-select"
                     @click="closedAgents.set(agent.endpoint, !closedAgents.get(agent.endpoint))">
                    <span class="me-1">
                        <font-awesome-icon v-show="closedAgents.get(agent.endpoint)" icon="chevron-circle-right" />
                        <font-awesome-icon v-show="!closedAgents.get(agent.endpoint)" icon="chevron-circle-down" />
                    </span>
                    <span v-if="agent.endpoint === 'current'">{{ $t("currentEndpoint") }}</span>
                    <span v-else>{{ agent.endpoint }}</span>
                </div>

                <StackListItem
                    v-show="agentCount === 1 || !closedAgents.get(agent.endpoint)"
                    v-for="(item, i) in agent.stacks"
                    :key="i"
                    :stack="item"
                    :isSelectMode="selectMode"
                    :isSelected="isSelected"
                    :select="select"
                    :deselect="deselect"
                />
            </div>
        </div>
    </div>

    <Confirm ref="confirmPauseRef" :yes-text="$t('Yes')" :no-text="$t('No')" @yes="pauseSelected">
        {{ $t("pauseStackMsg") }}
    </Confirm>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch, onMounted, onBeforeUnmount, nextTick } from "vue";
import Confirm from "../components/Confirm.vue";
import StackListItem from "../components/StackListItem.vue";
import { useSocket } from "../composables/useSocket";
import { CREATED_FILE, CREATED_STACK, EXITED, RUNNING, RUNNING_AND_EXITED, UNHEALTHY, UNKNOWN, StackFilter, StackStatusInfo } from "../../../common/util-common";

defineProps<{
    scrollbar?: boolean;
}>();

const { completeStackList, agentCount, stackList, getSocket } = useSocket();

const searchText = ref("");
const selectMode = ref(false);
const selectAll = ref(false);
const disableSelectAllWatcher = ref(false);
const selectedStacks = ref<Record<string, boolean>>({});
const windowTop = ref(0);
const stackFilter = reactive(new StackFilter());
const closedAgents = reactive(new Map<string, boolean>());
const stackListRef = ref<HTMLElement>();
const confirmPauseRef = ref<InstanceType<typeof Confirm>>();

const boxStyle = computed(() => {
    if (window.innerWidth > 550) {
        return { height: `calc(100vh - 160px + ${windowTop.value}px)` };
    } else {
        return { height: "calc(100vh - 160px)" };
    }
});

const agentStackList = computed(() => {
    let result = Object.values(completeStackList.value) as any[];

    // Populate filter options from current data
    updateFilterOptions(result);

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

        // agent filter
        let agentMatch = true;
        if (stackFilter.agents.isFilterSelected()) {
            const endpoint = stack.endpoint || "current";
            agentMatch = stackFilter.agents.selected.has(endpoint);
        }

        // attribute filter
        let attributeMatch = true;
        if (stackFilter.attributes.isFilterSelected()) {
            attributeMatch = false;
            for (const attribute of stackFilter.attributes.selected) {
                if (stack[attribute] === true) {
                    attributeMatch = true;
                }
            }
        }

        return searchTextMatch && statusMatch && agentMatch && attributeMatch;
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

    // group by endpoint with 'current' first, others alphabetical
    const groups = [
        ...result.reduce((acc: Map<string, any[]>, stack: any) => {
            const endpoint = stack.endpoint || 'current';
            if (!acc.has(endpoint)) acc.set(endpoint, []);
            acc.get(endpoint)!.push(stack);
            return acc;
        }, new Map()).entries()
    ].map(([endpoint, stacks]) => ({ endpoint, stacks }));

    groups.sort((a, b) => {
        if (a.endpoint === 'current' && b.endpoint !== 'current') return -1;
        if (a.endpoint !== 'current' && b.endpoint === 'current') return 1;
        return a.endpoint.localeCompare(b.endpoint);
    });

    return groups;
});

const flatStackList = computed(() => {
    return agentStackList.value.flatMap((g: any) => g.stacks);
});

const stackListStyle = computed(() => {
    let listHeaderHeight = 60;
    if (selectMode.value) listHeaderHeight += 42;
    return { height: `calc(100% - ${listHeaderHeight}px)` };
});

function updateFilterOptions(stacks: any[]) {
    // Build status options from StackStatusInfo
    const statusOptions: Record<string, string> = {};
    for (const info of StackStatusInfo.ALL) {
        statusOptions[info.label] = info.label;
    }
    stackFilter.status.options = statusOptions;

    // Build agent options from current stacks
    if (agentCount.value > 1) {
        const agentOptions: Record<string, string> = {};
        for (const stack of stacks) {
            const endpoint = stack.endpoint || "current";
            if (!agentOptions[endpoint]) {
                agentOptions[endpoint] = endpoint;
            }
        }
        stackFilter.agents.options = agentOptions;
    }

    // Attribute filter options
    stackFilter.attributes.options = { imageUpdatesAvailable: "imageUpdatesAvailable" };
}

function clearSearchText() {
    searchText.value = "";
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
        .filter(id => (stackList.value as any)[id]?.active)
        .forEach(id => getSocket().emit("pauseStack", id, () => {}));
    cancelSelectMode();
}

function resumeSelected() {
    Object.keys(selectedStacks.value)
        .filter(id => !(stackList.value as any)[id]?.active)
        .forEach(id => getSocket().emit("resumeStack", id, () => {}));
    cancelSelectMode();
}

function onScroll() {
    if (window.top!.scrollY <= 133) {
        windowTop.value = window.top!.scrollY;
    } else {
        windowTop.value = 133;
    }
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

// Auto-scroll: track whether the active item is visible in the scroll container
const isActiveVisible = ref(false);
let activeObserver: IntersectionObserver | null = null;

function scrollToActive() {
    const el = stackListRef.value?.querySelector(".item.active");
    el?.scrollIntoView({ block: "center", behavior: "smooth" });
}

function observeActive() {
    activeObserver?.disconnect();
    const container = stackListRef.value;
    const active = container?.querySelector(".item.active");
    if (!active || !container) { isActiveVisible.value = false; return; }
    // Synchronous initial check — the IntersectionObserver callback is async
    // and won't fire before the first list reorder
    const cr = container.getBoundingClientRect();
    const ar = active.getBoundingClientRect();
    isActiveVisible.value = ar.bottom > cr.top && ar.top < cr.bottom;
    activeObserver = new IntersectionObserver(([entry]) => {
        isActiveVisible.value = entry.isIntersecting;
    }, { root: container, threshold: 0.1 });
    activeObserver.observe(active as Element);
}

watch(flatStackList, () => {
    const wasVisible = isActiveVisible.value;
    nextTick(() => {
        if (wasVisible) scrollToActive();
        observeActive();
    });
});

defineExpose({ scrollToActive });

onMounted(() => {
    window.addEventListener("scroll", onScroll);
    nextTick(() => {
        const active = stackListRef.value?.querySelector(".item.active");
        active?.scrollIntoView({ block: "center" });
        observeActive();
    });
});

onBeforeUnmount(() => {
    window.removeEventListener("scroll", onScroll);
    activeObserver?.disconnect();
});
</script>

<style lang="scss" scoped>
@import "../styles/vars.scss";

.shadow-box {
    height: calc(100vh - 150px);
    position: sticky;
    top: 10px;
}

.small-padding {
    padding-left: 5px !important;
    padding-right: 5px !important;
}

.list-header {
    border-bottom: 1px solid #dee2e6;
    border-radius: 10px 10px 0 0;
    margin: -10px;
    margin-bottom: 10px;
    padding: 5px;

    .dark & {
        background-color: $dark-header-bg;
        border-bottom: 0;
    }
}

.header-top {
    display: flex;
    justify-content: space-between;
    align-items: center;
}

@media (max-width: 770px) {
    .list-header {
        margin: -20px;
        margin-bottom: 10px;
        padding: 5px;
    }
}

.search-icon {
    padding: 10px;
    color: #c0c0c0;

    // Clear filter button (X)
    svg[data-icon="times"] {
        cursor: pointer;
        transition: all ease-in-out 0.1s;

        &:hover {
            opacity: 0.5;
        }
    }
}

.search-input {
    max-width: 15em;
}

:deep(.filter-icon-container) {
    text-decoration: none;
    padding-right: 0px;
}

.filter-icon {
    padding: 10px;
    color: $dark-font-color3 !important;
    cursor: pointer;
    border: 1px solid transparent;
}

.filter-icon-active {
    color: $info !important;
    border: 1px solid $info;
    border-radius: 5px;
}

:deep(.filter-dropdown) {
    background-color: $dark-bg;
    border-color: $dark-font-color3;
    color: $dark-font-color;

    .dropdown-header {
        color: $dark-font-color;
        font-weight: bolder;
        padding-top: 0.25rem;
        padding-bottom: 0.25rem;
    }

    .dropdown-divider {
        margin: 0.25rem 0;
    }

    .form-check-input {
        border-color: $dark-font-color3;
    }
}

:deep(.filter-dropdown-clear) {
    color: $dark-font-color;

    &:disabled {
        color: $dark-font-color3;
    }

    &:hover {
        background-color: $dark-header-active-bg;
        color: $dark-font-color;
    }
}

:deep(.filter-dropdown form) {
    padding: 0.15rem 1rem !important;
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
