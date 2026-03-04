<template>
    <transition name="slide-fade" appear>
        <div v-if="networkName">
            <h1 class="mb-3"><span v-if="badgeLabel" :class="badgeClass">{{ badgeLabel }}</span> {{ networkName }}</h1>

            <div class="row">
                <div class="col-lg-8">
                    <!-- Connected Containers -->
                    <CollapsibleSection>
                        <template #heading>{{ $t("networkContainers") }} <span class="section-count">({{ networkContainers.length }})</span></template>
                        <div v-if="networkContainers.length > 0">
                            <ContainerCard v-for="c in networkContainers" :key="c.containerId" :container="c">
                                <div class="info-chip">
                                    <span class="chip-label">{{ $t("networkIPv4") }}</span>
                                    <code>{{ c.networks[networkName]?.ipv4 || '–' }}</code>
                                </div>
                                <div class="info-chip">
                                    <span class="chip-label">{{ $t("networkIPv6") }}</span>
                                    <code>{{ c.networks[networkName]?.ipv6 || '–' }}</code>
                                </div>
                                <div class="info-chip">
                                    <span class="chip-label">{{ $t("networkMAC") }}</span>
                                    <code>{{ c.networks[networkName]?.mac || '–' }}</code>
                                </div>
                            </ContainerCard>
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
                        <div class="overview-list" role="region" :aria-label="$t('containerOverview')">
                            <div class="overview-item">
                                <div class="overview-label">{{ $t("overviewName") }}</div>
                                <div class="overview-value">{{ networkDetail.name }}</div>
                            </div>

                            <div class="overview-item">
                                <div class="overview-label">{{ $t("networkID") }}</div>
                                <div class="overview-value">
                                    <code class="truncate-id" :title="networkDetail.id">{{ networkDetail.id }}</code>
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
import { useContainerStore } from "../stores/containerStore";
import ContainerCard from "../components/ContainerCard.vue";

const route = useRoute();
const { t } = useI18n();
const { emit } = useSocket();
const containerStore = useContainerStore();

const networkDetail = ref<any>(null);
const loading = ref(false);

const networkName = computed(() => route.params.networkName as string || "");

const networkContainers = computed(() => {
    if (!networkName.value) return [];
    return containerStore.byNetwork(networkName.value);
});

const inUse = computed(() => networkContainers.value.length > 0);
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
    emit("networkInspect", networkName.value, (res: any) => {
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
@import "../styles/info-chips";

.overview-value code {
    padding: 0;
}
</style>
