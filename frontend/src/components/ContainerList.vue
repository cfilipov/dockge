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
                        <font-awesome-icon class="filter-icon" :class="{ 'filter-icon-active': containerFilter.isFilterSelected() }" icon="filter" />
                    </template>

                    <BDropdownItemButton :disabled="!containerFilter.isFilterSelected()" button-class="filter-dropdown-clear" @click="containerFilter.clear()">
                        <font-awesome-icon class="ms-1 me-2" icon="times" />{{ $t("clearFilter") }}
                    </BDropdownItemButton>

                    <BDropdownDivider></BDropdownDivider>

                    <template v-for="category in containerFilter.categories" :key="category.label">
                        <BDropdownGroup v-if="category.hasOptions()" :header="$t(category.label)">
                            <BDropdownForm v-for="(value, key) in category.options" :key="key" form-class="filter-option" @change="category.toggleSelected(value)" @click.stop>
                                <BFormCheckbox :checked="category.selected.has(value)">{{ $t(key) }}</BFormCheckbox>
                            </BDropdownForm>
                        </BDropdownGroup>
                    </template>
                </BDropdown>
            </div>
        </div>

        <div class="stack-list" :class="{ scrollbar: scrollbar }" :style="listStyle">
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
import { ref, reactive, computed, onMounted, onBeforeUnmount } from "vue";
import ContainerListItem from "./ContainerListItem.vue";
import { useSocket } from "../composables/useSocket";
import { StackFilter, StackStatusInfo } from "../../../common/util-common";

defineProps<{
    scrollbar?: boolean;
}>();

const { containerList } = useSocket();

const searchText = ref("");
const windowTop = ref(0);
const containerFilter = reactive(new StackFilter());

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

/**
 * Map a container's state/health to the same StackStatusInfo label
 * used by the stack list filter (active, exited, unhealthy, down, partially).
 */
function getStatusLabel(c: any): string {
    if (c.state === "running" && c.health === "unhealthy") return "unhealthy";
    if (c.state === "running") return "active";
    if (c.state === "exited" || c.state === "dead") return "exited";
    if (c.state === "paused") return "active";
    if (c.state === "created") return "down";
    return "down";
}

function updateFilterOptions() {
    // Same status options as the stack list: from StackStatusInfo.ALL
    const statusOptions: Record<string, string> = {};
    for (const info of StackStatusInfo.ALL) {
        statusOptions[info.label] = info.label;
    }
    containerFilter.status.options = statusOptions;

    // Same attribute option as the stack list
    containerFilter.attributes.options = { imageUpdatesAvailable: "imageUpdatesAvailable" };
}

const filteredContainers = computed(() => {
    let result = [...(containerList.value || [])];

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
            const label = getStatusLabel(c);
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

function clearSearchText() {
    searchText.value = "";
}

function onScroll() {
    if (window.top!.scrollY <= 133) {
        windowTop.value = window.top!.scrollY;
    } else {
        windowTop.value = 133;
    }
}

onMounted(() => {
    window.addEventListener("scroll", onScroll);
});

onBeforeUnmount(() => {
    window.removeEventListener("scroll", onScroll);
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
