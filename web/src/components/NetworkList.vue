<template>
    <div class="shadow-box mb-3">
        <ListHeader v-model:search-text="searchText" :filter="networkFilter" show-category-headers />

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
import { ref, reactive, computed } from "vue";
import { useActiveScroll } from "../composables/useActiveScroll";
import ListHeader from "./ListHeader.vue";
import NetworkListItem from "./NetworkListItem.vue";
import { useNetworkStore } from "../stores/networkStore";
import { StackFilterCategory } from "../common/util-common";
import { useFilterParams } from "../composables/useFilterParams";

defineProps<{
    scrollbar?: boolean;
}>();

const networkStore = useNetworkStore();

const searchText = ref("");
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
useFilterParams(searchText, [
    { param: "driver", category: networkFilter.driver },
    { param: "status", category: networkFilter.status },
]);

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
    for (const net of networkStore.networksWithStatus) {
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
    let result = [...networkStore.networksWithStatus];

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
            if (networkFilter.status.selected.has("networkInUse") && n.inUse) return true;
            if (networkFilter.status.selected.has("networkUnused") && !n.inUse) return true;
            return false;
        });
    }

    // Sort alphabetically by name
    result.sort((a: any, b: any) => a.name.localeCompare(b.name));

    return result;
});

const { scrollToActive } = useActiveScroll(listRef, filteredNetworks);

defineExpose({ scrollToActive });
</script>

<style lang="scss" scoped>
@import "../styles/list-common";
</style>
