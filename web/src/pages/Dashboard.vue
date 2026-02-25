<template>
    <div class="container-fluid">
        <div class="row">
            <div v-if="!isMobile" class="col-12 col-md-4 col-xl-3">
                <!-- Container sidebar for containers/logs/shell routes -->
                <template v-if="showContainerSidebar">
                    <div class="d-flex align-items-center mb-3">
                        <h1 class="mb-0">{{ $t("containersNav") }}</h1>
                        <button class="btn btn-link ms-auto locate-btn" :title="$t('scrollToSelected')" @click="containerListRef?.scrollToActive()">
                            <font-awesome-icon icon="crosshairs" />
                        </button>
                    </div>
                    <ContainerList ref="containerListRef" :scrollbar="true" />
                </template>
                <!-- Network sidebar for /networks routes -->
                <template v-else-if="showNetworkSidebar">
                    <div class="d-flex align-items-center mb-3">
                        <h1 class="mb-0">{{ $t("networksNav") }}</h1>
                        <button class="btn btn-link ms-auto locate-btn" :title="$t('scrollToSelected')" @click="networkListRef?.scrollToActive()">
                            <font-awesome-icon icon="crosshairs" />
                        </button>
                    </div>
                    <NetworkList ref="networkListRef" :scrollbar="true" />
                </template>
                <!-- Image sidebar for /images routes -->
                <template v-else-if="showImageSidebar">
                    <div class="d-flex align-items-center mb-3">
                        <h1 class="mb-0">{{ $t("imagesNav") }}</h1>
                        <button class="btn btn-link ms-auto locate-btn" :title="$t('scrollToSelected')" @click="imageListRef?.scrollToActive()">
                            <font-awesome-icon icon="crosshairs" />
                        </button>
                    </div>
                    <ImageList ref="imageListRef" :scrollbar="true" />
                </template>
                <!-- Volume sidebar for /volumes routes -->
                <template v-else-if="showVolumeSidebar">
                    <div class="d-flex align-items-center mb-3">
                        <h1 class="mb-0">{{ $t("volumesNav") }}</h1>
                        <button class="btn btn-link ms-auto locate-btn" :title="$t('scrollToSelected')" @click="volumeListRef?.scrollToActive()">
                            <font-awesome-icon icon="crosshairs" />
                        </button>
                    </div>
                    <VolumeList ref="volumeListRef" :scrollbar="true" />
                </template>
                <!-- Stack sidebar for all other routes (default) -->
                <template v-else>
                    <div class="d-flex align-items-center mb-3">
                        <router-link to="/stacks/new" class="btn btn-primary"><font-awesome-icon icon="plus" /> {{ $t("compose") }}</router-link>
                        <button class="btn btn-link ms-auto locate-btn" :title="$t('scrollToSelected')" @click="stackListRef?.scrollToActive()">
                            <font-awesome-icon icon="crosshairs" />
                        </button>
                    </div>
                    <StackList ref="stackListRef" :scrollbar="true" />
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
import NetworkList from "../components/NetworkList.vue";
import ImageList from "../components/ImageList.vue";
import VolumeList from "../components/VolumeList.vue";
import { useTheme } from "../composables/useTheme";

const { isMobile } = useTheme();
const route = useRoute();

const containerRef = ref<HTMLElement>();
const height = ref(0);

const stackListRef = ref<InstanceType<typeof StackList>>();
const containerListRef = ref<InstanceType<typeof ContainerList>>();
const networkListRef = ref<InstanceType<typeof NetworkList>>();
const imageListRef = ref<InstanceType<typeof ImageList>>();
const volumeListRef = ref<InstanceType<typeof VolumeList>>();

const showContainerSidebar = computed(() => {
    return route.path.startsWith("/containers") ||
           route.path.startsWith("/logs") ||
           route.path.startsWith("/shell");
});

const showNetworkSidebar = computed(() => {
    return route.path.startsWith("/networks");
});

const showImageSidebar = computed(() => {
    return route.path.startsWith("/images");
});

const showVolumeSidebar = computed(() => {
    return route.path.startsWith("/volumes");
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
    height: 100%;
    overflow: hidden;
}

.row {
    height: 100%;
}

// The sidebar column must sit above the content column so that
// dropdowns (e.g. the filter menu) can overlay the content area.
// Without this, the later-in-DOM content column paints on top.
.col-12.col-md-4 {
    position: relative;
    z-index: 1;
    height: 100%;
    display: flex;
    flex-direction: column;
    overflow: hidden;
}

// Detail pane — the only scroll container on Dashboard pages
.col-12.col-md-8 {
    height: 100%;
    overflow-y: auto;
}

.locate-btn {
    color: #c0c0c0;
    padding: 0.25rem 0.5rem;
    &:hover {
        color: inherit;
    }
}
</style>
