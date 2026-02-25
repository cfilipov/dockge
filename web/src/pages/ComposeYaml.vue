<template>
    <transition name="slide-fade" appear>
        <div v-if="stackName">
            <h1 class="mb-3">
                <Uptime :stack="globalStack" :pill="true" /> {{ stack.name }}
                <span v-if="agentCount > 1" class="agent-name">
                    ({{ endpointDisplay }})
                </span>
            </h1>

            <div v-if="stack.isManagedByDockge" class="d-flex align-items-center justify-content-between mb-3">
                <div class="d-flex align-items-center">
                    <div class="btn-group me-2" role="group">
                        <button v-if="isEditMode" class="btn btn-primary" :disabled="processing" :title="$t('tooltipStackDeploy')" @click="deployStack">
                            <font-awesome-icon icon="rocket" class="me-1" />
                            {{ $t("deployStack") }}
                        </button>

                        <button v-if="isEditMode" class="btn btn-normal" :disabled="processing" :title="$t('tooltipStackSave')" @click="saveStack">
                            <font-awesome-icon icon="save" class="me-1" />
                            {{ $t("saveStackDraft") }}
                        </button>

                        <button v-if="!isEditMode" class="btn btn-secondary" :disabled="processing" :title="$t('tooltipStackEdit')" @click="enableEditMode">
                            <font-awesome-icon icon="pen" class="me-1" />
                            {{ $t("editStack") }}
                        </button>

                        <button v-if="!isEditMode && !active" class="btn btn-primary" :disabled="processing" :title="$t('tooltipStackStart')" @click="startStack">
                            <font-awesome-icon icon="play" class="me-1" />
                            {{ $t("startStack") }}
                        </button>

                        <button v-if="!isEditMode && active" class="btn btn-normal" :disabled="processing" :title="$t('tooltipStackRestart')" @click="restartStack">
                            <font-awesome-icon icon="rotate" class="me-1" />
                            {{ $t("restartStack") }}
                        </button>

                        <button v-if="!isEditMode" class="btn" :class="stack.imageUpdatesAvailable ? 'btn-info' : 'btn-normal'" :disabled="processing" :title="$t('tooltipStackUpdate')" @click="showUpdateDialog = true">
                            <font-awesome-icon icon="cloud-arrow-down" class="me-1" />
                            <span class="d-none d-xl-inline">{{ $t("updateStack") }}</span>
                        </button>

                        <BModal v-model="showUpdateDialog" :title="$t('updateStack')" :close-on-esc="true" @show="resetUpdateDialog" @hidden="resetUpdateDialog">
                            <p class="mb-3" v-html="$t('updateStackMsg')"></p>

                            <div v-if="changelogLinks.length > 0" class="mb-3">
                                <h5>{{ $t("changelog") }}</h5>
                                <div v-for="link in changelogLinks" :key="link.service">
                                    <strong>{{ link.service }}:</strong>{{ " " }}
                                    <a :href="link.url" target="_blank">{{ link.url }}</a>
                                </div>
                            </div>

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

                        <button v-if="!isEditMode && active" class="btn btn-normal" :disabled="processing" :title="$t('tooltipStackStop')" @click="stopStack">
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
                            <BDropdownItem v-if="!isEditMode && !errorDelete" :title="$t('tooltipStackDelete')" @click="showDeleteDialog = !showDeleteDialog">
                                <font-awesome-icon icon="trash" class="me-1 text-danger" />
                                {{ $t("deleteStack") }}
                            </BDropdownItem>
                            <BDropdownItem v-if="errorDelete" :title="$t('tooltipStackForceDelete')" @click="showForceDeleteDialog = !showForceDeleteDialog">
                                <font-awesome-icon icon="trash" class="me-1 text-danger" />
                                {{ $t("forceDeleteStack") }}
                            </BDropdownItem>
                        </BDropdown>
                    </div>

                    <button v-if="isEditMode" class="btn btn-normal" :disabled="processing" :title="$t('tooltipStackDiscard')" @click="discardStack">{{ $t("discardStack") }}</button>
                </div>

                <!-- Parsed / Raw toggle -->
                <div class="btn-group" role="group">
                    <input
                        id="yaml-view-parsed"
                        v-model="viewMode"
                        type="radio"
                        class="btn-check"
                        name="yamlViewMode"
                        autocomplete="off"
                        value="parsed"
                    />
                    <label class="btn btn-outline-primary" for="yaml-view-parsed">
                        {{ $t("parsed") }}
                    </label>

                    <input
                        id="yaml-view-raw"
                        v-model="viewMode"
                        type="radio"
                        class="btn-check"
                        name="yamlViewMode"
                        autocomplete="off"
                        value="raw"
                    />
                    <label class="btn btn-outline-primary" for="yaml-view-raw">
                        {{ $t("raw") }}
                    </label>
                </div>
            </div>

            <!-- URLs -->
            <div v-if="urls.length > 0" class="mb-3">
                <a v-for="(urlItem, index) in urls" :key="index" target="_blank" :href="urlItem.url">
                    <span class="badge bg-secondary me-2">{{ urlItem.display }}</span>
                </a>
            </div>

            <!-- Progress Terminal -->
            <ProgressTerminal
                ref="progressTerminalRef"
                class="mb-3"
                :name="terminalName"
                :endpoint="endpoint"
                :rows="progressTerminalRows"
            />

            <div v-if="stack.isManagedByDockge">
                <!-- Override YAML editor (only show if file exists) -->
                <div v-if="stack.composeOverrideYAML && stack.composeOverrideYAML.trim() !== ''">
                    <h4 class="mb-3">{{ stack.composeOverrideFileName || 'compose.override.yaml' }}</h4>
                    <div class="shadow-box mb-3 editor-box" :class="{'edit-mode' : isEditMode}">
                        <button v-if="isEditMode" v-b-modal.yaml-compose-override-editor-modal class="expand-button">
                            <font-awesome-icon icon="expand" />
                        </button>
                        <code-mirror
                            v-model="stack.composeOverrideYAML"
                            :extensions="extensions"
                            minimal
                            :wrap="true"
                            :dark="true"
                            :tab="true"
                            :disabled="!isEditMode"
                            :hasFocus="editorFocus"
                            @change="yamlCodeChange"
                        />
                    </div>
                    <div v-if="isEditMode" class="mb-3">
                        {{ yamlError }}
                    </div>

                    <!-- Override modal fullscreen editor -->
                    <BModal id="yaml-compose-override-editor-modal" :title="stack.composeOverrideFileName || 'compose.override.yaml'" scrollable size="fullscreen" hide-footer>
                        <div class="shadow-box mb-3 editor-box" :class="{'edit-mode' : isEditMode}">
                            <code-mirror
                                v-model="stack.composeOverrideYAML"
                                :extensions="extensions"
                                minimal
                                :wrap="true"
                                :dark="true"
                                :tab="true"
                                :disabled="!isEditMode"
                                :hasFocus="editorFocus"
                                @change="yamlCodeChange"
                            />
                        </div>
                        <div v-if="isEditMode" class="mb-3">
                            {{ yamlError }}
                        </div>
                    </BModal>
                </div>

                <h4 class="mb-3">{{ stack.composeFileName }}</h4>

                <!-- YAML editor (inline) -->
                <div class="shadow-box mb-3 editor-box" :class="{'edit-mode' : isEditMode}">
                    <button v-if="isEditMode" v-b-modal.yaml-compose-editor-modal class="expand-button">
                        <font-awesome-icon icon="expand" />
                    </button>
                    <code-mirror
                        v-model="stack.composeYAML"
                        :extensions="extensions"
                        minimal
                        :wrap="true"
                        :dark="true"
                        :tab="true"
                        :disabled="!isEditMode"
                        :hasFocus="editorFocus"
                        @change="yamlCodeChange"
                    />
                </div>
                <div v-if="isEditMode" class="mb-3">
                    {{ yamlError }}
                </div>

                <!-- YAML modal fullscreen editor -->
                <BModal id="yaml-compose-editor-modal" :title="stack.composeFileName" scrollable size="fullscreen" hide-footer>
                    <div class="shadow-box mb-3 editor-box" :class="{'edit-mode' : isEditMode}">
                        <code-mirror
                            v-model="stack.composeYAML"
                            :extensions="extensions"
                            minimal
                            :wrap="true"
                            :dark="true"
                            :tab="true"
                            :disabled="!isEditMode"
                            :hasFocus="editorFocus"
                            @change="yamlCodeChange"
                        />
                    </div>
                    <div v-if="isEditMode" class="mb-3">
                        {{ yamlError }}
                    </div>
                </BModal>

                <!-- ENV editor -->
                <div v-if="isEditMode">
                    <h4 class="mb-3">.env</h4>
                    <div class="shadow-box mb-3 editor-box" :class="{'edit-mode' : isEditMode}">
                        <button v-if="isEditMode" v-b-modal.yaml-env-editor-modal class="expand-button">
                            <font-awesome-icon icon="expand" />
                        </button>
                        <code-mirror
                            v-model="stack.composeENV"
                            :extensions="extensionsEnv"
                            minimal
                            :wrap="true"
                            :dark="true"
                            :tab="true"
                            :disabled="!isEditMode"
                            :hasFocus="editorFocus"
                            @change="yamlCodeChange"
                        />
                    </div>
                </div>

                <!-- ENV modal fullscreen editor -->
                <BModal id="yaml-env-editor-modal" title=".env" scrollable size="fullscreen" hide-footer>
                    <div class="shadow-box mb-3 editor-box" :class="{'edit-mode' : isEditMode}">
                        <code-mirror
                            v-model="stack.composeENV"
                            :extensions="extensionsEnv"
                            minimal
                            :wrap="true"
                            :dark="true"
                            :tab="true"
                            :disabled="!isEditMode"
                            :hasFocus="editorFocus"
                            @change="yamlCodeChange"
                        />
                    </div>
                </BModal>

                <div v-if="isEditMode">
                    <!-- Networks -->
                    <h4 class="mb-3">{{ $tc("network", 2) }}</h4>
                    <div class="shadow-box big-padding mb-3">
                        <NetworkInput />
                    </div>
                </div>
            </div>

            <div v-if="!stack.isManagedByDockge && !processing">
                {{ $t("stackNotManagedByDockgeMsg") }}
            </div>

            <!-- Delete Dialog -->
            <BModal v-model="showDeleteDialog" :cancelTitle="$t('cancel')" :okTitle="$t('deleteStack')" okVariant="danger" @ok="deleteDialog">
                {{ $t("deleteStackMsg") }}
                <div class="form-check mt-4">
                    <label><input v-model="deleteStackFiles" class="form-check-input" type="checkbox" />{{
                        $t("deleteStackFilesConfirmation") }}</label>
                </div>
            </BModal>

            <!-- Force Delete Dialog -->
            <BModal v-model="showForceDeleteDialog" :okTitle="$t('forceDeleteStack')" okVariant="danger" @ok="forceDeleteDialog">
                {{ $t("forceDeleteStackMsg") }}
            </BModal>
        </div>
        <div v-else>
            <h1 class="mb-3">{{ $t("compose") }}</h1>
            <div class="shadow-box big-padding">
                <p class="text-muted mb-0">{{ $t("selectStack") }}</p>
            </div>
        </div>
    </transition>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch, provide, onMounted } from "vue";
import { useRoute, useRouter, onBeforeRouteUpdate, onBeforeRouteLeave } from "vue-router";
import { useI18n } from "vue-i18n";
import CodeMirror from "vue-codemirror6";
import { yaml } from "@codemirror/lang-yaml";
import { python } from "@codemirror/lang-python";
import { dracula as editorTheme } from "thememirror";
import { lineNumbers, EditorView } from "@codemirror/view";
import { indentUnit, indentService } from "@codemirror/language";
import { parseDocument, Document } from "yaml";
import { FontAwesomeIcon } from "@fortawesome/vue-fontawesome";
import {
    copyYAMLComments, envsubstYAML,
    getComposeTerminalName,
    PROGRESS_TERMINAL_ROWS,
} from "../common/util-common";
import { BModal } from "bootstrap-vue-next";
import { LABEL_IMAGEUPDATES_CHANGELOG, LABEL_URLS_PREFIX } from "../common/compose-labels";
import NetworkInput from "../components/NetworkInput.vue";
import ProgressTerminal from "../components/ProgressTerminal.vue";
import dotenv from "dotenv";
import { useSocket } from "../composables/useSocket";
import { useAppToast } from "../composables/useAppToast";

const route = useRoute();
const router = useRouter();
const { t } = useI18n();
const { emitAgent, agentCount, completeStackList, endpointDisplayFunction, info } = useSocket();
const { toastRes, toastError } = useAppToast();

// CodeMirror setup
const editorFocus = ref(false);

const focusEffectHandler = (state: any, focusing: boolean) => {
    editorFocus.value = focusing;
    return null;
};

const yamlIndent = indentService.of((cx: any, pos: number) => {
    const line = cx.lineAt(pos);
    if (line.number === 1) {
        return 0;
    }
    const prev = cx.lineAt(line.from - 1);
    const prevText = prev.text;
    const prevIndent = prevText.match(/^\s*/)[0].length;
    const trimmed = prevText.trimEnd();
    if (trimmed.endsWith(":") || trimmed.endsWith("|-") || trimmed.endsWith("|") || trimmed.endsWith(">") || trimmed.endsWith(">-")) {
        return prevIndent + 2;
    }
    return prevIndent;
});

const extensions = [
    editorTheme,
    yaml(),
    indentUnit.of("  "),
    yamlIndent,
    lineNumbers(),
    EditorView.focusChangeEffect.of(focusEffectHandler)
];

const extensionsEnv = [
    editorTheme,
    python(),
    lineNumbers(),
    EditorView.focusChangeEffect.of(focusEffectHandler)
];

// Timeouts
let yamlErrorTimeout: ReturnType<typeof setTimeout> | null = null;
let serviceStatusTimeout: ReturnType<typeof setTimeout> | null = null;

// YAML document for comment preservation
let yamlDoc: any = null;

// Template refs
const progressTerminalRef = ref<InstanceType<typeof ProgressTerminal>>();

// Data
const jsonConfig = reactive<Record<string, any>>({});
const envsubstJSONConfig = reactive<Record<string, any>>({});
const yamlError = ref("");
const processing = ref(true);
const progressTerminalRows = PROGRESS_TERMINAL_ROWS;
const stack = reactive<Record<string, any>>({
    composeOverrideYAML: "",
});
const serviceUpdateStatus = ref<Record<string, any>>({});
const serviceRecreateStatus = ref<Record<string, any>>({});
const isEditMode = ref(false);
const errorDelete = ref(false);
const showDeleteDialog = ref(false);
const deleteStackFiles = ref(false);
const showForceDeleteDialog = ref(false);
const showUpdateDialog = ref(false);
const updateDialogData = reactive({
    pruneAfterUpdate: false,
    pruneAllAfterUpdate: false,
});
const viewMode = ref<"parsed" | "raw">("raw");
const stopServiceStatusTimeout = ref(false);

// Provide to children (NetworkInput)
provide("jsonConfig", jsonConfig);
provide("envsubstJSONConfig", envsubstJSONConfig);
provide("composeStack", stack);
provide("composeEndpoint", computed(() => endpoint.value));
provide("editorFocus", editorFocus);
provide("startComposeAction", startComposeAction);
provide("stopComposeAction", stopComposeAction);

// Computed
const stackName = computed(() => (route.params.stackName as string) || "");

const endpointDisplay = computed(() => endpointDisplayFunction(endpoint.value));

const urls = computed(() => {
    const result: { display: string; url: string }[] = [];
    const services = envsubstJSONConfig.services;
    if (!services) {
        return result;
    }
    for (const svc of Object.values(services) as any[]) {
        const labels = svc?.labels;
        if (!labels) {
            continue;
        }
        for (const [key, value] of Object.entries(labels)) {
            if (key.startsWith(LABEL_URLS_PREFIX) && value) {
                let display;
                try {
                    let obj = new URL(value as string);
                    let pathname = obj.pathname;
                    if (pathname === "/") {
                        pathname = "";
                    }
                    display = obj.host + pathname + obj.search;
                } catch (e) {
                    display = value;
                }
                result.push({ display: display as string, url: value as string });
            }
        }
    }
    return result;
});

const changelogLinks = computed(() => {
    const links: { service: string; url: string }[] = [];
    const services = envsubstJSONConfig.services;
    if (!services) {
        return links;
    }
    for (const [name, svc] of Object.entries(services) as [string, any][]) {
        const url = svc?.labels?.[LABEL_IMAGEUPDATES_CHANGELOG];
        if (url) {
            links.push({ service: name, url });
        }
    }
    return links;
});

const globalStack = computed(() => completeStackList.value[stack.name + "_" + endpoint.value]);

const active = computed(() => globalStack.value?.started ?? false);

const terminalName = computed(() => {
    if (!stack.name) {
        return "";
    }
    return getComposeTerminalName(endpoint.value, stack.name);
});

const endpoint = computed(() => stack.endpoint || (route.params.endpoint as string) || "");

const yamlUrl = computed(() => {
    if (stack.endpoint) {
        return `/stacks/${stack.name}/raw/${stack.endpoint}`;
    }
    return `/stacks/${stack.name}/raw`;
});

// Watchers
watch(() => stack.composeYAML, () => {
    if (editorFocus.value) {
        yamlCodeChange();
    }
});

watch(() => stack.composeENV, () => {
    if (editorFocus.value) {
        yamlCodeChange();
    }
});

watch(() => stack.composeOverrideYAML, () => {
    if (editorFocus.value) {
        yamlCodeChange();
    }
});

watch(jsonConfig, () => {
    if (!editorFocus.value) {
        const doc = new Document(jsonConfig);

        if (yamlDoc) {
            copyYAMLComments(doc, yamlDoc);
        }

        stack.composeYAML = doc.toString();
        yamlDoc = doc;
    }
}, { deep: true });

// Navigate to parsed view when toggle changes
watch(viewMode, (mode) => {
    if (mode === "parsed" && stack.name) {
        const parsedUrl = stack.endpoint
            ? `/stacks/${stack.name}/${stack.endpoint}`
            : `/stacks/${stack.name}`;
        router.push(parsedUrl);
    }
});

// Navigation guards
onBeforeRouteUpdate((to, from, next) => {
    exitConfirm(next);
});

onBeforeRouteLeave((to, from, next) => {
    exitConfirm(next);
});

// Methods
function startServiceStatusTimeout() {
    clearTimeout(serviceStatusTimeout!);
    serviceStatusTimeout = setTimeout(async () => {
        requestServiceStatus();
    }, 5000);
}

function requestServiceStatus() {
    if (!stackName.value) {
        return;
    }

    emitAgent(endpoint.value, "serviceStatusList", stack.name, (res: any) => {
        if (res.ok) {
            serviceUpdateStatus.value = res.serviceUpdateStatus || {};
            serviceRecreateStatus.value = res.serviceRecreateStatus || {};
            stack.imageUpdatesAvailable = Object.values(serviceUpdateStatus.value).some((v: any) => v === true);
        }
        if (!stopServiceStatusTimeout.value) {
            startServiceStatusTimeout();
        }
    });
}

function exitConfirm(next: (val?: boolean | undefined) => void) {
    if (isEditMode.value) {
        if (confirm(t("confirmLeaveStack"))) {
            exitAction();
            next();
        } else {
            next(false);
        }
    } else {
        exitAction();
        next();
    }
}

function exitAction() {
    stopServiceStatusTimeout.value = true;
    clearTimeout(serviceStatusTimeout!);
}

function startComposeAction() {
    processing.value = true;
    progressTerminalRef.value?.show();
}

function stopComposeAction() {
    processing.value = false;
}

function loadStack() {
    processing.value = true;
    emitAgent(endpoint.value, "getStack", stack.name, (res: any) => {
        if (res.ok) {
            Object.assign(stack, res.stack);
            yamlCodeChange();
            processing.value = false;
        } else {
            toastRes(res);
        }
    });
}

function deployStack() {
    if (!jsonConfig.services) {
        toastError("No services found in compose.yaml");
        return;
    }

    if (typeof jsonConfig.services !== "object") {
        toastError("Services must be an object");
        return;
    }

    startComposeAction();

    emitAgent(stack.endpoint, "deployStack", stack.name, stack.composeYAML, stack.composeENV, stack.composeOverrideYAML || "", false, (res: any) => {
        stopComposeAction();
        toastRes(res);

        if (res.ok) {
            isEditMode.value = false;
        }
    });
}

function saveStack() {
    processing.value = true;

    emitAgent(stack.endpoint, "saveStack", stack.name, stack.composeYAML, stack.composeENV, stack.composeOverrideYAML || "", false, (res: any) => {
        processing.value = false;
        toastRes(res);

        if (res.ok) {
            isEditMode.value = false;
        }
    });
}

function startStack() {
    startComposeAction();

    emitAgent(endpoint.value, "startStack", stack.name, (res: any) => {
        stopComposeAction();
        toastRes(res);
    });
}

function stopStack() {
    startComposeAction();

    emitAgent(endpoint.value, "stopStack", stack.name, (res: any) => {
        stopComposeAction();
        toastRes(res);
    });
}

function downStack() {
    startComposeAction();

    emitAgent(endpoint.value, "downStack", stack.name, (res: any) => {
        stopComposeAction();
        toastRes(res);
    });
}

function restartStack() {
    startComposeAction();

    emitAgent(endpoint.value, "restartStack", stack.name, (res: any) => {
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

    emitAgent(endpoint.value, "updateStack", stack.name, updateDialogData.pruneAfterUpdate, updateDialogData.pruneAllAfterUpdate, (res: any) => {
        stopComposeAction();
        toastRes(res);
    });
}

function deleteDialog() {
    emitAgent(endpoint.value, "deleteStack", stack.name, { deleteStackFiles: deleteStackFiles.value }, (res: any) => {
        toastRes(res);
        if (res.ok) {
            router.push("/stacks");
        } else {
            errorDelete.value = true;
        }
    });
}

function forceDeleteDialog() {
    emitAgent(endpoint.value, "forceDeleteStack", stack.name, (res: any) => {
        toastRes(res);
        if (res.ok) {
            router.push("/stacks");
        }
    });
}

function discardStack() {
    loadStack();
    isEditMode.value = false;
}

function yamlToJSON(yamlStr: string) {
    const doc = parseDocument(yamlStr);
    if (doc.errors.length > 0) {
        throw doc.errors[0];
    }

    const config = doc.toJS() ?? {};

    if (!config.services) {
        config.services = {};
    }

    if (Array.isArray(config.services) || typeof config.services !== "object") {
        throw new Error("Services must be an object");
    }

    return {
        config,
        doc,
    };
}

function yamlCodeChange() {
    try {
        const { config, doc } = yamlToJSON(stack.composeYAML);

        yamlDoc = doc;
        Object.keys(jsonConfig).forEach(key => delete jsonConfig[key]);
        Object.assign(jsonConfig, config);

        const env = dotenv.parse(stack.composeENV);
        const envYAMLStr = envsubstYAML(stack.composeYAML, env);
        const envConfig = yamlToJSON(envYAMLStr).config;
        Object.keys(envsubstJSONConfig).forEach(key => delete envsubstJSONConfig[key]);
        Object.assign(envsubstJSONConfig, envConfig);

        if (yamlErrorTimeout) {
            clearTimeout(yamlErrorTimeout);
        }
        yamlError.value = "";
    } catch (e: any) {
        if (yamlErrorTimeout) {
            clearTimeout(yamlErrorTimeout);
        }

        if (yamlError.value) {
            yamlError.value = e.message;
        } else {
            yamlErrorTimeout = setTimeout(() => {
                yamlError.value = e.message;
            }, 3000);
        }
    }
}

function enableEditMode() {
    if (document.activeElement instanceof HTMLElement) {
        document.activeElement.blur();
    }
    isEditMode.value = true;
}

function checkImageUpdates() {
    processing.value = true;

    emitAgent(endpoint.value, "checkImageUpdates", stack.name, (res: any) => {
        processing.value = false;
        toastRes(res);

        if (res.ok) {
            requestServiceStatus();
        }
    });
}

// Initialize
onMounted(() => {
    if (stackName.value) {
        stack.name = stackName.value;
        loadStack();
        requestServiceStatus();
    }
});
</script>

<style scoped lang="scss">
@import "../styles/vars.scss";

.editor-box {
    font-family: 'JetBrains Mono', monospace;
    font-size: 14px;
    &.edit-mode {
        background-color: #2c2f38 !important;
    }
    position: relative;
}

.expand-button {
    all: unset;
    position: absolute;
    right: 15px;
    top: 15px;
    z-index: 10;
}

.expand-button svg {
    width:20px;
    height: 20px;
}

.expand-button:hover {
    color: white;
}

.agent-name {
    font-size: 13px;
    color: $dark-font-color3;
}

.btn-check:active + .btn-outline-primary,
.btn-check:checked + .btn-outline-primary {
    color: #fff;

    .dark & {
        color: #000;
    }
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
