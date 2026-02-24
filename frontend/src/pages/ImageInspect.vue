<template>
    <transition name="slide-fade" appear>
        <div v-if="imageRef">
            <h1 class="mb-3"><span v-if="badgeLabel" :class="badgeClass">{{ badgeLabel }}</span> {{ imageRef }}</h1>

            <div class="row">
                <div class="col-lg-8">
                    <!-- Containers Card -->
                    <h4 class="mb-3">{{ $t("imageContainers") }} ({{ imageDetail?.containers?.length ?? 0 }})</h4>
                    <div v-if="imageDetail && imageDetail.containers && imageDetail.containers.length > 0">
                        <div v-for="c in imageDetail.containers" :key="c.containerId" class="shadow-box big-padding mb-3">
                            <h5 class="mb-3">
                                <router-link :to="{ name: 'containerDetail', params: { containerName: c.name } }" class="stack-link">{{ c.name }}</router-link>
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
                    <div v-else-if="imageDetail" class="shadow-box big-padding mb-3">
                        <p class="text-muted mb-0">{{ $t("noImageContainers") }}</p>
                    </div>
                    <div v-else class="shadow-box big-padding mb-3">
                        <p class="text-muted mb-0">{{ loading ? "Loading..." : "" }}</p>
                    </div>

                    <!-- Layers Card -->
                    <h4 class="mb-3">{{ $t("imageLayers") }} ({{ imageDetail?.layers?.length ?? 0 }})</h4>
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
                </div>

                <div class="col-lg-4">
                    <!-- Overview Card -->
                    <h4 class="mb-3">{{ $t("containerOverview") }}</h4>
                    <div v-if="imageDetail" class="shadow-box big-padding mb-3">
                        <div class="overview-list">
                            <div class="overview-item">
                                <div class="overview-label">{{ $t("name") }}</div>
                                <div class="overview-value">{{ imageDetail.repoTags?.[0] || imageRef }}</div>
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
import { StackStatusInfo } from "../../../common/util-common";

const route = useRoute();
const { t } = useI18n();
const { emitAgent } = useSocket();

const imageDetail = ref<any>(null);
const loading = ref(false);

const imageRef = computed(() => route.params.imageRef as string || "");

const inUse = computed(() => {
    if (!imageDetail.value) return false;
    return (imageDetail.value.containers?.length ?? 0) > 0;
});
const badgeClass = computed(() =>
    imageDetail.value ? `badge rounded-pill ${inUse.value ? "bg-success" : "bg-warning"}` : ""
);
const badgeLabel = computed(() =>
    imageDetail.value ? (inUse.value ? t("imageInUse") : t("imageUnused")) : ""
);

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
    if (c.state === "running" && c.health === "unhealthy") return "unhealthy";
    if (c.state === "running") return "active";
    if (c.state === "exited" || c.state === "dead") return "exited";
    if (c.state === "paused") return "active";
    if (c.state === "created") return "down";
    return "down";
}

function containerBadgeColor(c: Record<string, any>): string {
    const label = containerStatusLabel(c);
    const info = StackStatusInfo.ALL.find(i => i.label === label);
    return info ? info.badgeColor : "secondary";
}

function fetchDetail() {
    if (!imageRef.value) {
        imageDetail.value = null;
        return;
    }
    loading.value = true;
    emitAgent("", "imageInspect", imageRef.value, (res: any) => {
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

.layer-table {
    font-size: 0.9em;

    th, td {
        padding: 0.55rem 0.75rem;
    }

    th {
        font-weight: 600;
        color: $dark-font-color3;
        border-bottom-width: 1px;

        .dark & {
            color: $dark-font-color3;
        }
    }

    td {
        .dark & {
            color: $dark-font-color;
            border-color: $dark-border-color;
        }
    }

    code {
        font-family: 'JetBrains Mono', monospace;
        font-size: 0.9em;
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
