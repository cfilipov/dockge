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

        <div ref="stackList" class="stack-list" :class="{ scrollbar: scrollbar }" :style="stackListStyle">
            <div v-if="flatStackList.length === 0" class="text-center mt-3">
                <router-link to="/compose">{{ $t("addFirstStackMsg") }}</router-link>
            </div>

            <div class="stack-list-inner" v-for="(agent, index) in agentStackList" :key="index">
                <div v-if="$root.agentCount > 1"
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
                    v-show="$root.agentCount === 1 || !closedAgents.get(agent.endpoint)"
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

    <Confirm ref="confirmPause" :yes-text="$t('Yes')" :no-text="$t('No')" @yes="pauseSelected">
        {{ $t("pauseStackMsg") }}
    </Confirm>
</template>

<script>
import Confirm from "../components/Confirm.vue";
import StackListItem from "../components/StackListItem.vue";
import { CREATED_FILE, CREATED_STACK, EXITED, RUNNING, RUNNING_AND_EXITED, UNHEALTHY, UNKNOWN, StackFilter, StackStatusInfo } from "../../../common/util-common";

export default {
    components: { Confirm, StackListItem },
    props: {
        scrollbar: { type: Boolean },
    },
    data() {
        return {
            searchText: "",
            selectMode: false,
            selectAll: false,
            disableSelectAllWatcher: false,
            selectedStacks: {},
            windowTop: 0,
            stackFilter: new StackFilter(),
            closedAgents: new Map(),
        };
    },
    computed: {
        boxStyle() {
            if (window.innerWidth > 550) {
                return { height: `calc(100vh - 160px + ${this.windowTop}px)` };
            } else {
                return { height: "calc(100vh - 160px)" };
            }
        },
        /** Grouped stacks (PR #800 behavior), with filters + sort applied */
        agentStackList() {
            let result = Object.values(this.$root.completeStackList);

            // Populate filter options from current data
            this.updateFilterOptions(result);

            // filter
            result = result.filter(stack => {
                // search text
                let searchTextMatch = true;
                if (this.searchText !== "") {
                    const lowered = this.searchText.toLowerCase();
                    searchTextMatch =
                        stack.name.toLowerCase().includes(lowered) ||
                        (stack.tags && stack.tags.find(tag =>
                            tag.name.toLowerCase().includes(lowered) ||
                            tag.value?.toLowerCase().includes(lowered)
                        ));
                }

                // status filter
                let statusMatch = true;
                if (this.stackFilter.status.isFilterSelected()) {
                    const statusLabel = StackStatusInfo.get(stack.status).label;
                    statusMatch = this.stackFilter.status.selected.has(statusLabel);
                }

                // agent filter
                let agentMatch = true;
                if (this.stackFilter.agents.isFilterSelected()) {
                    const endpoint = stack.endpoint || "current";
                    agentMatch = this.stackFilter.agents.selected.has(endpoint);
                }

                // attribute filter
                let attributeMatch = true;
                if (this.stackFilter.attributes.isFilterSelected()) {
                    attributeMatch = false;
                    for (const attribute of this.stackFilter.attributes.selected) {
                        if (stack[attribute] === true) {
                            attributeMatch = true;
                        }
                    }
                }

                return searchTextMatch && statusMatch && agentMatch && attributeMatch;
            });

            // sort
            result.sort((m1, m2) => {
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
                ...result.reduce((acc, stack) => {
                    const endpoint = stack.endpoint || 'current';
                    if (!acc.has(endpoint)) acc.set(endpoint, []);
                    acc.get(endpoint).push(stack);
                    return acc;
                }, new Map()).entries()
            ].map(([endpoint, stacks]) => ({ endpoint, stacks }));

            groups.sort((a, b) => {
                if (a.endpoint === 'current' && b.endpoint !== 'current') return -1;
                if (a.endpoint !== 'current' && b.endpoint === 'current') return 1;
                return a.endpoint.localeCompare(b.endpoint);
            });

            return groups;
        },
        /** flat list for convenience (button states, updateAll, selection watchers) */
        flatStackList() {
            return this.agentStackList.flatMap(g => g.stacks);
        },
        isDarkTheme() {
            return document.body.classList.contains("dark");
        },
        stackListStyle() {
            let listHeaderHeight = 60;
            if (this.selectMode) listHeaderHeight += 42;
            return { height: `calc(100% - ${listHeaderHeight}px)` };
        },
        selectedStackCount() {
            return Object.keys(this.selectedStacks).length;
        },
        filtersActive() {
            return this.stackFilter.isFilterSelected() || this.searchText !== "";
        }
    },
    watch: {
        searchText() {
            for (let stack of this.flatStackList) {
                if (!this.selectedStacks[stack.id]) {
                    if (this.selectAll) {
                        this.disableSelectAllWatcher = true;
                        this.selectAll = false;
                    }
                    break;
                }
            }
        },
        selectAll() {
            if (!this.disableSelectAllWatcher) {
                this.selectedStacks = {};
                if (this.selectAll) {
                    this.flatStackList.forEach((item) => {
                        this.selectedStacks[item.id] = true;
                    });
                }
            } else {
                this.disableSelectAllWatcher = false;
            }
        },
        selectMode() {
            if (!this.selectMode) {
                this.selectAll = false;
                this.selectedStacks = {};
            }
        },
    },
    mounted() {
        window.addEventListener("scroll", this.onScroll);
    },
    beforeUnmount() {
        window.removeEventListener("scroll", this.onScroll);
    },
    methods: {
        onScroll() {
            if (window.top.scrollY <= 133) {
                this.windowTop = window.top.scrollY;
            } else {
                this.windowTop = 133;
            }
        },
        clearSearchText() {
            this.searchText = "";
        },
        updateFilterOptions(stacks) {
            // Build status options from StackStatusInfo
            const statusOptions = {};
            for (const info of StackStatusInfo.ALL) {
                statusOptions[info.label] = info.label;
            }
            this.stackFilter.status.options = statusOptions;

            // Build agent options from current stacks
            if (this.$root.agentCount > 1) {
                const agentOptions = {};
                for (const stack of stacks) {
                    const endpoint = stack.endpoint || "current";
                    if (!agentOptions[endpoint]) {
                        agentOptions[endpoint] = endpoint;
                    }
                }
                this.stackFilter.agents.options = agentOptions;
            }

            // Attribute filter options
            this.stackFilter.attributes.options = { imageUpdatesAvailable: "imageUpdatesAvailable" };
        },
        deselect(id) {
            delete this.selectedStacks[id];
        },
        select(id) {
            this.selectedStacks[id] = true;
        },
        isSelected(id) {
            return id in this.selectedStacks;
        },
        cancelSelectMode() {
            this.selectMode = false;
            this.selectedStacks = {};
        },
        pauseDialog() {
            this.$refs.confirmPause.show();
        },
        pauseSelected() {
            Object.keys(this.selectedStacks)
                .filter(id => this.$root.stackList[id].active)
                .forEach(id => this.$root.getSocket().emit("pauseStack", id, () => {}));
            this.cancelSelectMode();
        },
        resumeSelected() {
            Object.keys(this.selectedStacks)
                .filter(id => !this.$root.stackList[id].active)
                .forEach(id => this.$root.getSocket().emit("resumeStack", id, () => {}));
            this.cancelSelectMode();
        },
    },
};
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
