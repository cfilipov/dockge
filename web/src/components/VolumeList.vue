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
                        <font-awesome-icon class="filter-icon" :class="{ 'filter-icon-active': volumeFilter.isFilterSelected() }" icon="filter" />
                    </template>

                    <BDropdownItemButton :disabled="!volumeFilter.isFilterSelected()" button-class="filter-dropdown-clear" @click="volumeFilter.clear()">
                        <font-awesome-icon class="ms-1 me-2" icon="times" />{{ $t("clearFilter") }}
                    </BDropdownItemButton>

                    <BDropdownDivider></BDropdownDivider>

                    <template v-for="category in volumeFilter.categories" :key="category.label">
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
import { ref, reactive, computed, watch, onMounted, onBeforeUnmount, nextTick } from "vue";
import VolumeListItem from "./VolumeListItem.vue";
import { useSocket } from "../composables/useSocket";
import { StackFilterCategory } from "../common/util-common";

defineProps<{
    scrollbar?: boolean;
}>();

const { emitAgent } = useSocket();

const searchText = ref("");
const windowTop = ref(0);
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

function clearSearchText() {
    searchText.value = "";
}

function fetchVolumes() {
    emitAgent("", "getDockerVolumeList", (res: any) => {
        if (res.ok && res.dockerVolumeList) {
            volumeList.value = res.dockerVolumeList;
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
    const container = listRef.value;
    const el = container?.querySelector(".item.active") as HTMLElement | null;
    if (!el || !container) return;
    container.scrollTo({
        top: el.offsetTop - container.clientHeight / 2 + el.clientHeight / 2,
        behavior: "smooth",
    });
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

watch(filteredVolumes, () => {
    const wasVisible = isActiveVisible.value;
    nextTick(() => {
        if (wasVisible) scrollToActive();
        observeActive();
    });
});

defineExpose({ scrollToActive });

onMounted(() => {
    fetchVolumes();
    window.addEventListener("scroll", onScroll);
    nextTick(() => {
        const container = listRef.value;
        const active = container?.querySelector(".item.active") as HTMLElement | null;
        if (active && container) {
            container.scrollTop = active.offsetTop - container.clientHeight / 2 + active.clientHeight / 2;
        }
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
