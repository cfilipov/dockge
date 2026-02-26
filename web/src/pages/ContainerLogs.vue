<template>
    <transition name="slide-fade" appear>
        <div v-if="containerName" class="logs-page">
            <h1 class="mb-3"><span v-if="badgeLabel" :class="badgeClass">{{ badgeLabel }}</span> {{ containerName }}</h1>

            <div v-if="stackName && stackManaged" class="mb-3">
                <div class="btn-group me-2" role="group">
                    <button v-if="!containerActive" class="btn btn-primary" :disabled="processing" :title="$t('tooltipServiceStart', [serviceName])" @click="startService">
                        <font-awesome-icon icon="play" class="me-1" />
                        {{ $t("startStack") }}
                    </button>

                    <button v-if="containerActive" class="btn btn-normal" :disabled="processing" :title="$t('tooltipServiceRestart', [serviceName])" @click="restartService">
                        <font-awesome-icon icon="rotate" class="me-1" />
                        {{ $t("restartStack") }}
                    </button>

                    <button class="btn" :class="imageUpdatesAvailable ? 'btn-info' : 'btn-normal'" :disabled="processing" :title="$t('tooltipServiceUpdate', [serviceName])" @click="showUpdateDialog = true">
                        <font-awesome-icon icon="cloud-arrow-down" class="me-1" />
                        <span class="d-none d-xl-inline">{{ $t("updateStack") }}</span>
                    </button>

                    <UpdateDialog
                        v-model="showUpdateDialog"
                        :stack-name="stackName"
                        :endpoint="endpoint"
                        :service-name="serviceName"
                        @update="doUpdate"
                    />

                    <button v-if="containerActive" class="btn btn-normal" :disabled="processing" :title="$t('tooltipServiceStop', [serviceName])" @click="stopService">
                        <font-awesome-icon icon="stop" class="me-1" />
                        {{ $t("stopStack") }}
                    </button>

                    <BDropdown right text="" variant="normal" menu-class="overflow-dropdown">
                        <BDropdownItem :title="$t('tooltipCheckUpdates')" @click="checkImageUpdates">
                            <font-awesome-icon icon="search" class="me-1" />
                            {{ $t("checkUpdates") }}
                        </BDropdownItem>
                    </BDropdown>
                </div>
            </div>

            <!-- Progress Terminal -->
            <ProgressTerminal
                ref="progressTerminalRef"
                class="mb-3"
                :name="composeTerminalName"
                :endpoint="endpoint"
            />

            <Terminal class="terminal flex-grow-1" :rows="20" mode="displayOnly"
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
import { ref, computed, onMounted } from "vue";
import { useRoute } from "vue-router";
import { useI18n } from "vue-i18n";
import { FontAwesomeIcon } from "@fortawesome/vue-fontawesome";
import { useSocket } from "../composables/useSocket";
import { useAppToast } from "../composables/useAppToast";
import { ContainerStatusInfo, getComposeTerminalName } from "../common/util-common";
import ProgressTerminal from "../components/ProgressTerminal.vue";
import UpdateDialog from "../components/UpdateDialog.vue";

const route = useRoute();
const { t } = useI18n();
const { emitAgent, containerList, completeStackList } = useSocket();
const { toastRes } = useAppToast();

const containerInfo = computed(() =>
    (containerList.value || []).find((c: any) => c.name === containerName.value)
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

const processing = ref(false);
const showUpdateDialog = ref(false);
const progressTerminalRef = ref<InstanceType<typeof ProgressTerminal>>();

const endpoint = computed(() => (route.params.endpoint as string) || "");
const containerName = computed(() => route.params.containerName as string || "");
const stackName = computed(() => containerInfo.value?.stackName || "");
const serviceName = computed(() => containerInfo.value?.serviceName || "");
const globalStack = computed(() => completeStackList.value[stackName.value + "_" + endpoint.value]);
const stackManaged = computed(() => globalStack.value?.isManagedByDockge ?? false);
const containerActive = computed(() => {
    const state = containerInfo.value?.state;
    return state === "running";
});
const imageUpdatesAvailable = computed(() => containerInfo.value?.imageUpdatesAvailable ?? false);
const composeTerminalName = computed(() => stackName.value ? getComposeTerminalName(endpoint.value, stackName.value) : "");
const terminalName = computed(() => "container-log-by-name--" + containerName.value);

function startComposeAction() {
    processing.value = true;
    progressTerminalRef.value?.show();
}

function stopComposeAction() {
    processing.value = false;
}

function startService() {
    startComposeAction();
    emitAgent(endpoint.value, "startService", stackName.value, serviceName.value, (res: any) => {
        stopComposeAction();
        toastRes(res);
    });
}

function stopService() {
    startComposeAction();
    emitAgent(endpoint.value, "stopService", stackName.value, serviceName.value, (res: any) => {
        stopComposeAction();
        toastRes(res);
    });
}

function restartService() {
    startComposeAction();
    emitAgent(endpoint.value, "restartService", stackName.value, serviceName.value, (res: any) => {
        stopComposeAction();
        toastRes(res);
    });
}

function doUpdate(data: { pruneAfterUpdate: boolean; pruneAllAfterUpdate: boolean }) {
    startComposeAction();
    emitAgent(endpoint.value, "updateService", stackName.value, serviceName.value, data.pruneAfterUpdate, data.pruneAllAfterUpdate, (res: any) => {
        stopComposeAction();
        toastRes(res);
    });
}

function checkImageUpdates() {
    processing.value = true;
    emitAgent(endpoint.value, "checkImageUpdates", stackName.value, (res: any) => {
        processing.value = false;
        toastRes(res);
    });
}

onMounted(() => {
    if (containerName.value) {
        emitAgent(endpoint.value, "joinContainerLogByName", containerName.value, () => {});
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

:deep(.overflow-dropdown) {
    background-color: $dark-bg;
    border-color: $dark-font-color3;

    .dropdown-item {
        color: $dark-font-color;

        &:hover {
            background-color: $dark-header-bg;
            color: $dark-font-color;
        }
    }
}
</style>
