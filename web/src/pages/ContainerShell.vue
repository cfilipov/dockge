<template>
    <transition name="slide-fade" appear>
        <div v-if="containerName">
            <h1 class="mb-3"><span v-if="badgeLabel" :class="badgeClass">{{ badgeLabel }}</span> {{ containerName }}</h1>

            <div class="mb-3">
                <router-link :to="switchShellLink" class="btn btn-normal me-2">{{ $t(switchShellLabel) }}</router-link>
            </div>

            <Terminal class="terminal" :rows="20" mode="interactive"
                :name="terminalName"
                :container-name="containerName" :shell="shell" />
        </div>
        <div v-else>
            <h1 class="mb-3">{{ $t("shell") }}</h1>
            <div class="shadow-box big-padding">
                <p class="text-muted mb-0">{{ $t("selectContainer") }}</p>
            </div>
        </div>
    </transition>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useRoute } from "vue-router";
import { useI18n } from "vue-i18n";
import { useContainerStore } from "../stores/containerStore";
import { ContainerStatusInfo } from "../common/util-common";

const route = useRoute();
const { t } = useI18n();
const containerStore = useContainerStore();

const containerInfo = computed(() =>
    containerStore.containers.find(c => c.name === containerName.value)
);
const statusInfo = computed(() =>
    containerInfo.value ? ContainerStatusInfo.from(containerInfo.value) : null
);
const badgeClass = computed(() =>
    statusInfo.value ? `badge rounded-pill bg-${statusInfo.value.badgeColor}` : ""
);
const badgeLabel = computed(() =>
    statusInfo.value ? t(statusInfo.value.label) : ""
);

const containerName = computed(() => route.params.containerName as string || "");
const shell = computed(() => (route.params.type as string) || "bash");
const terminalName = computed(() => "container-exec-by-name-" + containerName.value);

const alternateShell = computed(() => shell.value === "bash" ? "sh" : "bash");
const switchShellLabel = computed(() => shell.value === "bash" ? "Switch to sh" : "Switch to bash");
const switchShellLink = computed(() => ({
    name: "containerShell",
    params: {
        containerName: containerName.value,
        type: alternateShell.value,
    },
}));
</script>

<style scoped lang="scss">
.terminal {
    height: 410px;
}
</style>
