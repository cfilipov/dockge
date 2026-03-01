<template>
    <transition name="slide-fade" appear>
        <div v-if="containerName" class="logs-page">
            <h1 class="mb-3"><span v-if="badgeLabel" :class="badgeClass">{{ badgeLabel }}</span> {{ containerName }}</h1>

            <div v-if="stackName && stackManaged" class="mb-3">
                <ServiceActionBar
                    :active="containerActive"
                    :processing="processing"
                    :image-updates-available="imageUpdatesAvailable"
                    :recreate-necessary="recreateNecessary"
                    :stack-name="stackName"
                    :service-name="serviceName"
                    @start="startService"
                    @stop="stopService"
                    @restart="restartService"
                    @recreate="recreateService"
                    @update="doUpdate"
                    @check-updates="checkImageUpdates"
                />
            </div>

            <!-- Progress Terminal -->
            <ProgressTerminal
                ref="progressTerminalRef"
                class="mb-3"
                :name="composeTerminalName"
            />

            <Terminal class="terminal flex-grow-1" :rows="20" mode="displayOnly"
                :name="terminalName" />
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
import { computed, onMounted } from "vue";
import { useRoute } from "vue-router";
import { useI18n } from "vue-i18n";
import { FontAwesomeIcon } from "@fortawesome/vue-fontawesome";
import { useSocket } from "../composables/useSocket";
import { useContainerStore } from "../stores/containerStore";
import { useStackStore } from "../stores/stackStore";
import { useUpdateStore } from "../stores/updateStore";
import { useServiceActions } from "../composables/useServiceActions";
import { ContainerStatusInfo, getComposeTerminalName } from "../common/util-common";
import ProgressTerminal from "../components/ProgressTerminal.vue";
import ServiceActionBar from "../components/ServiceActionBar.vue";
import { ref } from "vue";

const route = useRoute();
const { t } = useI18n();
const { emit } = useSocket();
const containerStore = useContainerStore();
const stackStoreInstance = useStackStore();
const updateStoreInstance = useUpdateStore();

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

const progressTerminalRef = ref<InstanceType<typeof ProgressTerminal>>();

const containerName = computed(() => route.params.containerName as string || "");
const stackName = computed(() => containerInfo.value?.stackName || "");
const serviceName = computed(() => containerInfo.value?.serviceName || "");
const globalStack = computed(() => stackStoreInstance.allStacks.find(s => s.name === stackName.value));
const stackManaged = computed(() => globalStack.value?.isManagedByDockge ?? false);
const containerActive = computed(() => {
    const state = containerInfo.value?.state;
    return state === "running";
});
const imageUpdatesAvailable = computed(() => {
    if (!stackName.value || !serviceName.value) return false;
    return updateStoreInstance.hasUpdate(`${stackName.value}/${serviceName.value}`);
});
const recreateNecessary = computed(() => {
    if (!containerInfo.value || !globalStack.value?.images) return false;
    const composeImage = globalStack.value.images[serviceName.value];
    return !!(composeImage && containerInfo.value.image && containerInfo.value.image !== composeImage);
});
const composeTerminalName = computed(() => stackName.value ? getComposeTerminalName(stackName.value) : "");
const terminalName = computed(() => "container-log-by-name-" + containerName.value);

const {
    processing, showUpdateDialog,
    startService, stopService, restartService, recreateService,
    doUpdate, checkImageUpdates,
} = useServiceActions(stackName, serviceName, progressTerminalRef);

onMounted(() => {
    if (containerName.value) {
        emit("joinContainerLogByName", containerName.value, () => {});
    }
});
</script>

<style scoped lang="scss">
@import "../styles/vars.scss";

.logs-page {
    display: flex;
    flex-direction: column;
    height: 100%;
}

.terminal {
    min-height: 0;
    margin-bottom: 1rem;
}

</style>
