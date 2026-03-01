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
                            <div v-for="c in imageContainers" :key="c.containerId" class="shadow-box big-padding mb-3">
                                <h5 class="mb-3">
                                    <span class="badge rounded-pill me-2" :class="'bg-' + containerBadgeColor(c)">{{ $t(containerStatusLabel(c)) }}</span>
                                    <router-link :to="{ name: 'containerDetail', params: { containerName: c.name } }" class="stack-link">{{ c.name }}</router-link>
                                </h5>
                                <div class="network-props">
                                    <div class="network-chip">
                                        <span class="chip-label">{{ $t("containerID") }}</span>
                                        <code :title="c.containerId">{{ c.containerId.substring(0, 12) }}</code>
                                    </div>
                                </div>
                            </div>
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
                    <h4 class="mb-3">{{ $t("containerOverview") }}</h4>
                    <div v-if="imageDetail" class="shadow-box big-padding mb-3">
                        <div class="overview-list" role="region" :aria-label="$t('containerOverview')">
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
                        </div>
                    </div>
                    <div v-else class="shadow-box big-padding mb-3">
                        <p class="text-muted mb-0">{{ loading ? "Loading..." : "" }}</p>
                    </div>
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
import { ref, computed, watch, onMounted } from "vue";
import { useRoute } from "vue-router";
import { useI18n } from "vue-i18n";
import { useSocket } from "../composables/useSocket";
import { useContainerStore } from "../stores/containerStore";
import { ContainerStatusInfo } from "../common/util-common";

const route = useRoute();
const { t } = useI18n();
const { emit } = useSocket();
const containerStore = useContainerStore();

const imageDetail = ref<any>(null);
const loading = ref(false);

const imageRef = computed(() => route.params.imageRef as string || "");

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

function truncateCommand(cmd: string): string {
    if (!cmd) return "";
    if (cmd.length > 120) return cmd.substring(0, 120) + "...";
    return cmd;
}

function containerStatusLabel(c: Record<string, any>): string {
    return ContainerStatusInfo.from(c).label;
}

function containerBadgeColor(c: Record<string, any>): string {
    return ContainerStatusInfo.from(c).badgeColor;
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

watch(imageRef, fetchDetail);

onMounted(() => {
    fetchDetail();
});
</script>

<style scoped lang="scss">
@import "../styles/vars.scss";

.network-props {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
}

.network-chip {
    display: inline-flex;
    align-items: baseline;
    gap: 0.4rem;
    background: rgba(0, 0, 0, 0.06);
    border-radius: 10px;
    padding: 0.3rem 0.6rem;

    .dark & {
        background: $dark-header-bg;
    }

    .chip-label {
        font-size: 0.8em;
        font-weight: 600;
        color: $dark-font-color3;
        text-transform: uppercase;
        white-space: nowrap;
    }

    code {
        font-family: 'JetBrains Mono', monospace;
        font-size: 0.85em;
        color: $primary;
        background: none;
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
    color: $primary;

    code {
        font-family: 'JetBrains Mono', monospace;
        font-size: 0.85em;
        color: inherit;
        background: none;
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
