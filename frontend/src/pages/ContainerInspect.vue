<template>
    <transition name="slide-fade" appear>
        <div v-if="containerName">
            <h1 class="mb-3"><span v-if="badgeLabel" :class="badgeClass">{{ badgeLabel }}</span> {{ containerName }}</h1>

            <div v-if="stackName && stackManaged" class="mb-3">
                <div class="btn-group me-2" role="group">
                    <button v-if="!stackActive" class="btn btn-primary" :disabled="processing" :title="$t('tooltipStackStart')" @click="startStack">
                        <font-awesome-icon icon="play" class="me-1" />
                        {{ $t("startStack") }}
                    </button>

                    <button v-if="stackActive" class="btn btn-normal" :disabled="processing" :title="$t('tooltipStackRestart')" @click="restartStack">
                        <font-awesome-icon icon="rotate" class="me-1" />
                        {{ $t("restartStack") }}
                    </button>

                    <button class="btn" :class="imageUpdatesAvailable ? 'btn-info' : 'btn-normal'" :disabled="processing" :title="$t('tooltipStackUpdate')" @click="showUpdateDialog = true">
                        <font-awesome-icon icon="cloud-arrow-down" class="me-1" />
                        <span class="d-none d-xl-inline">{{ $t("updateStack") }}</span>
                    </button>

                    <BModal v-model="showUpdateDialog" :title="$t('updateStack')" :close-on-esc="true" @show="resetUpdateDialog" @hidden="resetUpdateDialog">
                        <p class="mb-3" v-html="$t('updateStackMsg')"></p>

                        <BForm>
                            <BFormCheckbox v-model="updateDialogData.pruneAfterUpdate" switch><span v-html="$t('pruneAfterUpdate')"></span></BFormCheckbox>
                            <div style="margin-left: 2.5rem;">
                                <BFormCheckbox v-model="updateDialogData.pruneAllAfterUpdate" :checked="updateDialogData.pruneAfterUpdate && updateDialogData.pruneAllAfterUpdate" :disabled="!updateDialogData.pruneAfterUpdate"><span v-html="$t('pruneAllAfterUpdate')"></span></BFormCheckbox>
                            </div>
                        </BForm>

                        <template #footer>
                            <button class="btn btn-primary" @click="updateStack">
                                <font-awesome-icon icon="cloud-arrow-down" class="me-1" />{{ $t("updateStack") }}
                            </button>
                        </template>
                    </BModal>

                    <button v-if="stackActive" class="btn btn-normal" :disabled="processing" :title="$t('tooltipStackStop')" @click="stopStack">
                        <font-awesome-icon icon="stop" class="me-1" />
                        {{ $t("stopStack") }}
                    </button>

                    <BDropdown right text="" variant="normal" menu-class="overflow-dropdown">
                        <BDropdownItem :title="$t('tooltipCheckUpdates')" @click="checkImageUpdates">
                            <font-awesome-icon icon="search" class="me-1" />
                            {{ $t("checkUpdates") }}
                        </BDropdownItem>
                        <BDropdownItem :title="$t('tooltipStackDown')" @click="downStack">
                            <font-awesome-icon icon="stop" class="me-1" />
                            {{ $t("downStack") }}
                        </BDropdownItem>
                    </BDropdown>
                </div>
            </div>

            <!-- Progress Terminal -->
            <ProgressTerminal
                ref="progressTerminalRef"
                class="mb-3"
                :name="terminalName"
                :endpoint="endpoint"
            />

            <div class="shadow-box mb-3 editor-box">
                <code-mirror
                    v-model="inspectData"
                    :extensions="extensionsYAML"
                    minimal
                    :wrap="true"
                    :dark="true"
                    :tab="true"
                    :disabled="true"
                />
            </div>
        </div>
        <div v-else>
            <h1 class="mb-3">{{ $t("containersNav") }}</h1>
            <div class="shadow-box big-padding">
                <p class="text-muted mb-0">{{ $t("selectContainer") }}</p>
            </div>
        </div>
    </transition>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from "vue";
import { useRoute } from "vue-router";
import { useI18n } from "vue-i18n";
import CodeMirror from "vue-codemirror6";
import { yaml as yamlLang } from "@codemirror/lang-yaml";
import { dracula as editorTheme } from "thememirror";
import { lineNumbers } from "@codemirror/view";
import yaml from "yaml";
import { FontAwesomeIcon } from "@fortawesome/vue-fontawesome";
import { BModal } from "bootstrap-vue-next";
import { useSocket } from "../composables/useSocket";
import { useAppToast } from "../composables/useAppToast";
import { StackStatusInfo, RUNNING, getComposeTerminalName } from "../../../common/util-common";
import ProgressTerminal from "../components/ProgressTerminal.vue";

const route = useRoute();
const { t } = useI18n();
const { emitAgent, containerList, completeStackList } = useSocket();
const { toastRes } = useAppToast();

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

const inspectData = ref("fetching ...");
const processing = ref(false);
const showUpdateDialog = ref(false);
const updateDialogData = reactive({
    pruneAfterUpdate: false,
    pruneAllAfterUpdate: false,
});
const progressTerminalRef = ref<InstanceType<typeof ProgressTerminal>>();

const extensionsYAML = [
    editorTheme,
    yamlLang(),
    lineNumbers(),
];

const endpoint = computed(() => (route.params.endpoint as string) || "");
const containerName = computed(() => route.params.containerName as string || "");
const stackName = computed(() => containerInfo.value?.stackName || "");
const globalStack = computed(() => completeStackList.value[stackName.value + "_" + endpoint.value]);
const stackActive = computed(() => globalStack.value?.status === RUNNING);
const stackManaged = computed(() => globalStack.value?.isManagedByDockge ?? false);
const imageUpdatesAvailable = computed(() => globalStack.value?.imageUpdatesAvailable ?? false);
const terminalName = computed(() => stackName.value ? getComposeTerminalName(endpoint.value, stackName.value) : "");

function startComposeAction() {
    processing.value = true;
    progressTerminalRef.value?.show();
}

function stopComposeAction() {
    processing.value = false;
    progressTerminalRef.value?.hideWithTimeout();
}

function startStack() {
    startComposeAction();
    emitAgent(endpoint.value, "startStack", stackName.value, (res: any) => {
        stopComposeAction();
        toastRes(res);
    });
}

function stopStack() {
    startComposeAction();
    emitAgent(endpoint.value, "stopStack", stackName.value, (res: any) => {
        stopComposeAction();
        toastRes(res);
    });
}

function restartStack() {
    startComposeAction();
    emitAgent(endpoint.value, "restartStack", stackName.value, (res: any) => {
        stopComposeAction();
        toastRes(res);
    });
}

function resetUpdateDialog() {
    updateDialogData.pruneAfterUpdate = false;
    updateDialogData.pruneAllAfterUpdate = false;
}

function updateStack() {
    showUpdateDialog.value = false;
    startComposeAction();
    emitAgent(endpoint.value, "updateStack", stackName.value, updateDialogData.pruneAfterUpdate, updateDialogData.pruneAllAfterUpdate, (res: any) => {
        stopComposeAction();
        toastRes(res);
    });
}

function downStack() {
    startComposeAction();
    emitAgent(endpoint.value, "downStack", stackName.value, (res: any) => {
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
        emitAgent(endpoint.value, "containerInspect", containerName.value, (res: any) => {
            if (res.ok) {
                const inspectObj = JSON.parse(res.inspectData);
                if (inspectObj) {
                    inspectData.value = yaml.stringify(inspectObj, { lineWidth: 0 });
                }
            }
        });
    }
});
</script>

<style scoped lang="scss">
@import "../styles/vars.scss";

.editor-box {
    font-family: 'JetBrains Mono', monospace;
    font-size: 14px;
    height: 500px;
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
