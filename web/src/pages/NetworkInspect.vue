<template>
    <transition name="slide-fade" appear>
        <div v-if="networkName">
            <h1 class="mb-3"><span v-if="badgeLabel" :class="badgeClass">{{ badgeLabel }}</span> {{ networkName }}</h1>

            <div class="row">
                <div class="col-lg-8">
                    <!-- Connected Containers -->
                    <CollapsibleSection>
                        <template #heading>{{ $t("networkContainers") }} ({{ networkDetail?.containers?.length ?? 0 }})</template>
                        <div v-if="networkDetail && networkDetail.containers && networkDetail.containers.length > 0">
                            <div v-for="c in networkDetail.containers" :key="c.containerId" class="shadow-box big-padding mb-3">
                                <h5 class="mb-3">
                                    <span class="badge rounded-pill me-2" :class="'bg-' + containerBadgeColor(c)">{{ $t(containerStatusLabel(c)) }}</span>
                                    <router-link :to="{ name: 'containerDetail', params: { containerName: c.name } }" class="stack-link">{{ c.name }}</router-link>
                                </h5>
                                <div class="network-props">
                                    <div class="network-chip">
                                        <span class="chip-label">{{ $t("containerID") }}</span>
                                        <code :title="c.containerId">{{ c.containerId.substring(0, 12) }}</code>
                                    </div>
                                    <div class="network-chip">
                                        <span class="chip-label">{{ $t("networkIPv4") }}</span>
                                        <code>{{ c.ipv4 || '–' }}</code>
                                    </div>
                                    <div class="network-chip">
                                        <span class="chip-label">{{ $t("networkIPv6") }}</span>
                                        <code>{{ c.ipv6 || '–' }}</code>
                                    </div>
                                    <div class="network-chip">
                                        <span class="chip-label">{{ $t("networkMAC") }}</span>
                                        <code>{{ c.mac || '–' }}</code>
                                    </div>
                                </div>
                            </div>
                        </div>
                        <div v-else-if="networkDetail" class="shadow-box big-padding mb-3">
                            <p class="text-muted mb-0">{{ $t("noNetworkContainers") }}</p>
                        </div>
                        <div v-else class="shadow-box big-padding mb-3">
                            <p class="text-muted mb-0">{{ loading ? "Loading..." : "" }}</p>
                        </div>
                    </CollapsibleSection>
                </div>

                <div class="col-lg-4">
                    <!-- Overview Card -->
                    <h4 class="mb-3">{{ $t("containerOverview") }}</h4>
                    <div v-if="networkDetail" class="shadow-box big-padding mb-3">
                        <div class="overview-list">
                            <div class="overview-item">
                                <div class="overview-label">{{ $t("overviewName") }}</div>
                                <div class="overview-value">{{ networkDetail.name }}</div>
                            </div>

                            <div class="overview-item">
                                <div class="overview-label">{{ $t("containerID") }}</div>
                                <div class="overview-value">
                                    <code :title="networkDetail.id">{{ networkDetail.id.substring(0, 12) }}</code>
                                </div>
                            </div>

                            <div class="overview-item">
                                <div class="overview-label">{{ $t("networkDriver") }}</div>
                                <div class="overview-value">{{ networkDetail.driver }}</div>
                            </div>

                            <div class="overview-item">
                                <div class="overview-label">{{ $t("networkScope") }}</div>
                                <div class="overview-value">{{ networkDetail.scope }}</div>
                            </div>

                            <div v-if="networkDetail.created" class="overview-item">
                                <div class="overview-label">{{ $t("networkCreatedAt") }}</div>
                                <div class="overview-value">{{ formatDate(networkDetail.created) }}</div>
                            </div>

                            <div v-if="primarySubnet" class="overview-item">
                                <div class="overview-label">{{ $t("networkSubnet") }}</div>
                                <div class="overview-value"><code>{{ primarySubnet }}</code></div>
                            </div>

                            <div v-if="primaryGateway" class="overview-item">
                                <div class="overview-label">{{ $t("networkGatewayAddr") }}</div>
                                <div class="overview-value"><code>{{ primaryGateway }}</code></div>
                            </div>

                            <div class="overview-item">
                                <div class="overview-label">{{ $t("networkAttachable") }}</div>
                                <div class="overview-value">{{ networkDetail.attachable ? $t("yes") : $t("no") }}</div>
                            </div>

                            <div class="overview-item">
                                <div class="overview-label">{{ $t("networkInternal") }}</div>
                                <div class="overview-value">{{ networkDetail.internal ? $t("yes") : $t("no") }}</div>
                            </div>

                            <div class="overview-item">
                                <div class="overview-label">{{ $t("networkIPv6Enabled") }}</div>
                                <div class="overview-value">{{ networkDetail.ipv6 ? $t("yes") : $t("no") }}</div>
                            </div>

                            <div class="overview-item">
                                <div class="overview-label">{{ $t("networkIngress") }}</div>
                                <div class="overview-value">{{ networkDetail.ingress ? $t("yes") : $t("no") }}</div>
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
            <h1 class="mb-3">{{ $t("networksNav") }}</h1>
            <div class="shadow-box big-padding">
                <p class="text-muted mb-0">{{ $t("noNetworkSelected") }}</p>
            </div>
        </div>
    </transition>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from "vue";
import { useRoute } from "vue-router";
import { useI18n } from "vue-i18n";
import { useSocket } from "../composables/useSocket";
import { ContainerStatusInfo } from "../common/util-common";

const route = useRoute();
const { t } = useI18n();
const { emitAgent } = useSocket();

const networkDetail = ref<any>(null);
const loading = ref(false);

const networkName = computed(() => route.params.networkName as string || "");

const inUse = computed(() => {
    if (!networkDetail.value) return false;
    return (networkDetail.value.containers?.length ?? 0) > 0;
});
const badgeClass = computed(() =>
    networkDetail.value ? `badge rounded-pill ${inUse.value ? "bg-success" : "bg-warning"}` : ""
);
const badgeLabel = computed(() =>
    networkDetail.value ? (inUse.value ? t("networkInUse") : t("networkUnused")) : ""
);

const primarySubnet = computed(() => {
    if (!networkDetail.value?.ipam?.length) return "";
    return networkDetail.value.ipam[0].subnet || "";
});

const primaryGateway = computed(() => {
    if (!networkDetail.value?.ipam?.length) return "";
    return networkDetail.value.ipam[0].gateway || "";
});

function containerStatusLabel(c: Record<string, any>): string {
    return ContainerStatusInfo.from(c).label;
}

function containerBadgeColor(c: Record<string, any>): string {
    return ContainerStatusInfo.from(c).badgeColor;
}

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

function fetchDetail() {
    if (!networkName.value) {
        networkDetail.value = null;
        return;
    }
    loading.value = true;
    emitAgent("", "networkInspect", networkName.value, (res: any) => {
        loading.value = false;
        if (res.ok && res.networkDetail) {
            networkDetail.value = res.networkDetail;
        }
    });
}

watch(networkName, fetchDetail);

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
</style>
