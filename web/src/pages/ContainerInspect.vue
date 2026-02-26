<template>
    <transition name="slide-fade" appear>
        <div v-if="containerName">
            <h1 class="mb-3"><span v-if="badgeLabel" :class="badgeClass">{{ badgeLabel }}</span> {{ containerName }}</h1>

            <div class="d-flex align-items-center justify-content-between mb-3">
                <div v-if="stackName && stackManaged" class="d-flex align-items-center">
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
                <div v-else></div>

                <!-- Parsed / Raw toggle -->
                <div class="btn-group" role="group">
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

            <!-- Progress Terminal -->
            <ProgressTerminal
                ref="progressTerminalRef"
                class="mb-3"
                :name="terminalName"
                :endpoint="endpoint"
            />

            <!-- Metrics Card -->
            <div class="shadow-box big-padding text-center mb-3">
                <div class="row g-3">
                    <div class="col">
                        <div class="metric-cell">
                            <div class="metric-label">{{ $t('CPU') }}</div>
                            <span class="num" :class="{ 'placeholder-value': !containerStat }">{{ containerStat?.CPUPerc ?? '--' }}</span>
                        </div>
                    </div>
                    <div class="col">
                        <div class="metric-cell">
                            <div class="metric-label">{{ $t('memory') }}</div>
                            <span class="num" :class="{ 'placeholder-value': !containerStat }">{{ containerStat?.MemUsage ?? '--' }}</span>
                            <span class="num-sub">({{ containerStat?.MemPerc ?? '--' }})</span>
                        </div>
                    </div>
                    <div class="col">
                        <div class="metric-cell">
                            <div class="metric-label">{{ $t('blockIO') }}</div>
                            <span class="num" :class="{ 'placeholder-value': !containerStat }">{{ containerStat?.BlockIO ?? '--' }}</span>
                            <span class="num-sub">read / write</span>
                        </div>
                    </div>
                    <div class="col">
                        <div class="metric-cell">
                            <div class="metric-label">{{ $t('networkIO') }}</div>
                            <span class="num" :class="{ 'placeholder-value': !containerStat }">{{ containerStat?.NetIO ?? '--' }}</span>
                            <span class="num-sub">rx / tx</span>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Parsed View: two-column layout -->
            <div v-if="viewMode === 'parsed'" class="row">
                <div class="col-lg-8">
                    <!-- Networks Card -->
                    <CollapsibleSection>
                        <template #heading>{{ $t("containerNetworks") }} ({{ networks.length }})</template>
                        <div v-if="networks.length > 0">
                            <div v-for="net in networks" :key="net.name" class="shadow-box big-padding mb-3">
                                <h5 class="mb-3">
                                    <router-link :to="{ name: 'networkDetail', params: { networkName: net.name } }" class="stack-link"><font-awesome-icon icon="network-wired" class="me-2" />{{ net.name }}</router-link>
                                </h5>
                                <div class="inspect-grid">
                                    <div class="inspect-label">{{ $t("networkIPv4") }}</div>
                                    <div class="inspect-value"><code>{{ net.ipv4 || '–' }}</code></div>

                                    <div class="inspect-label">{{ $t("networkIPv6") }}</div>
                                    <div class="inspect-value"><code>{{ net.ipv6 || '–' }}</code></div>

                                    <div class="inspect-label">{{ $t("networkMAC") }}</div>
                                    <div class="inspect-value"><code>{{ net.mac || '–' }}</code></div>

                                    <div class="inspect-label">{{ $t("networkGateway") }}</div>
                                    <div class="inspect-value"><code>{{ net.gateway || '–' }}</code></div>

                                    <template v-if="net.aliases && net.aliases.length > 0">
                                        <div class="inspect-label">{{ $t("networkAliases") }}</div>
                                        <div class="inspect-value">
                                            <template v-for="(alias, i) in net.aliases" :key="i">
                                                <code>{{ alias }}</code><template v-if="i < net.aliases.length - 1">, </template>
                                            </template>
                                        </div>
                                    </template>
                                </div>
                            </div>
                        </div>
                        <div v-else class="shadow-box big-padding mb-3">
                            <p class="text-muted mb-0">{{ $t("noNetworks") }}</p>
                        </div>
                    </CollapsibleSection>

                    <!-- Mounts Card -->
                    <CollapsibleSection>
                        <template #heading>{{ $t("containerMounts") }} ({{ mounts.length }})</template>
                        <div v-if="mounts.length > 0">
                            <div v-for="(mount, idx) in mounts" :key="idx" class="shadow-box big-padding mb-3">
                                <div class="inspect-grid">
                                    <div class="inspect-label">{{ $t("mountType") }}</div>
                                    <div class="inspect-value">{{ mount.Type }}</div>

                                    <template v-if="mount.Type === 'volume' && mount.Name">
                                        <div class="inspect-label">{{ $t("mountVolume") }}</div>
                                        <div class="inspect-value">
                                            <router-link :to="{ name: 'volumeDetail', params: { volumeName: mount.Name } }" class="stack-link"><font-awesome-icon icon="hard-drive" class="me-2" />{{ mount.Name }}</router-link>
                                        </div>
                                    </template>

                                    <div class="inspect-label">{{ $t("mountSource") }}</div>
                                    <div class="inspect-value"><code>{{ mount.Source || mount.Name || '–' }}</code></div>

                                    <div class="inspect-label">{{ $t("mountDestination") }}</div>
                                    <div class="inspect-value"><code>{{ mount.Destination }}</code></div>

                                    <div class="inspect-label">{{ $t("mountReadWrite") }}</div>
                                    <div class="inspect-value">{{ mount.RW ? 'rw' : 'ro' }}</div>
                                </div>
                            </div>
                        </div>
                        <div v-else class="shadow-box big-padding mb-3">
                            <p class="text-muted mb-0">{{ $t("noMounts") }}</p>
                        </div>
                    </CollapsibleSection>

                    <!-- Processes Card -->
                    <CollapsibleSection>
                        <template #heading>{{ $t("containerProcesses") }} ({{ processList.length }})</template>
                        <div class="shadow-box big-padding mb-3">
                        <div v-if="processList.length > 0" class="table-responsive">
                            <table class="table table-sm mb-0 process-table">
                                <thead>
                                    <tr>
                                        <th>{{ $t("processPID") }}</th>
                                        <th>{{ $t("processUser") }}</th>
                                        <th>{{ $t("processCommand") }}</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    <tr v-for="(proc, idx) in processList" :key="idx">
                                        <td>{{ proc.pid }}</td>
                                        <td>{{ proc.user }}</td>
                                        <td><code>{{ proc.command }}</code></td>
                                    </tr>
                                </tbody>
                            </table>
                        </div>
                        <p v-else class="text-muted mb-0">{{ $t("noProcesses") }}</p>
                    </div>
                    </CollapsibleSection>
                </div>

                <div class="col-lg-4">
                    <!-- Overview Card -->
                    <h4 class="mb-3">{{ $t("containerOverview") }}</h4>
                    <div v-if="parsed" class="shadow-box big-padding mb-3">
                        <div class="overview-list">
                            <!-- Stack -->
                            <div v-if="stackName" class="overview-item">
                                <div class="overview-label">{{ $t("containerStack") }}</div>
                                <div class="overview-value">
                                    <router-link :to="stackLink" class="stack-link"><font-awesome-icon icon="layer-group" class="me-2" />{{ stackName }}</router-link>
                                </div>
                            </div>

                            <!-- Image -->
                            <div v-if="parsed.Config?.Image" class="overview-item">
                                <div class="overview-label">{{ $t("containerImage") }}</div>
                                <div class="overview-value">
                                    <router-link :to="{ name: 'imageDetail', params: { imageRef: fullImageRef } }" class="stack-link">
                                        <font-awesome-icon icon="box-archive" class="me-2" />{{ fullImageRef }}
                                    </router-link>
                                </div>
                            </div>

                            <!-- Command -->
                            <div v-if="commandStr" class="overview-item">
                                <div class="overview-label">{{ $t("containerCommand") }}</div>
                                <div class="overview-value"><code>{{ commandStr }}</code></div>
                            </div>

                            <!-- Restart Policy -->
                            <div v-if="restartPolicyStr" class="overview-item">
                                <div class="overview-label">{{ $t("containerRestartPolicy") }}</div>
                                <div class="overview-value">{{ restartPolicyStr }}</div>
                            </div>

                            <!-- Restart Count -->
                            <div v-if="parsed.RestartCount != null" class="overview-item">
                                <div class="overview-label">{{ $t("containerRestartCount") }}</div>
                                <div class="overview-value">{{ parsed.RestartCount }}</div>
                            </div>

                            <!-- Container ID -->
                            <div v-if="parsed.Id" class="overview-item">
                                <div class="overview-label">{{ $t("containerID") }}</div>
                                <div class="overview-value">
                                    <code :title="parsed.Id">{{ parsed.Id.substring(0, 12) }}</code>
                                </div>
                            </div>

                            <!-- Created -->
                            <div v-if="parsed.Created" class="overview-item">
                                <div class="overview-label">{{ $t("containerCreated") }}</div>
                                <div class="overview-value">{{ formatDate(parsed.Created) }}</div>
                            </div>

                            <!-- Started -->
                            <div v-if="parsed.State?.StartedAt && isValidDate(parsed.State.StartedAt)" class="overview-item">
                                <div class="overview-label">{{ $t("containerStarted") }}</div>
                                <div class="overview-value">{{ formatDate(parsed.State.StartedAt) }}</div>
                            </div>

                            <!-- Uptime -->
                            <div v-if="uptimeStr" class="overview-item">
                                <div class="overview-label">{{ $t("containerUptime") }}</div>
                                <div class="overview-value">{{ uptimeStr }}</div>
                            </div>

                            <!-- Working Dir -->
                            <div v-if="parsed.Config?.WorkingDir !== undefined" class="overview-item">
                                <div class="overview-label">{{ $t("containerWorkingDir") }}</div>
                                <div class="overview-value">
                                    <code v-if="parsed.Config.WorkingDir">{{ parsed.Config.WorkingDir }}</code>
                                    <span v-else class="text-muted">&ndash;</span>
                                </div>
                            </div>

                            <!-- User -->
                            <div v-if="parsed.Config?.User !== undefined" class="overview-item">
                                <div class="overview-label">{{ $t("containerUserGroup") }}</div>
                                <div class="overview-value">
                                    <span v-if="parsed.Config.User">{{ parsed.Config.User }}</span>
                                    <span v-else class="text-muted">&ndash;</span>
                                </div>
                            </div>
                        </div>
                    </div>
                    <div v-else class="shadow-box big-padding mb-3">
                        <p class="text-muted mb-0">{{ inspectData }}</p>
                    </div>
                </div>
            </div>

            <!-- Raw View -->
            <div v-if="viewMode === 'raw'" class="shadow-box mb-3 editor-box">
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
import { ref, reactive, computed, onMounted, onUnmounted } from "vue";
import { useRoute } from "vue-router";
import { useI18n } from "vue-i18n";
import CodeMirror from "vue-codemirror6";
import { yaml as yamlLang } from "@codemirror/lang-yaml";
import { tomorrowNightEighties as editorTheme } from "../editor-theme";
import { lineNumbers } from "@codemirror/view";
import yamlLib from "yaml";
import dayjs from "dayjs";
import { FontAwesomeIcon } from "@fortawesome/vue-fontawesome";
import { BModal } from "bootstrap-vue-next";
import { useSocket } from "../composables/useSocket";
import { useAppToast } from "../composables/useAppToast";
import { ContainerStatusInfo, getComposeTerminalName } from "../common/util-common";
import ProgressTerminal from "../components/ProgressTerminal.vue";

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

const viewMode = ref<"parsed" | "raw">("parsed");
const inspectData = ref("fetching ...");
const inspectObj = ref<any>(null);
const processing = ref(false);
const showUpdateDialog = ref(false);
const dockerStats = ref<Record<string, any>>({});
const updateDialogData = reactive({
    pruneAfterUpdate: false,
    pruneAllAfterUpdate: false,
});
const progressTerminalRef = ref<InstanceType<typeof ProgressTerminal>>();
const now = ref(Date.now());
let uptimeTimer: ReturnType<typeof setInterval> | null = null;
let processTimer: ReturnType<typeof setInterval> | null = null;
let statsTimer: ReturnType<typeof setInterval> | null = null;

// Process list from containerTop
const processList = ref<Array<{ pid: string; user: string; command: string }>>([]);

const extensionsYAML = [
    editorTheme,
    yamlLang(),
    lineNumbers(),
];

const endpoint = computed(() => (route.params.endpoint as string) || "");
const containerName = computed(() => route.params.containerName as string || "");
const stackName = computed(() => containerInfo.value?.stackName || "");
const globalStack = computed(() => completeStackList.value[stackName.value + "_" + endpoint.value]);
const stackActive = computed(() => globalStack.value?.started ?? false);
const stackManaged = computed(() => globalStack.value?.isManagedByDockge ?? false);
const imageUpdatesAvailable = computed(() => globalStack.value?.imageUpdatesAvailable ?? false);
const terminalName = computed(() => stackName.value ? getComposeTerminalName(endpoint.value, stackName.value) : "");

const parsed = computed(() => inspectObj.value);

const fullImageRef = computed(() => {
    const img = parsed.value?.Config?.Image || "";
    if (img && !img.includes(":")) return img + ":latest";
    return img;
});

const stackLink = computed(() => {
    if (endpoint.value) {
        return `/stacks/${stackName.value}/${endpoint.value}`;
    }
    return `/stacks/${stackName.value}`;
});

const commandStr = computed(() => {
    if (!parsed.value) return "";
    const cmd = parsed.value.Config?.Cmd;
    if (cmd && Array.isArray(cmd) && cmd.length > 0) {
        return cmd.join(" ");
    }
    const path = parsed.value.Path;
    const args = parsed.value.Args;
    if (path) {
        return args && args.length > 0 ? `${path} ${args.join(" ")}` : path;
    }
    return "";
});

const restartPolicyStr = computed(() => {
    if (!parsed.value) return "";
    const rp = parsed.value.HostConfig?.RestartPolicy;
    if (!rp?.Name) return "";
    if (rp.Name === "on-failure" && rp.MaximumRetryCount > 0) {
        return `${rp.Name}:${rp.MaximumRetryCount}`;
    }
    return rp.Name;
});

// Docker stats for the current container
const containerStat = computed(() => {
    if (!containerName.value) return null;
    return dockerStats.value[containerName.value] || null;
});

function requestDockerStats() {
    if (!containerName.value) return;
    emitAgent(endpoint.value, "dockerStats", stackName.value, (res: any) => {
        if (res.ok) {
            dockerStats.value = res.dockerStats || {};
        }
    });
}

// Networks extracted from inspect data
const networks = computed(() => {
    if (!parsed.value?.NetworkSettings?.Networks) return [];
    const nets = parsed.value.NetworkSettings.Networks;
    return Object.entries(nets).map(([name, cfg]: [string, any]) => ({
        name,
        ipv4: cfg.IPAddress || "",
        ipv6: cfg.GlobalIPv6Address || "",
        mac: cfg.MacAddress || "",
        gateway: cfg.Gateway || "",
        aliases: cfg.Aliases || [],
    }));
});

// Mounts extracted from inspect data
const mounts = computed(() => {
    if (!parsed.value?.Mounts) return [];
    return parsed.value.Mounts;
});

function isValidDate(dateStr: string): boolean {
    if (!dateStr || dateStr.startsWith("0001-")) return false;
    return dayjs(dateStr).isValid();
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

const uptimeStr = computed(() => {
    if (!parsed.value?.State?.Running) return "";
    const startedAt = parsed.value.State?.StartedAt;
    if (!startedAt || !isValidDate(startedAt)) return "";

    const startMs = dayjs(startedAt).valueOf();
    const diffMs = now.value - startMs;
    if (diffMs < 0) return "";

    const totalMin = Math.floor(diffMs / 60000);
    const days = Math.floor(totalMin / 1440);
    const hours = Math.floor((totalMin % 1440) / 60);
    const minutes = totalMin % 60;

    const parts: string[] = [];
    if (days > 0) parts.push(`${days}d`);
    if (hours > 0) parts.push(`${hours}h`);
    parts.push(`${minutes}m`);
    return parts.join(" ");
});

function startComposeAction() {
    processing.value = true;
    progressTerminalRef.value?.show();
}

function stopComposeAction() {
    processing.value = false;
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

function fetchProcesses() {
    if (!containerName.value) return;
    emitAgent(endpoint.value, "containerTop", containerName.value, (res: any) => {
        if (res.ok && res.processes) {
            // Map columns by title position (PID, USER, COMMAND)
            const titles: string[] = res.titles || [];
            const pidIdx = titles.findIndex((t: string) => t === "PID");
            const userIdx = titles.findIndex((t: string) => t === "USER");
            const cmdIdx = titles.findIndex((t: string) => t === "COMMAND" || t === "CMD" || t === "ARGS");

            processList.value = res.processes.map((row: string[]) => ({
                pid: pidIdx >= 0 ? row[pidIdx] : row[0] || "",
                user: userIdx >= 0 ? row[userIdx] : row[1] || "",
                command: cmdIdx >= 0 ? row[cmdIdx] : row[row.length - 1] || "",
            }));
        }
    });
}

onMounted(() => {
    if (containerName.value) {
        emitAgent(endpoint.value, "containerInspect", containerName.value, (res: any) => {
            if (res.ok) {
                const data = JSON.parse(res.inspectData);
                if (Array.isArray(data) && data.length > 0) {
                    inspectObj.value = data[0];
                } else if (data) {
                    inspectObj.value = data;
                }
                if (data) {
                    inspectData.value = yamlLib.stringify(data, { lineWidth: 0 });
                }
            }
        });

        fetchProcesses();
        processTimer = setInterval(fetchProcesses, 10000);

        requestDockerStats();
        statsTimer = setInterval(requestDockerStats, 5000);
    }

    uptimeTimer = setInterval(() => {
        now.value = Date.now();
    }, 60000);
});

onUnmounted(() => {
    if (uptimeTimer) {
        clearInterval(uptimeTimer);
    }
    if (processTimer) {
        clearInterval(processTimer);
    }
    if (statsTimer) {
        clearInterval(statsTimer);
    }
});
</script>

<style scoped lang="scss">
@import "../styles/vars.scss";

.editor-box {
    font-family: 'JetBrains Mono', monospace;
    font-size: 14px;
}

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

.process-table {
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

.btn-check:active + .btn-outline-primary,
.btn-check:checked + .btn-outline-primary {
    color: #fff;

    .dark & {
        color: #000;
    }
}

.metric-cell {
    background: $dark-header-bg;
    border-radius: 10px;
    padding: 0.75rem 0.5rem;
    height: 100%;
}

.metric-label {
    font-size: 0.95rem;
    font-weight: 600;
    color: $dark-font-color;
    margin-bottom: 0.25rem;
}

.num {
    font-size: 30px;
    font-weight: bold;
    display: block;
    color: $primary;
}

.num.placeholder-value {
    opacity: 0.3;
}

.num-sub {
    font-size: 14px;
    color: $dark-font-color3;
    display: block;
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
