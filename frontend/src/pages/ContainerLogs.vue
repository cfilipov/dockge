<template>
    <transition name="slide-fade" appear>
        <div v-if="containerName">
            <h1 class="mb-3"><span v-if="badgeLabel" :class="badgeClass">{{ badgeLabel }}</span> {{ containerName }}</h1>

            <Terminal class="terminal" :rows="20" mode="displayOnly"
                :name="terminalName" :endpoint="endpoint" />
        </div>
        <div v-else>
            <h1 class="mb-3">{{ $t("logs") }}</h1>
            <div class="shadow-box big-padding">
                <p class="text-muted mb-0">{{ $t("selectContainer") }}</p>
            </div>
        </div>
    </transition>
</template>

<script setup lang="ts">
import { computed, onMounted, watch } from "vue";
import { useRoute } from "vue-router";
import { useI18n } from "vue-i18n";
import { useSocket } from "../composables/useSocket";
import { StackStatusInfo } from "../../../common/util-common";

const route = useRoute();
const { t } = useI18n();
const { emitAgent, containerList } = useSocket();

function getContainerStatusLabel(c: Record<string, any>): string {
    if (c.state === "running" && c.health === "unhealthy") return "unhealthy";
    if (c.state === "running") return "active";
    if (c.state === "exited" || c.state === "dead") return "exited";
    if (c.state === "paused") return "active";
    if (c.state === "created") return "down";
    return "down";
}

const containerInfo = computed(() =>
    (containerList.value || []).find((c: any) => c.name === containerName.value)
);
const statusInfo = computed(() => {
    if (!containerInfo.value) return null;
    const label = getContainerStatusLabel(containerInfo.value);
    return StackStatusInfo.ALL.find(i => i.label === label);
});
const badgeClass = computed(() =>
    statusInfo.value ? `badge rounded-pill bg-${statusInfo.value.badgeColor}` : ""
);
const badgeLabel = computed(() =>
    statusInfo.value ? t(statusInfo.value.label) : ""
);

const containerName = computed(() => route.params.containerName as string || "");
const endpoint = computed(() => "");
const terminalName = computed(() => "container-log-by-name--" + containerName.value);

onMounted(() => {
    if (containerName.value) {
        emitAgent(endpoint.value, "joinContainerLogByName", containerName.value, () => {});
    }
});
</script>

<style scoped lang="scss">
.terminal {
    height: 410px;
}
</style>
