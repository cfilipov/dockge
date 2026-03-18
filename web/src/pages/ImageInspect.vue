<template>
    <transition name="slide-fade" appear>
        <div v-if="imageRef">
            <h1 class="mb-3"><span v-if="badgeLabel" :class="badgeClass">{{ badgeLabel }}</span> {{ displayName }}</h1>

            <div class="row">
                <div class="col-lg-8">
                    <!-- Containers Card -->
                    <CollapsibleSection>
                        <template #heading>{{ $t("imageContainers") }} <span class="section-count">({{ imageContainers.length }})</span></template>
                        <div v-if="imageContainers.length > 0">
                            <ContainerCard v-for="c in imageContainers" :key="c.containerId" :container="c" />
                        </div>
                        <div v-else-if="imageDetail" class="shadow-box big-padding mb-3">
                            <p class="text-muted mb-0">{{ $t("noImageContainers") }}</p>
                        </div>
                        <div v-else class="shadow-box big-padding mb-3">
                            <p class="text-muted mb-0">{{ loading ? "Loading..." : "" }}</p>
                        </div>
                    </CollapsibleSection>

                    <!-- Layers Card -->
                    <CollapsibleSection>
                        <template #heading>{{ $t("imageLayers") }} <span class="section-count">({{ imageDetail?.layers?.length ?? 0 }})</span></template>
                        <div v-if="imageDetail && imageDetail.layers && imageDetail.layers.length > 0" class="shadow-box big-padding mb-3">
                            <div class="table-responsive">
                                <table class="table table-sm mb-0 layer-table">
                                    <thead>
                                        <tr>
                                            <th>ID</th>
                                            <th>{{ $t("imageSize") }}</th>
                                            <th>{{ $t("processCommand") }}</th>
                                        </tr>
                                    </thead>
                                    <tbody>
                                        <tr v-for="(layer, idx) in imageDetail.layers" :key="idx">
                                            <td><code>{{ layer.id }}</code></td>
                                            <td>{{ layer.size }}</td>
                                            <td class="command-cell"><code>{{ truncateCommand(layer.command) }}</code></td>
                                        </tr>
                                    </tbody>
                                </table>
                            </div>
                        </div>
                        <div v-else-if="imageDetail" class="shadow-box big-padding mb-3">
                            <p class="text-muted mb-0">No layers.</p>
                        </div>
                        <div v-else class="shadow-box big-padding mb-3">
                            <p class="text-muted mb-0">{{ loading ? "Loading..." : "" }}</p>
                        </div>
                    </CollapsibleSection>
                </div>

                <div class="col-lg-4">
                    <!-- Overview Card -->
                    <OverviewCard :data="imageDetail" :loading="loading">
                        <div class="overview-item">
                            <div class="overview-label">{{ $t("overviewName") }}</div>
                            <div class="overview-value">{{ displayName }}</div>
                        </div>

                        <div class="overview-item">
                            <div class="overview-label">{{ $t("imageID") }}</div>
                            <div class="overview-value">
                                <code :title="imageDetail.id">{{ imageDetail.id.substring(0, 19) }}</code>
                            </div>
                        </div>

                        <div class="overview-item">
                            <div class="overview-label">{{ $t("imageSize") }}</div>
                            <div class="overview-value">{{ imageDetail.size }}</div>
                        </div>

                        <div v-if="imageDetail.created" class="overview-item">
                            <div class="overview-label">{{ $t("imageCreatedAt") }}</div>
                            <div class="overview-value">{{ formatDate(imageDetail.created) }}</div>
                        </div>

                        <div class="overview-item">
                            <div class="overview-label">{{ $t("imageArchitecture") }}</div>
                            <div class="overview-value">{{ imageDetail.architecture }}</div>
                        </div>

                        <div class="overview-item">
                            <div class="overview-label">{{ $t("imageOS") }}</div>
                            <div class="overview-value">{{ imageDetail.os }}</div>
                        </div>

                        <div v-if="imageDetail.workingDir" class="overview-item">
                            <div class="overview-label">{{ $t("imageWorkingDir") }}</div>
                            <div class="overview-value"><code>{{ imageDetail.workingDir }}</code></div>
                        </div>
                    </OverviewCard>
                </div>
            </div>
        </div>
        <div v-else>
            <h1 class="mb-3">{{ $t("imagesNav") }}</h1>
            <div class="shadow-box big-padding">
                <p class="text-muted mb-0">{{ $t("noImageSelected") }}</p>
            </div>
        </div>
    </transition>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from "vue";
import { useRoute } from "vue-router";
import { useI18n } from "vue-i18n";
import { useSocket } from "../composables/useSocket";
import { useContainerStore } from "../stores/containerStore";
import { useImageStore } from "../stores/imageStore";
import { formatDate } from "../common/util-common";
import ContainerCard from "../components/ContainerCard.vue";

const route = useRoute();
const { t } = useI18n();
const { emit } = useSocket();
const containerStore = useContainerStore();
const imageStoreInstance = useImageStore();

const imageDetail = ref<any>(null);
const loading = ref(false);

const imageRef = computed(() => route.params.imageRef as string || "");

// Find this image in the store by matching imageRef against ID or repoTags
const storeImage = computed(() => {
    if (!imageRef.value) return undefined;
    for (const img of imageStoreInstance.imageMap.values()) {
        if (img.id === imageRef.value) return img;
        if (img.repoTags?.some(t => t === imageRef.value)) return img;
    }
    return undefined;
});

const isDangling = computed(() => {
    if (!imageDetail.value) return false;
    const tags = imageDetail.value.repoTags || [];
    return tags.length === 0;
});

const displayName = computed(() => {
    if (imageDetail.value) {
        const tags = imageDetail.value.repoTags || [];
        if (tags.length > 0) return tags[0];
        const id = imageDetail.value.id || "";
        if (id.startsWith("sha256:")) return id.substring(0, 19);
        return id.substring(0, 12) || imageRef.value;
    }
    return imageRef.value;
});

// Get containers using this image from the container store
const imageContainers = computed(() => {
    if (!imageDetail.value?.id) return [];
    return containerStore.byImage(imageDetail.value.id);
});

const inUse = computed(() => imageContainers.value.length > 0);
const badgeClass = computed(() => {
    if (!imageDetail.value) return "";
    if (isDangling.value) return "badge rounded-pill bg-secondary";
    return `badge rounded-pill ${inUse.value ? "bg-success" : "bg-warning"}`;
});
const badgeLabel = computed(() => {
    if (!imageDetail.value) return "";
    if (isDangling.value) return t("imageDangling");
    return inUse.value ? t("imageInUse") : t("imageUnused");
});

function truncateCommand(cmd: string): string {
    if (!cmd) return "";
    if (cmd.length > 120) return cmd.substring(0, 120) + "...";
    return cmd;
}

function fetchDetail() {
    if (!imageRef.value) {
        imageDetail.value = null;
        return;
    }
    loading.value = true;
    emit("imageInspect", imageRef.value, (res: any) => {
        loading.value = false;
        if (res.ok && res.imageDetail) {
            imageDetail.value = res.imageDetail;
        }
    });
}

// Debounced re-fetch on relevant events
let refetchTimeout: ReturnType<typeof setTimeout> | null = null;

watch(() => imageStoreInstance.lastEvent, (evt) => {
    if (!evt) return;
    // Images are keyed by ID, but events have name — match via storeImage
    if (storeImage.value && evt.id === storeImage.value.id) {
        if (refetchTimeout) clearTimeout(refetchTimeout);
        refetchTimeout = setTimeout(fetchDetail, 500);
    }
});

watch(imageRef, fetchDetail);

onMounted(() => {
    fetchDetail();
});

onUnmounted(() => {
    if (refetchTimeout) clearTimeout(refetchTimeout);
});
</script>

<style scoped lang="scss">
@import "../styles/info-chips";

.layer-table {
    font-size: 0.9em;

    th, td {
        padding: 0.55rem 0.75rem;
        border: none;
    }

    thead th {
        font-size: 0.8em;
        font-weight: 600;
        text-transform: uppercase;
        letter-spacing: 0.03em;
        color: $dark-font-color3;
        padding-bottom: 0.4rem;
        border-bottom: 1px solid rgba(0, 0, 0, 0.1);

        .dark & {
            border-bottom-color: $dark-border-color;
        }
    }

    tbody td {
        border-bottom: 1px solid rgba(0, 0, 0, 0.05);

        .dark & {
            color: $dark-font-color;
            border-bottom-color: $dark-border-color;
        }
    }

    tbody tr:last-child td {
        border-bottom: none;
    }

    code {
        font-family: 'JetBrains Mono', monospace;
        font-size: 0.9em;
        color: inherit;
        background: none;
    }

    .dark & {
        --bs-table-bg: transparent;
    }
}

.command-cell {
    max-width: 400px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}
</style>
