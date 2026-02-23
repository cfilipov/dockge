<template>
    <div class="container-fluid">
        <div class="row">
            <div v-if="!isMobile" class="col-12 col-md-4 col-xl-3">
                <!-- Container sidebar for containers/logs/shell routes -->
                <template v-if="showContainerSidebar">
                    <h1 class="mb-3">{{ $t("containersNav") }}</h1>
                    <ContainerList :scrollbar="true" />
                </template>
                <!-- Stack sidebar for all other routes (default) -->
                <template v-else>
                    <div>
                        <router-link to="/stacks/compose" class="btn btn-primary mb-3"><font-awesome-icon icon="plus" /> {{ $t("compose") }}</router-link>
                    </div>
                    <StackList :scrollbar="true" />
                </template>
            </div>

            <div ref="containerRef" class="col-12 col-md-8 col-xl-9 mb-3">
                <!-- Add :key to disable vue router re-use the same component -->
                <router-view :key="$route.fullPath" :calculatedHeight="height" />
            </div>
        </div>
    </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import { useRoute } from "vue-router";
import StackList from "../components/StackList.vue";
import ContainerList from "../components/ContainerList.vue";
import { useTheme } from "../composables/useTheme";

const { isMobile } = useTheme();
const route = useRoute();

const containerRef = ref<HTMLElement>();
const height = ref(0);

const showContainerSidebar = computed(() => {
    return route.path.startsWith("/containers") ||
           route.path.startsWith("/logs") ||
           route.path.startsWith("/shell");
});

onMounted(() => {
    if (containerRef.value) {
        height.value = containerRef.value.offsetHeight;
    }
});
</script>

<style lang="scss" scoped>
.container-fluid {
    width: 98%;
}

// The sidebar column must sit above the content column so that
// dropdowns (e.g. the filter menu) can overlay the content area.
// Without this, the later-in-DOM content column paints on top.
.col-12.col-md-4 {
    position: relative;
    z-index: 1;
}
</style>
