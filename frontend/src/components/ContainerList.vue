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
            </div>
        </div>

        <div class="stack-list" :class="{ scrollbar: scrollbar }" :style="listStyle">
            <div v-if="sortedContainers.length === 0" class="text-center mt-3">
                {{ $t("noContainers") }}
            </div>

            <ContainerListItem
                v-for="(item, i) in sortedContainers"
                :key="i"
                :container="item"
            />
        </div>
    </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from "vue";
import ContainerListItem from "./ContainerListItem.vue";
import { useSocket } from "../composables/useSocket";

defineProps<{
    scrollbar?: boolean;
}>();

const { containerList } = useSocket();

const searchText = ref("");
const windowTop = ref(0);

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

const sortedContainers = computed(() => {
    let result = [...(containerList.value || [])];

    // Filter by search text
    if (searchText.value !== "") {
        const lowered = searchText.value.toLowerCase();
        result = result.filter((c: any) =>
            c.name.toLowerCase().includes(lowered) ||
            c.serviceName.toLowerCase().includes(lowered) ||
            c.stackName.toLowerCase().includes(lowered)
        );
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
</style>
