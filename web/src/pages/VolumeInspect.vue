<template>
    <transition name="slide-fade" appear>
        <div v-if="volumeName">
            <h1 class="mb-3"><span v-if="badgeLabel" :class="badgeClass">{{ badgeLabel }}</span> {{ volumeName }}</h1>

            <div class="row">
                <div class="col-lg-8">
                    <!-- Containers Card -->
                    <CollapsibleSection>
                        <template #heading>{{ $t("volumeContainers") }} <span class="section-count">({{ volumeContainers.length }})</span></template>
                        <div v-if="volumeContainers.length > 0">
                            <ContainerCard v-for="c in volumeContainers" :key="c.containerId" :container="c" />
                        </div>
                        <div v-else-if="volumeDetail" class="shadow-box big-padding mb-3">
                            <p class="text-muted mb-0">{{ $t("noVolumeContainers") }}</p>
                        </div>
                        <div v-else class="shadow-box big-padding mb-3">
                            <p class="text-muted mb-0">{{ loading ? "Loading..." : "" }}</p>
                        </div>
                    </CollapsibleSection>
                </div>

                <div class="col-lg-4">
                    <!-- Overview Card -->
                    <OverviewCard :data="volumeDetail" :loading="loading">
                        <div class="overview-item">
                            <div class="overview-label">{{ $t("overviewName") }}</div>
                            <div class="overview-value">{{ volumeDetail.name }}</div>
                        </div>

                        <div class="overview-item">
                            <div class="overview-label">{{ $t("volumeDriver") }}</div>
                            <div class="overview-value">{{ volumeDetail.driver }}</div>
                        </div>

                        <div class="overview-item">
                            <div class="overview-label">{{ $t("volumeScope") }}</div>
                            <div class="overview-value">{{ volumeDetail.scope }}</div>
                        </div>

                        <div class="overview-item">
                            <div class="overview-label">{{ $t("volumeMountpoint") }}</div>
                            <div class="overview-value"><code>{{ volumeDetail.mountpoint }}</code></div>
                        </div>

                        <div v-if="volumeDetail.created" class="overview-item">
                            <div class="overview-label">{{ $t("volumeCreatedAt") }}</div>
                            <div class="overview-value">{{ formatDate(volumeDetail.created) }}</div>
                        </div>
                    </OverviewCard>
                </div>
            </div>
        </div>
        <div v-else>
            <h1 class="mb-3">{{ $t("volumesNav") }}</h1>
            <div class="shadow-box big-padding">
                <p class="text-muted mb-0">{{ $t("noVolumeSelected") }}</p>
            </div>
        </div>
    </transition>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from "vue";
import { useRoute } from "vue-router";
import { useI18n } from "vue-i18n";
import { useSocket } from "../composables/useSocket";
import { useContainerStore } from "../stores/containerStore";
import { formatDate } from "../common/util-common";
import ContainerCard from "../components/ContainerCard.vue";

const route = useRoute();
const { t } = useI18n();
const { emit } = useSocket();
const containerStore = useContainerStore();

const volumeDetail = ref<any>(null);
const loading = ref(false);

const volumeName = computed(() => route.params.volumeName as string || "");

// Get containers using this volume from the container store
const volumeContainers = computed(() => {
    if (!volumeName.value) return [];
    return containerStore.byVolume(volumeName.value);
});

const inUse = computed(() => volumeContainers.value.length > 0);
const badgeClass = computed(() => {
    if (!volumeDetail.value) return "";
    return `badge rounded-pill ${inUse.value ? "bg-success" : "bg-warning"}`;
});
const badgeLabel = computed(() => {
    if (!volumeDetail.value) return "";
    return inUse.value ? t("volumeInUse") : t("volumeUnused");
});

function fetchDetail() {
    if (!volumeName.value) {
        volumeDetail.value = null;
        return;
    }
    loading.value = true;
    emit("volumeInspect", volumeName.value, (res: any) => {
        loading.value = false;
        if (res.ok && res.volumeDetail) {
            volumeDetail.value = res.volumeDetail;
        }
    });
}

watch(volumeName, fetchDetail);

onMounted(() => {
    fetchDetail();
});
</script>

<style scoped lang="scss">
@import "../styles/info-chips";
</style>
