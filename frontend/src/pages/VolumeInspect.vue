<template>
    <transition name="slide-fade" appear>
        <div v-if="volumeName">
            <h1 class="mb-3"><span v-if="badgeLabel" :class="badgeClass">{{ badgeLabel }}</span> {{ volumeName }}</h1>

            <div class="row">
                <div class="col-lg-8">
                    <!-- Containers Card -->
                    <h4 class="mb-3">{{ $t("volumeContainers") }} ({{ volumeDetail?.containers?.length ?? 0 }})</h4>
                    <div v-if="volumeDetail && volumeDetail.containers && volumeDetail.containers.length > 0">
                        <div v-for="c in volumeDetail.containers" :key="c.containerId" class="shadow-box big-padding mb-3">
                            <h5 class="mb-3">
                                <router-link :to="{ name: 'containerDetail', params: { containerName: c.name } }" class="stack-link"><font-awesome-icon icon="cubes" class="me-2" />{{ c.name }}</router-link>
                            </h5>
                            <div class="inspect-grid">
                                <div class="inspect-label">{{ $t("containerID") }}</div>
                                <div class="inspect-value"><code :title="c.containerId">{{ c.containerId.substring(0, 12) }}</code></div>

                                <div class="inspect-label">{{ $t("status") }}</div>
                                <div class="inspect-value">
                                    <span class="badge rounded-pill" :class="'bg-' + containerBadgeColor(c)">{{ $t(containerStatusLabel(c)) }}</span>
                                </div>
                            </div>
                        </div>
                    </div>
                    <div v-else-if="volumeDetail" class="shadow-box big-padding mb-3">
                        <p class="text-muted mb-0">{{ $t("noVolumeContainers") }}</p>
                    </div>
                    <div v-else class="shadow-box big-padding mb-3">
                        <p class="text-muted mb-0">{{ loading ? "Loading..." : "" }}</p>
                    </div>
                </div>

                <div class="col-lg-4">
                    <!-- Overview Card -->
                    <h4 class="mb-3">{{ $t("containerOverview") }}</h4>
                    <div v-if="volumeDetail" class="shadow-box big-padding mb-3">
                        <div class="overview-list">
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
                        </div>
                    </div>
                    <div v-else class="shadow-box big-padding mb-3">
                        <p class="text-muted mb-0">{{ loading ? "Loading..." : "" }}</p>
                    </div>
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
import { ContainerStatusInfo } from "../../../common/util-common";

const route = useRoute();
const { t } = useI18n();
const { emitAgent } = useSocket();

const volumeDetail = ref<any>(null);
const loading = ref(false);

const volumeName = computed(() => route.params.volumeName as string || "");

const inUse = computed(() => {
    if (!volumeDetail.value) return false;
    return (volumeDetail.value.containers?.length ?? 0) > 0;
});
const badgeClass = computed(() => {
    if (!volumeDetail.value) return "";
    return `badge rounded-pill ${inUse.value ? "bg-success" : "bg-warning"}`;
});
const badgeLabel = computed(() => {
    if (!volumeDetail.value) return "";
    return inUse.value ? t("volumeInUse") : t("volumeUnused");
});

function formatDate(dateStr: string): string {
    if (!dateStr) return "";
    const d = new Date(dateStr);
    if (isNaN(d.getTime())) return dateStr;
    return d.toLocaleString(undefined, {
        year: "numeric",
        month: "short",
        day: "numeric",
        hour: "numeric",
        minute: "2-digit",
    });
}

function containerStatusLabel(c: Record<string, any>): string {
    return ContainerStatusInfo.from(c).label;
}

function containerBadgeColor(c: Record<string, any>): string {
    return ContainerStatusInfo.from(c).badgeColor;
}

function fetchDetail() {
    if (!volumeName.value) {
        volumeDetail.value = null;
        return;
    }
    loading.value = true;
    emitAgent("", "volumeInspect", volumeName.value, (res: any) => {
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
@import "../styles/vars.scss";

.inspect-grid {
    display: grid;
    grid-template-columns: auto 1fr;
    gap: 0.6rem 1.5rem;
    align-items: baseline;
}

.inspect-label {
    font-weight: 600;
    white-space: nowrap;
    color: $dark-font-color3;

    .dark & {
        color: $dark-font-color3;
    }
}

.inspect-value {
    word-break: break-all;

    code {
        font-family: 'JetBrains Mono', monospace;
        font-size: 0.9em;
    }
}

.overview-list {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
}

.overview-item {
    display: flex;
    flex-direction: column;
}

.overview-label {
    font-size: 0.8em;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.03em;
    color: $dark-font-color3;
    margin-bottom: 0.15rem;

    .dark & {
        color: $dark-font-color3;
    }
}

.overview-value {
    word-break: break-all;

    code {
        font-family: 'JetBrains Mono', monospace;
        font-size: 0.9em;
    }
}

.stack-link {
    font-weight: 600;
    text-decoration: none;
    color: $primary;

    &:hover {
        color: lighten($primary, 10%);
    }
}
</style>
