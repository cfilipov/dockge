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
                        <font-awesome-icon class="filter-icon" :class="{ 'filter-icon-active': networkFilter.isFilterSelected() }" icon="filter" />
                    </template>

                    <BDropdownItemButton :disabled="!networkFilter.isFilterSelected()" button-class="filter-dropdown-clear" @click="networkFilter.clear()">
                        <font-awesome-icon class="ms-1 me-2" icon="times" />{{ $t("clearFilter") }}
                    </BDropdownItemButton>

                    <BDropdownDivider></BDropdownDivider>

                    <template v-for="category in networkFilter.categories" :key="category.label">
                        <BDropdownGroup v-if="category.hasOptions()" :header="$t(category.label)">
                            <BDropdownForm v-for="(value, key) in category.options" :key="key" form-class="filter-option" @change="category.toggleSelected(value)" @click.stop>
                                <BFormCheckbox :checked="category.selected.has(value)">{{ $t(key) }}</BFormCheckbox>
                            </BDropdownForm>
                        </BDropdownGroup>
                    </template>
                </BDropdown>
            </div>
        </div>

        <div ref="listRef" class="stack-list" :class="{ scrollbar: scrollbar }" :style="listStyle">
            <div v-if="filteredNetworks.length === 0" class="text-center mt-3">
                {{ $t("noNetworks") }}
            </div>

            <NetworkListItem
                v-for="(item, i) in filteredNetworks"
                :key="i"
                :network="item"
            />
        </div>
    </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch, onMounted, onBeforeUnmount, nextTick } from "vue";
import NetworkListItem from "./NetworkListItem.vue";
import { useSocket } from "../composables/useSocket";
import { StackFilterCategory } from "../../../common/util-common";

defineProps<{
    scrollbar?: boolean;
}>();

const { emitAgent } = useSocket();

const searchText = ref("");
const windowTop = ref(0);
const networkList = ref<Record<string, any>[]>([]);
const listRef = ref<HTMLElement>();

// Two filter categories: Driver and Status (In Use / Unused)
class NetworkFilter {
    driver = new StackFilterCategory("networkDriver");
    status = new StackFilterCategory("status");

    get categories() {
        return [this.driver, this.status];
    }

    isFilterSelected(): boolean {
        return this.categories.some(c => c.isFilterSelected());
    }

    clear() {
        this.categories.forEach(c => c.clear());
    }
}

const networkFilter = reactive(new NetworkFilter());

const boxStyle = computed(() => {
    if (window.innerWidth > 550) {
        return { height: `calc(100vh - 160px + ${windowTop.value}px)` };
    } else {
        return { height: "calc(100vh - 160px)" };
    }
});

const listStyle = computed(() => {
    return { height: "calc(100% - 60px)" };
});

function driverDisplayName(driver: string): string {
    if (driver === "null") return "none";
    return driver;
}

function updateFilterOptions() {
    // Driver options — collect unique drivers from the data, display "null" as "none"
    const driverOptions: Record<string, string> = {};
    for (const net of networkList.value) {
        const d = net.driver || "unknown";
        const label = driverDisplayName(d);
        driverOptions[label] = d;
    }
    networkFilter.driver.options = driverOptions;

    // Status: In Use / Unused
    networkFilter.status.options = {
        networkInUse: "networkInUse",
        networkUnused: "networkUnused",
    };
}

const filteredNetworks = computed(() => {
    let result = [...networkList.value];

    updateFilterOptions();

    // Search filter
    if (searchText.value !== "") {
        const lowered = searchText.value.toLowerCase();
        result = result.filter((n: any) =>
            n.name.toLowerCase().includes(lowered) ||
            n.driver.toLowerCase().includes(lowered)
        );
    }

    // Driver filter
    if (networkFilter.driver.isFilterSelected()) {
        result = result.filter((n: any) => {
            return networkFilter.driver.selected.has(n.driver);
        });
    }

    // Status filter (In Use / Unused)
    if (networkFilter.status.isFilterSelected()) {
        result = result.filter((n: any) => {
            const inUse = (n.containers ?? 0) > 0;
            if (networkFilter.status.selected.has("networkInUse") && inUse) return true;
            if (networkFilter.status.selected.has("networkUnused") && !inUse) return true;
            return false;
        });
    }

    // Sort alphabetically by name
    result.sort((a: any, b: any) => a.name.localeCompare(b.name));

    return result;
});

function clearSearchText() {
    searchText.value = "";
}

function fetchNetworks() {
    emitAgent("", "getDockerNetworkList", (res: any) => {
        if (res.ok && res.dockerNetworkList) {
            networkList.value = res.dockerNetworkList;
        }
    });
}

function onScroll() {
    if (window.top!.scrollY <= 133) {
        windowTop.value = window.top!.scrollY;
    } else {
        windowTop.value = 133;
    }
}

// Auto-scroll: track whether the active item is visible in the scroll container
const isActiveVisible = ref(false);
let activeObserver: IntersectionObserver | null = null;

function scrollToActive() {
    const el = listRef.value?.querySelector(".item.active");
    el?.scrollIntoView({ block: "center", behavior: "smooth" });
}

function observeActive() {
    activeObserver?.disconnect();
    const container = listRef.value;
    const active = container?.querySelector(".item.active");
    if (!active || !container) { isActiveVisible.value = false; return; }
    const cr = container.getBoundingClientRect();
    const ar = active.getBoundingClientRect();
    isActiveVisible.value = ar.bottom > cr.top && ar.top < cr.bottom;
    activeObserver = new IntersectionObserver(([entry]) => {
        isActiveVisible.value = entry.isIntersecting;
    }, { root: container, threshold: 0.1 });
    activeObserver.observe(active as Element);
}

watch(filteredNetworks, () => {
    const wasVisible = isActiveVisible.value;
    nextTick(() => {
        if (wasVisible) scrollToActive();
        observeActive();
    });
});

defineExpose({ scrollToActive });

onMounted(() => {
    fetchNetworks();
    window.addEventListener("scroll", onScroll);
    nextTick(() => {
        const active = listRef.value?.querySelector(".item.active");
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
</style>
