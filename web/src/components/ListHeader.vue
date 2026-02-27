<template>
    <div class="list-header">
        <div class="header-top">
            <div class="d-flex flex-grow-1 align-items-center">
                <a v-if="searchText == ''" class="search-icon">
                    <font-awesome-icon icon="search" />
                </a>
                <a v-if="searchText != ''" class="search-icon" style="cursor: pointer" @click="emit('update:searchText', '')">
                    <font-awesome-icon icon="times" />
                </a>
                <input
                    :value="searchText"
                    class="form-control search-input"
                    :placeholder="$t('Search')"
                    autocomplete="off"
                    @input="onSearchInput"
                />
            </div>

            <BDropdown variant="link" placement="bottom-end" menu-class="filter-dropdown" toggle-class="filter-icon-container" no-caret>
                <template #button-content>
                    <font-awesome-icon class="filter-icon" :class="{ 'filter-icon-active': filter.isFilterSelected() }" icon="filter" />
                    <span class="visually-hidden">{{ $t("filter") }}</span>
                </template>

                <BDropdownItemButton :disabled="!filter.isFilterSelected()" button-class="filter-dropdown-clear" @click="filter.clear()">
                    <font-awesome-icon class="ms-1 me-2" icon="times" />{{ $t("clearFilter") }}
                </BDropdownItemButton>

                <BDropdownDivider></BDropdownDivider>

                <!-- With category headers (BDropdownGroup) -->
                <template v-if="showCategoryHeaders">
                    <template v-for="category in filter.categories" :key="category.label">
                        <BDropdownGroup v-if="category.hasOptions()" :header="$t(category.label)">
                            <BDropdownForm v-for="(value, key) in category.options" :key="key" form-class="filter-option" @change="category.toggleSelected(value)" @click.stop>
                                <BFormCheckbox :checked="category.selected.has(value)">{{ $t(key) }}</BFormCheckbox>
                            </BDropdownForm>
                        </BDropdownGroup>
                    </template>
                </template>

                <!-- Without category headers (flat checkboxes) -->
                <template v-else>
                    <template v-for="category in filter.categories" :key="category.label">
                        <template v-if="category.hasOptions()">
                            <BDropdownForm v-for="(value, key) in category.options" :key="key" form-class="filter-option" @change="category.toggleSelected(value)" @click.stop>
                                <BFormCheckbox :checked="category.selected.has(value)">{{ $t(key) }}</BFormCheckbox>
                            </BDropdownForm>
                        </template>
                    </template>
                </template>
            </BDropdown>
        </div>
    </div>
</template>

<script setup lang="ts">
interface FilterCategory {
    label: string;
    hasOptions(): boolean;
    options: Record<string, string>;
    selected: Set<string>;
    toggleSelected(value: string): void;
}

interface FilterLike {
    isFilterSelected(): boolean;
    clear(): void;
    categories: FilterCategory[];
}

withDefaults(defineProps<{
    searchText: string;
    filter: FilterLike;
    showCategoryHeaders?: boolean;
}>(), {
    showCategoryHeaders: false,
});

const emit = defineEmits<{
    "update:searchText": [value: string];
}>();

function onSearchInput(event: Event) {
    emit("update:searchText", (event.target as HTMLInputElement).value);
}
</script>

<style lang="scss" scoped>
@import "../styles/vars.scss";

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
    .dropdown-header {
        font-weight: bolder;
        padding-top: 0.25rem;
        padding-bottom: 0.25rem;
    }

    .dropdown-divider {
        margin: 0.25rem 0;
    }
}

:deep(.filter-dropdown form) {
    padding: 0.15rem 1rem !important;
}
</style>

<style lang="scss">
@import "../styles/vars.scss";

.dark .filter-dropdown {
    background-color: $dark-bg;
    border-color: $dark-font-color3;
    color: $dark-font-color;

    .dropdown-header {
        color: $dark-font-color;
    }

    .form-check-input {
        border-color: $dark-font-color3;
    }
}

.dark .filter-dropdown-clear {
    color: $dark-font-color;

    &:disabled {
        color: $dark-font-color3;
    }

    &:hover {
        background-color: $dark-header-active-bg;
        color: $dark-font-color;
    }
}
</style>
