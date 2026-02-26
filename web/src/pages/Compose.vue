<template>
    <transition name="slide-fade" appear>
        <div>
            <h1 v-if="isAdd" class="mb-3">{{ $t("compose") }}</h1>
            <h1 v-else class="mb-3">
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

                        <UpdateDialog
                            v-model="showUpdateDialog"
                            :stack-name="stack.name"
                            :endpoint="endpoint"
                            :compose-yaml="stack.composeYAML"
                            @update="doUpdateStack"
                        />

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

                    <button v-if="isEditMode && !isAdd" class="btn btn-normal" :disabled="processing" :title="$t('tooltipStackDiscard')" @click="discardStack">{{ $t("discardStack") }}</button>
                </div>

                <!-- Parsed / Raw toggle -->
                <div v-if="!isAdd" class="btn-group" role="group">
                    <input
                        id="view-parsed"
                        v-model="viewMode"
                        type="radio"
                        class="btn-check"
                        name="viewMode"
                        autocomplete="off"
                        value="parsed"
                    />
                    <label class="btn btn-outline-primary" for="view-parsed">
                        {{ $t("parsed") }}
                    </label>

                    <input
                        id="view-raw"
                        v-model="viewMode"
                        type="radio"
                        class="btn-check"
                        name="viewMode"
                        autocomplete="off"
                        value="raw"
                    />
                    <label class="btn btn-outline-primary" for="view-raw">
                        {{ $t("raw") }}
                    </label>
                </div>
            </div>

            <!-- URLs -->
            <div v-if="urls.length > 0" class="mb-3">
                <a v-for="(url, index) in urls" :key="index" target="_blank" :href="url.url">
                    <span class="badge bg-secondary me-2">{{ url.display }}</span>
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

            <div v-if="stack.isManagedByDockge" class="row">
                <div class="col-lg-6">
                    <!-- General -->
                    <div v-if="isAdd">
                        <h4 class="mb-3">{{ $t("general") }}</h4>
                        <div class="shadow-box big-padding mb-3">
                            <!-- Stack Name -->
                            <div>
                                <label for="name" class="form-label">{{ $t("stackName") }}</label>
                                <input id="name" v-model="stack.name" type="text" class="form-control" required @blur="stackNameToLowercase">
                                <div class="form-text">{{ $t("Lowercase only") }}</div>
                            </div>

                            <!-- Endpoint -->
                            <div class="mt-3">
                                <label for="name" class="form-label">{{ $t("dockgeAgent") }}</label>
                                <select v-model="stack.endpoint" class="form-select">
                                    <option v-for="(agent, ep) in agentList" :key="ep" :value="ep" :disabled="agentStatusList[ep] != 'online'">
                                        ({{ agentStatusList[ep] }}) {{ (agent.name !== '') ? agent.name : agent.url || $t("Controller") }}
                                    </option>
                                </select>
                            </div>
                        </div>
                    </div>

                    <!-- Containers -->
                    <CollapsibleSection>
                        <template #heading>{{ $tc("container", 2) }} <span class="section-count">({{ Object.keys(jsonConfig.services || {}).length }})</span></template>

                        <div v-if="isEditMode" class="input-group mb-3">
                            <input
                                v-model="newContainerName"
                                :placeholder="$t(`New Container Name...`)"
                                class="form-control"
                                @keyup.enter="addContainer"
                            />
                            <button class="btn btn-primary" @click="addContainer">
                                {{ $t("addContainer") }}
                            </button>
                        </div>

                        <div ref="containerListRef">
                            <template v-for="(service, name, index) in jsonConfig.services" :key="name">
                                <Container
                                    v-if="index < renderedCount"
                                    :name="name"
                                    :is-edit-mode="isEditMode"
                                    :first="index === 0"
                                    :serviceStatus="serviceStatusList[name]"
                                    :serviceImageUpdateAvailable="serviceUpdateStatus[name] || false"
                                    :serviceRecreateNecessary="serviceRecreateStatus[name] || false"
                                    :processing="processing"
                                    @start-service="startService"
                                    @stop-service="stopService"
                                    @restart-service="restartService"
                                    @update-service="updateService"
                                    @scroll-to-service="scrollToService"
                                />
                            </template>
                        </div>

                        <button v-if="false && isEditMode && jsonConfig.services && Object.keys(jsonConfig.services).length > 0" class="btn btn-normal mb-3" @click="addContainer">{{ $t("addContainer") }}</button>
                    </CollapsibleSection>

                    <!-- Combined Terminal Output -->
                    <div v-show="!isEditMode">
                        <h4 class="mb-3">{{ $t("logs") }}</h4>
                        <Terminal
                            ref="combinedTerminal"
                            class="mb-3 terminal"
                            :name="combinedTerminalName"
                            :endpoint="endpoint"
                            :rows="combinedTerminalRows"
                            :cols="combinedTerminalCols"
                            style="height: 315px;"
                        ></Terminal>
                    </div>
                </div>
                <div class="col-lg-6">
                    <!-- Override YAML editor (only show if file exists) -->
                    <div v-if="stack.composeOverrideYAML && stack.composeOverrideYAML.trim() !== ''">
                    <h4 class="mb-3">{{ stack.composeOverrideFileName || 'compose.override.yaml' }}</h4>
                    <div class="shadow-box mb-3 editor-box" :class="{'edit-mode' : isEditMode}">
                        <div class="editor-buttons">
                            <button class="editor-btn" :title="wordWrap ? 'Disable word wrap' : 'Enable word wrap'" @click="wordWrap = !wordWrap">
                                <font-awesome-icon :icon="wordWrap ? 'arrow-turn-down' : 'arrow-right'" />
                            </button>
                            <button v-if="isEditMode" v-b-modal.compose-override-editor-modal class="editor-btn">
                                <font-awesome-icon icon="expand" />
                            </button>
                        </div>
                        <code-mirror
                            ref="overrideEditor"
                            v-model="stack.composeOverrideYAML"
                            :extensions="extensions"
                            minimal
                            :wrap="wordWrap"
                            :dark="isDark"
                            :tab="true"
                            :disabled="!isEditMode"
                            :hasFocus="editorFocus"
                            @change="yamlCodeChange"
                        />
                    </div>
                    <div v-if="isEditMode" class="mb-3">
                        {{ yamlError }}
                    </div>

                    <!-- Override modal fullscreen editor (CodeMirror) -->
                    <BModal id="compose-override-editor-modal" :title="stack.composeOverrideFileName || 'compose.override.yaml'"
scrollable size="fullscreen" hide-footer>
                        <div class="shadow-box mb-3 editor-box" :class="{'edit-mode' : isEditMode}">
                            <code-mirror
                                ref="editorModal"
                                v-model="stack.composeOverrideYAML"
                                :extensions="extensions"
                                minimal
                                :wrap="wordWrap"
                                :dark="isDark"
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
                        <div class="editor-buttons">
                            <button class="editor-btn" :title="wordWrap ? 'Disable word wrap' : 'Enable word wrap'" @click="wordWrap = !wordWrap">
                                <font-awesome-icon :icon="wordWrap ? 'arrow-turn-down' : 'arrow-right'" />
                            </button>
                            <button v-if="isEditMode" v-b-modal.compose-editor-modal class="editor-btn">
                                <font-awesome-icon icon="expand" />
                            </button>
                        </div>
                        <code-mirror
                            ref="editorInline"
                            v-model="stack.composeYAML"
                            :extensions="extensions"
                            minimal
                            :wrap="wordWrap"
                            :dark="isDark"
                            :tab="true"
                            :disabled="!isEditMode"
                            :hasFocus="editorFocus"
                            @change="yamlCodeChange"
                        />
                    </div>
                    <div v-if="isEditMode" class="mb-3">
                        {{ yamlError }}
                    </div>

                    <!-- YAML modal fullscreen editor (CodeMirror) -->
                    <BModal id="compose-editor-modal" :title="stack.composeFileName" scrollable size="fullscreen" hide-footer>
                        <div class="shadow-box mb-3 editor-box" :class="{'edit-mode' : isEditMode}">
                            <code-mirror
                                ref="editorModal"
                                v-model="stack.composeYAML"
                                :extensions="extensions"
                                minimal
                                :wrap="wordWrap"
                                :dark="isDark"
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
                            <div class="editor-buttons">
                                <button class="editor-btn" :title="wordWrap ? 'Disable word wrap' : 'Enable word wrap'" @click="wordWrap = !wordWrap">
                                    <font-awesome-icon :icon="wordWrap ? 'arrow-turn-down' : 'arrow-right'" />
                                </button>
                                <button v-if="isEditMode" v-b-modal.env-editor-modal class="editor-btn">
                                    <font-awesome-icon icon="expand" />
                                </button>
                            </div>
                            <code-mirror
                                ref="editorEnv"
                                v-model="stack.composeENV"
                                :extensions="extensionsEnv"
                                minimal
                                :wrap="wordWrap"
                                :dark="isDark"
                                :tab="true"
                                :disabled="!isEditMode"
                                :hasFocus="editorFocus"
                                @change="yamlCodeChange"
                            />
                        </div>
                    </div>

                    <!-- ENV modal fullscreen editor (CodeMirror) -->
                    <BModal id="env-editor-modal" title=".env" scrollable size="fullscreen" hide-footer>
                        <div class="shadow-box mb-3 editor-box" :class="{'edit-mode' : isEditMode}">
                            <code-mirror
                                ref="editorEnvModal"
                                v-model="stack.composeENV"
                                :extensions="extensionsEnv"
                                minimal
                                :wrap="wordWrap"
                                :dark="isDark"
                                :tab="true"
                                :disabled="!isEditMode"
                                :hasFocus="editorFocus"
                                @change="yamlCodeChange"
                            />
                        </div>
                    </BModal>

                    <div v-if="isEditMode">
                        <!-- Volumes -->
                        <div v-if="false">
                            <h4 class="mb-3">{{ $tc("volume", 2) }}</h4>
                            <div class="shadow-box big-padding mb-3">
                            </div>
                        </div>

                        <!-- Networks -->
                        <CollapsibleSection>
                            <template #heading>{{ $tc("network", 2) }} <span class="section-count">({{ Object.keys(networks || {}).length }})</span></template>
                            <div class="shadow-box big-padding mb-3">
                                <NetworkInput />
                            </div>
                        </CollapsibleSection>
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
    </transition>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch, provide, onMounted, onUnmounted, nextTick } from "vue";
import { useRoute, useRouter, onBeforeRouteUpdate, onBeforeRouteLeave } from "vue-router";
import { useI18n } from "vue-i18n";
import CodeMirror from "vue-codemirror6";
import { yaml } from "@codemirror/lang-yaml";
import { python } from "@codemirror/lang-python";
import { tomorrowNightEighties as editorTheme } from "../editor-theme";
import { lineNumbers, EditorView } from "@codemirror/view";
import { indentUnit, indentService } from "@codemirror/language";
import { parseDocument, Document } from "yaml";
import { FontAwesomeIcon } from "@fortawesome/vue-fontawesome";
import {
    COMBINED_TERMINAL_COLS,
    COMBINED_TERMINAL_ROWS,
    copyYAMLComments, envsubstYAML, parseEnv,
    getCombinedTerminalName,
    getComposeTerminalName,
    PROGRESS_TERMINAL_ROWS,
} from "../common/util-common";
import { BModal } from "bootstrap-vue-next";
import { LABEL_URLS_PREFIX } from "../common/compose-labels";
import NetworkInput from "../components/NetworkInput.vue";
import ProgressTerminal from "../components/ProgressTerminal.vue";
import UpdateDialog from "../components/UpdateDialog.vue";
import { useSocket } from "../composables/useSocket";
import { useAppToast } from "../composables/useAppToast";
import { useTheme } from "../composables/useTheme";

const route = useRoute();
const router = useRouter();
const { t } = useI18n();
const { isDark } = useTheme();
const { emitAgent, agentCount, agentList, agentStatusList, containerList, completeStackList, composeTemplate, envTemplate, endpointDisplayFunction, info } = useSocket();
const { toastRes, toastError } = useAppToast();

// Suppress jsonConfig → YAML sync during programmatic updates (e.g. loadStack)
let skipConfigSync = false;

// CodeMirror setup
const editorInline = ref<InstanceType<typeof CodeMirror>>();
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

// Templates
const defaultTemplate = `
services:
  nginx:
    image: nginx:latest
    restart: unless-stopped
    ports:
      - "8080:80"
`;
const envDefault = "# VARIABLE=value #comment";

// Timeouts
let yamlErrorTimeout: ReturnType<typeof setTimeout> | null = null;
let serviceStatusTimeout: ReturnType<typeof setTimeout> | null = null;
let dockerStatsTimeout: ReturnType<typeof setTimeout> | null = null;

// YAML document for comment preservation
let yamlDoc: any = null;

// Template refs
const progressTerminalRef = ref<InstanceType<typeof ProgressTerminal>>();
const containerListRef = ref<HTMLElement>();

// Data
const jsonConfig = reactive<Record<string, any>>({});
const envsubstJSONConfig = reactive<Record<string, any>>({});
const yamlError = ref("");
const processing = ref(true);
const progressTerminalRows = PROGRESS_TERMINAL_ROWS;
const combinedTerminalRows = COMBINED_TERMINAL_ROWS;
const combinedTerminalCols = COMBINED_TERMINAL_COLS;
const stack = reactive<Record<string, any>>({
    composeOverrideYAML: "",
});
const serviceStatusList = ref<Record<string, any>>({});
const serviceUpdateStatus = ref<Record<string, any>>({});
const serviceRecreateStatus = ref<Record<string, any>>({});
const dockerStats = ref<Record<string, any>>({});
const isEditMode = ref(false);
const errorDelete = ref(false);
const submitted = ref(false);
const showDeleteDialog = ref(false);
const deleteStackFiles = ref(false);
const showForceDeleteDialog = ref(false);
const showUpdateDialog = ref(false);
const newContainerName = ref("");
const viewMode = ref<"parsed" | "raw">("parsed");
const wordWrap = ref(localStorage.getItem("editorWordWrap") !== "false");
const stopServiceStatusTimeout = ref(false);
const stopDockerStatsTimeout = ref(false);

// Progressive rendering: render containers in batches to avoid blocking the main thread
const RENDER_BATCH_SIZE = 20;
const renderedCount = ref(Infinity); // Infinity = render all (for small stacks / edit mode)
let renderRAF = 0;

function scheduleProgressiveRender(total: number) {
    cancelAnimationFrame(renderRAF);
    if (total <= RENDER_BATCH_SIZE) {
        renderedCount.value = Infinity;
        return;
    }
    renderedCount.value = RENDER_BATCH_SIZE;
    function renderBatch() {
        renderedCount.value += RENDER_BATCH_SIZE;
        if (renderedCount.value < total) {
            renderRAF = requestAnimationFrame(renderBatch);
        } else {
            renderedCount.value = Infinity;
        }
    }
    renderRAF = requestAnimationFrame(renderBatch);
}

// Provide to children (Container, NetworkInput)
provide("jsonConfig", jsonConfig);
provide("envsubstJSONConfig", envsubstJSONConfig);
provide("composeStack", stack);
provide("composeEndpoint", computed(() => endpoint.value));
provide("editorFocus", editorFocus);
provide("startComposeAction", startComposeAction);
provide("stopComposeAction", stopComposeAction);

// Computed
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

const isAdd = computed(() => route.path === "/stacks/new" && !submitted.value);

const globalStack = computed(() => completeStackList.value[stack.name + "_" + endpoint.value] ?? null);

const active = computed(() => globalStack.value?.started ?? false);

const terminalName = computed(() => {
    if (!stack.name) {
        return "";
    }
    return getComposeTerminalName(endpoint.value, stack.name);
});

const combinedTerminalName = computed(() => {
    if (!stack.name) {
        return "";
    }
    return getCombinedTerminalName(endpoint.value, stack.name);
});

const networks = computed(() => jsonConfig.networks);

const endpoint = computed(() => stack.endpoint || (route.params.endpoint as string) || "");

const url = computed(() => {
    if (stack.endpoint) {
        return `/stacks/${stack.name}/${stack.endpoint}`;
    }
    return `/stacks/${stack.name}`;
});

// Watchers
watch(() => stack.composeYAML, () => {
    if (editorFocus.value) {
        console.debug("yaml code changed");
        yamlCodeChange();
    }
});

watch(() => stack.composeENV, () => {
    if (editorFocus.value) {
        console.debug("env code changed");
        yamlCodeChange();
    }
});

watch(() => stack.composeOverrideYAML, () => {
    if (editorFocus.value) {
        console.debug("override yaml code changed");
        yamlCodeChange();
    }
});

watch(jsonConfig, () => {
    if (skipConfigSync) return;
    if (!editorFocus.value) {
        console.debug("jsonConfig changed");

        const doc = new Document(jsonConfig);

        // Stick back the yaml comments
        if (yamlDoc) {
            copyYAMLComments(doc, yamlDoc);
        }

        stack.composeYAML = doc.toString();
        yamlDoc = doc;
    }
}, { deep: true });

// Refresh service status when the backend broadcasts an updated container list
// (e.g. after compose restart/stop/start actions or Docker events).
watch(containerList, () => {
    if (!isAdd.value && stack.name) {
        requestServiceStatus();
    }
});

watch(wordWrap, (v) => {
    localStorage.setItem("editorWordWrap", v ? "true" : "false");
});

// Navigate to raw view when toggle changes
watch(viewMode, (mode) => {
    if (mode === "raw" && stack.name) {
        const rawUrl = stack.endpoint
            ? `/stacks/${stack.name}/raw/${stack.endpoint}`
            : `/stacks/${stack.name}/raw`;
        router.push(rawUrl);
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

function startDockerStatsTimeout() {
    clearTimeout(dockerStatsTimeout!);
    dockerStatsTimeout = setTimeout(async () => {
        requestDockerStats();
    }, 5000);
}

function requestServiceStatus() {
    if (isAdd.value) {
        return;
    }

    emitAgent(endpoint.value, "serviceStatusList", stack.name, (res: any) => {
        if (res.ok) {
            serviceStatusList.value = res.serviceStatusList;
            serviceUpdateStatus.value = res.serviceUpdateStatus || {};
            serviceRecreateStatus.value = res.serviceRecreateStatus || {};
            stack.imageUpdatesAvailable = Object.values(serviceUpdateStatus.value).some((v: any) => v === true);
        }
        if (!stopServiceStatusTimeout.value) {
            startServiceStatusTimeout();
        }
    });
}

function requestDockerStats() {
    emitAgent(endpoint.value, "dockerStats", stack.name, (res: any) => {
        if (res.ok) {
            dockerStats.value = res.dockerStats;
        }
        if (!stopDockerStatsTimeout.value) {
            startDockerStatsTimeout();
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
    console.log("exitAction");
    stopServiceStatusTimeout.value = true;
    stopDockerStatsTimeout.value = true;
    clearTimeout(serviceStatusTimeout!);
    clearTimeout(dockerStatsTimeout!);

    console.debug("leaveCombinedTerminal", endpoint.value, stack.name);
    emitAgent(endpoint.value, "leaveCombinedTerminal", stack.name, () => {});
}

function bindTerminal() {
    // ProgressTerminal handles binding internally via show()
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
            // Suppress the jsonConfig → YAML deep watcher during initial load.
            // yamlCodeChange() sets jsonConfig from YAML; the watcher would
            // redundantly convert it back (with expensive copyYAMLComments).
            skipConfigSync = true;
            Object.assign(stack, res.stack);
            yamlCodeChange();
            // Progressive rendering: render first batch immediately, then
            // schedule remaining batches via requestAnimationFrame so the
            // browser can paint between batches instead of blocking for 500ms+.
            const serviceCount = jsonConfig.services ? Object.keys(jsonConfig.services).length : 0;
            scheduleProgressiveRender(serviceCount);
            processing.value = false;
            bindTerminal();
            nextTick(() => { skipConfigSync = false; });
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

    const serviceNameList = Object.keys(jsonConfig.services);

    if (!stack.name && serviceNameList.length > 0) {
        const serviceName = serviceNameList[0];
        const service = jsonConfig.services[serviceName];

        if (service && service.container_name) {
            stack.name = service.container_name;
        } else {
            stack.name = serviceName;
        }
    }

    startComposeAction();
    submitted.value = true;

    emitAgent(stack.endpoint, "deployStack", stack.name, stack.composeYAML, stack.composeENV, stack.composeOverrideYAML || "", isAdd.value, (res: any) => {
        stopComposeAction();
        toastRes(res);

        if (res.ok) {
            isEditMode.value = false;
            router.push(url.value);
        }
    });
}

function saveStack() {
    processing.value = true;

    emitAgent(stack.endpoint, "saveStack", stack.name, stack.composeYAML, stack.composeENV, stack.composeOverrideYAML || "", isAdd.value, (res: any) => {
        processing.value = false;
        toastRes(res);

        if (res.ok) {
            isEditMode.value = false;
            router.push(url.value);
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

function doUpdateStack(data: { pruneAfterUpdate: boolean; pruneAllAfterUpdate: boolean }) {
    startComposeAction();

    emitAgent(endpoint.value, "updateStack", stack.name, data.pruneAfterUpdate, data.pruneAllAfterUpdate, (res: any) => {
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

        const env = parseEnv(stack.composeENV);
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
    renderedCount.value = Infinity; // Show all containers when editing
    isEditMode.value = true;
}

function checkYAML() {
}

function addContainer() {
    checkYAML();

    if (jsonConfig.services[newContainerName.value]) {
        toastError("Container name already exists");
        return;
    }

    if (!newContainerName.value) {
        toastError("Container name cannot be empty");
        return;
    }

    jsonConfig.services[newContainerName.value] = {
        restart: "unless-stopped",
    };
    newContainerName.value = "";
    const element = containerListRef.value?.lastElementChild;
    element?.scrollIntoView({
        block: "start",
        behavior: "smooth"
    });
}

function stackNameToLowercase() {
    stack.name = stack.name?.toLowerCase();
}

function startService(serviceName: string) {
    startComposeAction();

    emitAgent(endpoint.value, "startService", stack.name, serviceName, (res: any) => {
        stopComposeAction();
        toastRes(res);

        if (res.ok) {
            requestServiceStatus();
        }
    });
}

function stopService(serviceName: string) {
    startComposeAction();

    emitAgent(endpoint.value, "stopService", stack.name, serviceName, (res: any) => {
        stopComposeAction();
        toastRes(res);

        if (res.ok) {
            requestServiceStatus();
        }
    });
}

function restartService(serviceName: string) {
    startComposeAction();

    emitAgent(endpoint.value, "restartService", stack.name, serviceName, (res: any) => {
        stopComposeAction();
        toastRes(res);

        if (res.ok) {
            requestServiceStatus();
        }
    });
}

function scrollToService(serviceName: string) {
    const view = editorInline.value?.view;
    if (!view) return;

    // Find the service name as a top-level key under "services:" in the YAML
    const doc = view.state.doc;
    const pattern = new RegExp(`^  ${serviceName.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}\\s*:`);
    for (let i = 1; i <= doc.lines; i++) {
        const line = doc.line(i);
        if (pattern.test(line.text)) {
            // Select the service name within the line
            const nameStart = line.from + line.text.indexOf(serviceName);
            const nameEnd = nameStart + serviceName.length;
            view.dispatch({
                selection: { anchor: nameStart, head: nameEnd },
                effects: EditorView.scrollIntoView(nameStart, { y: "center" }),
            });
            view.focus();
            return;
        }
    }
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

function updateService(serviceName: string) {
    startComposeAction();

    emitAgent(endpoint.value, "updateService", stack.name, serviceName, (res: any) => {
        stopComposeAction();
        toastRes(res);

        if (res.ok) {
            requestServiceStatus();
        }
    });
}

// Initialize
onMounted(() => {
    if (isAdd.value) {
        processing.value = false;
        isEditMode.value = true;

        let composeYAML;
        let composeENV;

        if (composeTemplate.value) {
            composeYAML = composeTemplate.value;
            composeTemplate.value = "";
        } else {
            composeYAML = defaultTemplate;
        }
        if (envTemplate.value) {
            composeENV = envTemplate.value;
            envTemplate.value = "";
        } else {
            composeENV = envDefault;
        }

        Object.assign(stack, {
            name: "",
            composeYAML,
            composeENV,
            isManagedByDockge: true,
            endpoint: "",
        });

        yamlCodeChange();

    } else {
        stack.name = route.params.stackName as string;
        loadStack();
    }

    requestServiceStatus();
    requestDockerStats();
});

onUnmounted(() => {
    cancelAnimationFrame(renderRAF);
});
</script>

<style scoped lang="scss">
@import "../styles/vars.scss";

.terminal {
    height: 200px;
}

.editor-box {
    font-family: 'JetBrains Mono', monospace;
    font-size: 14px;
    position: relative;

    :deep(.cm-gutters) {
        background-color: white !important;
    }

    .dark & :deep(.cm-gutters) {
        background-color: $dark-bg !important;
    }
}

.editor-buttons {
    position: absolute;
    right: 15px;
    top: 15px;
    z-index: 10;
    display: flex;
    gap: 10px;
}

.editor-btn {
    all: unset;
    cursor: pointer;
    color: $dark-font-color3;

    svg {
        width: 20px;
        height: 20px;
    }

    &:hover {
        color: white;
    }
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

</style>
