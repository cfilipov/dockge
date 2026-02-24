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
                        <font-awesome-icon class="filter-icon" :class="{ 'filter-icon-active': imageFilter.isFilterSelected() }" icon="filter" />
                    </template>

                    <BDropdownItemButton :disabled="!imageFilter.isFilterSelected()" button-class="filter-dropdown-clear" @click="imageFilter.clear()">
                        <font-awesome-icon class="ms-1 me-2" icon="times" />{{ $t("clearFilter") }}
                    </BDropdownItemButton>

                    <BDropdownDivider></BDropdownDivider>

                    <template v-for="category in imageFilter.categories" :key="category.label">
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
import { ref, reactive, computed, onMounted, onBeforeUnmount, nextTick } from "vue";
import ImageListItem from "./ImageListItem.vue";
import { useSocket } from "../composables/useSocket";
import { StackFilterCategory } from "../../../common/util-common";

defineProps<{
    scrollbar?: boolean;
}>();

const { emitAgent } = useSocket();

const searchText = ref("");
const windowTop = ref(0);
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

function clearSearchText() {
    searchText.value = "";
}

function fetchImages() {
    emitAgent("", "getDockerImageList", (res: any) => {
        if (res.ok && res.dockerImageList) {
            imageList.value = res.dockerImageList;
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

onMounted(() => {
    fetchImages();
    window.addEventListener("scroll", onScroll);
    nextTick(() => {
        const active = listRef.value?.querySelector(".item.active");
        active?.scrollIntoView({ block: "center" });
    });
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
