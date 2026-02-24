<template>
    <div class="shadow-box big-padding mb-3 container">
        <div class="row">
            <div class="col-12 col-xxl-7">
                <h4>{{ name }}</h4>
            </div>
            <div class="col-12 col-xxl-5 mb-2 d-flex justify-content-xxl-end align-items-start">
                <button
                    v-if="!isEditMode && serviceRecreateNecessary"
                    class="btn btn-sm btn-info me-2"
                    :title="$t('tooltipServiceRecreate', [name])"
                    :disabled="processing"
                    @click="recreateService"
                >
                    <font-awesome-icon icon="rocket" />
                </button>

                <button
                    v-if="!isEditMode && serviceImageUpdateAvailable"
                    v-b-modal="updateModalId"
                    class="btn btn-sm btn-info me-2"
                    :title="$t('tooltipServiceUpdate', [name])"
                    :disabled="processing"
                >
                    <font-awesome-icon icon="arrow-up" />
                </button>

                <!-- Image update modal -->
                <BModal :id="updateModalId" :ref="(el: any) => { updateModalRef = el }" :title="$tc('imageUpdate', 1)">
                    <div>
                        <h5>{{ $t("image") }}</h5>
                        <span>{{ envsubstService.image }}</span>
                    </div>
                    <div v-if="changelogLink" class="mt-3">
                        <h5>{{ $t("changelog") }}</h5>
                        <a :href="changelogLink" target="_blank">{{ changelogLink }}</a>
                    </div>

                    <BForm class="mt-3">
                        <BFormCheckbox v-model="updateDialogData.pruneAfterUpdate" switch><span v-html="$t('pruneAfterUpdate')"></span></BFormCheckbox>
                        <div style="margin-left: 2.5rem;">
                            <BFormCheckbox v-model="updateDialogData.pruneAllAfterUpdate" :checked="updateDialogData.pruneAfterUpdate && updateDialogData.pruneAllAfterUpdate" :disabled="!updateDialogData.pruneAfterUpdate"><span v-html="$t('pruneAllAfterUpdate')"></span></BFormCheckbox>
                        </div>
                    </BForm>

                    <template #footer>
                        <button class="btn btn-normal" :title="$t('tooltipServiceUpdateIgnore')" @click="skipCurrentUpdate">
                            <font-awesome-icon icon="ban" class="me-1" />{{ $t("ignoreUpdate") }}
                        </button>
                        <button class="btn btn-primary" :title="$t('tooltipDoServiceUpdate', [name])" @click="doUpdateService">
                            <font-awesome-icon icon="cloud-arrow-down" class="me-1" />{{ $t("updateStack") }}
                        </button>
                    </template>
                </BModal>

                <div v-if="!isEditMode" class="btn-group service-actions me-2" role="group">
                    <router-link v-if="started" class="btn btn-sm btn-normal" :title="$t('tooltipServiceLog', [name])" :to="logRouteLink" :disabled="processing"><font-awesome-icon icon="file-lines" /></router-link>
                    <router-link v-if="started" class="btn btn-sm btn-normal" :title="$t('tooltipServiceInspect')" :to="inspectRouteLink" :disabled="processing"><font-awesome-icon icon="info-circle" /></router-link>
                    <router-link v-if="started" class="btn btn-sm btn-normal" :title="$t('tooltipServiceTerminal', [name])" :to="terminalRouteLink" :disabled="processing"><font-awesome-icon icon="terminal" /></router-link>
                </div>

                <div v-if="!isEditMode" class="btn-group service-actions" role="group">
                    <button v-if="!started" type="button" class="btn btn-sm btn-success" :title="$t('tooltipServiceStart', [name])" :disabled="processing" @click="startService"><font-awesome-icon icon="play" /></button>
                    <button v-if="started" type="button" class="btn btn-sm btn-danger" :title="$t('tooltipServiceStop', [name])" :disabled="processing" @click="stopService"><font-awesome-icon icon="stop" /></button>
                    <button v-if="started" type="button" class="btn btn-sm btn-warning" :title="$t('tooltipServiceRestart', [name])" :disabled="processing" @click="restartService"><font-awesome-icon icon="rotate" /></button>
                </div>
            </div>
        </div>
        <div v-if="!isEditMode" class="row">
            <div class="d-flex flex-wrap justify-content-between gap-3 mb-2">
                <div class="image">
                    <router-link :to="{ name: 'imageDetail', params: { imageRef: imageName + ':' + imageTag } }" class="image-link">
                        <span class="me-1">{{ imageName }}:</span><span class="tag">{{ imageTag }}</span>
                    </router-link>
                </div>
            </div>
            <div class="col">
                <span class="badge me-1" :class="bgStyle">{{ status }}</span>

                <a v-for="port in envsubstService.ports" :key="port" :href="parsePort(port).url" target="_blank">
                    <span class="badge me-1 bg-secondary">{{ parsePort(port).display }}</span>
                </a>
            </div>
        </div>

        <div v-if="isEditMode" class="mt-2">
            <button class="btn btn-normal me-2" @click="showConfig = !showConfig">
                <font-awesome-icon icon="edit" />
                {{ $t("Edit") }}
            </button>
            <button v-if="false" class="btn btn-normal me-2">Rename</button>
            <button class="btn btn-danger me-2" @click="remove">
                <font-awesome-icon icon="trash" />
                {{ $t("deleteContainer") }}
            </button>
        </div>
        <div v-else-if="statsInstances.length > 0" class="mt-2">
            <div class="d-flex align-items-center gap-3">
                <template v-if="!expandedStats">
                    <div class="stats">
                        {{ $t('CPU') }}: {{ statsInstances[0].CPUPerc }}
                    </div>
                    <div class="stats">
                        {{ $t('memoryAbbreviated') }}: {{ statsInstances[0].MemUsage }}
                    </div>
                </template>
                <div class="d-flex flex-grow-1 justify-content-end">
                    <button class="btn btn-sm btn-normal" @click="expandedStats = !expandedStats">
                        <font-awesome-icon :icon="expandedStats ? 'chevron-up' : 'chevron-down'" />
                    </button>
                </div>
            </div>
            <transition name="slide-fade" appear>
                <div v-if="expandedStats" class="d-flex flex-column gap-3 mt-2">
                    <DockerStat
                        v-for="stat in statsInstances"
                        :key="stat.Name"
                        :stat="stat"
                    />
                </div>
            </transition>
        </div>

        <transition name="slide-fade" appear>
            <div v-if="isEditMode && showConfig" class="config mt-3">
                <!-- Image -->
                <div class="mb-4">
                    <label class="form-label">
                        {{ $t("dockerImage") }}
                    </label>
                    <div class="input-group mb-3">
                        <input
                            v-model="service.image"
                            class="form-control"
                            list="image-datalist"
                        />
                    </div>

                    <datalist id="image-datalist">
                        <option value="louislam/uptime-kuma:1" />
                    </datalist>
                    <div class="form-text"></div>
                </div>

                <!-- Ports -->
                <div class="mb-4">
                    <label class="form-label">
                        {{ $tc("port", 2) }}
                    </label>
                    <ArrayInput name="ports" :display-name="$t('port')" placeholder="HOST:CONTAINER" />
                </div>

                <!-- Volumes -->
                <div class="mb-4">
                    <label class="form-label">
                        {{ $tc("volume", 2) }}
                    </label>
                    <ArrayInput name="volumes" :display-name="$t('volume')" placeholder="HOST:CONTAINER" />
                </div>

                <!-- Restart Policy -->
                <div class="mb-4">
                    <label class="form-label">
                        {{ $t("restartPolicy") }}
                    </label>
                    <select v-model="service.restart" class="form-select">
                        <option value="always">{{ $t("restartPolicyAlways") }}</option>
                        <option value="unless-stopped">{{ $t("restartPolicyUnlessStopped") }}</option>
                        <option value="on-failure">{{ $t("restartPolicyOnFailure") }}</option>
                        <option value="no">{{ $t("restartPolicyNo") }}</option>
                    </select>
                </div>

                <!-- Environment Variables -->
                <div class="mb-4">
                    <label class="form-label">
                        {{ $tc("environmentVariable", 2) }}
                    </label>
                    <ArrayInput name="environment" :display-name="$t('environmentVariable')" placeholder="KEY=VALUE" />
                </div>

                <!-- Container Name -->
                <div v-if="false" class="mb-4">
                    <label class="form-label">
                        {{ $t("containerName") }}
                    </label>
                    <div class="input-group mb-3">
                        <input
                            v-model="service.container_name"
                            class="form-control"
                        />
                    </div>
                    <div class="form-text"></div>
                </div>

                <!-- Network -->
                <div class="mb-4">
                    <label class="form-label">
                        {{ $tc("network", 2) }}
                    </label>

                    <div v-if="networkList.length === 0 && service.networks && service.networks.length > 0" class="text-warning mb-3">
                        {{ $t("NoNetworksAvailable") }}
                    </div>

                    <ArraySelect name="networks" :display-name="$t('network')" placeholder="Network Name" :options="networkList" />
                </div>

                <!-- Depends on -->
                <div class="mb-4">
                    <label class="form-label">
                        {{ $t("dependsOn") }}
                    </label>
                    <ArrayInput name="depends_on" :display-name="$t('dependsOn')" :placeholder="$t(`containerName`)" />
                </div>

                <!-- URLs -->
                <div class="mb-4">
                    <label class="form-label">
                        {{ $tc("url", 2) }}
                    </label>
                    <ul v-if="urlList.length > 0" class="list-group url-list">
                        <li v-for="entry in urlList" :key="entry.key" class="list-group-item">
                            <input :value="entry.url" type="text" class="no-bg domain-input" placeholder="https://" @input="updateUrl(entry.key, ($event.target as HTMLInputElement).value)" />
                            <font-awesome-icon icon="times" class="action remove ms-2 me-3 text-danger" @click="removeUrl(entry.key)" />
                        </li>
                    </ul>
                    <div>
                        <button class="btn btn-normal btn-sm mt-3" @click="addUrl">{{ $t("addListItem", [$t('url')]) }}</button>
                    </div>
                </div>

                <!-- Updates -->
                <div class="mb-4">
                    <label class="form-label">
                        {{ $t("updatesHeading") }}
                    </label>
                    <div class="mb-3">
                        <BFormCheckbox v-model="statusIgnore" switch>
                            {{ $t("ignoreStatus") }}
                        </BFormCheckbox>
                    </div>
                    <div class="mb-3">
                        <BFormCheckbox v-model="imageUpdatesCheck" switch>
                            {{ $t("checkForImageUpdates") }}
                        </BFormCheckbox>
                    </div>
                    <div>
                        <input v-model="changelogUrl" type="text" class="form-control" placeholder="https://" />
                        <div class="form-text">{{ $t("changelogLink") }}</div>
                    </div>
                </div>
            </div>
        </transition>
    </div>
</template>

<script setup lang="ts">
import { ref, computed, inject, provide, reactive, type Ref } from "vue";
import { FontAwesomeIcon } from "@fortawesome/vue-fontawesome";
import { parseDockerPort } from "../../../common/util-common";
import { LABEL_STATUS_IGNORE, LABEL_IMAGEUPDATES_CHECK, LABEL_IMAGEUPDATES_CHANGELOG, LABEL_URLS_PREFIX } from "../../../common/compose-labels";
import { BModal, BForm, BFormCheckbox } from "bootstrap-vue-next";
import DockerStat from "./DockerStat.vue";
import ArrayInput from "./ArrayInput.vue";
import ArraySelect from "./ArraySelect.vue";
import { useSocket } from "../composables/useSocket";
import { useAppToast } from "../composables/useAppToast";

const { emitAgent, info } = useSocket();
const { toastRes } = useAppToast();

// Injected from Compose.vue
const jsonConfig = inject<Record<string, any>>("jsonConfig")!;
const envsubstJSONConfig = inject<Record<string, any>>("envsubstJSONConfig")!;
const composeStack = inject<Record<string, any>>("composeStack")!;
const composeEndpoint = inject<Ref<string>>("composeEndpoint")!;
const startComposeAction = inject<() => void>("startComposeAction")!;
const stopComposeAction = inject<() => void>("stopComposeAction")!;

const props = defineProps<{
    name: string;
    isEditMode?: boolean;
    first?: boolean;
    serviceStatus: any;
    serviceImageUpdateAvailable?: boolean;
    serviceRecreateNecessary?: boolean;
    dockerStats: any;
    ports?: any[];
    processing?: boolean;
}>();

const emit = defineEmits<{
    (e: "start-service", name: string): void;
    (e: "stop-service", name: string): void;
    (e: "restart-service", name: string): void;
    (e: "update-service", name: string): void;
}>();

const showConfig = ref(false);
const expandedStats = ref(false);
const updateDialogData = reactive({
    pruneAfterUpdate: false,
    pruneAllAfterUpdate: false,
});
const updateModalRef = ref<any>(null);

// Computed from injected state
const endpoint = computed(() => composeEndpoint.value);
const stackName = computed(() => composeStack.name);

const service = computed(() => {
    if (!jsonConfig.services[props.name]) {
        return {};
    }
    return jsonConfig.services[props.name];
});

// Provide service to ArrayInput and ArraySelect children
provide("service", service);

const serviceCount = computed(() => Object.keys(jsonConfig.services).length);

const envsubstService = computed(() => {
    if (!envsubstJSONConfig.services[props.name]) {
        return {};
    }
    return envsubstJSONConfig.services[props.name];
});

const networkList = computed(() => {
    const list: string[] = [];
    for (const networkName in jsonConfig.networks) {
        list.push(networkName);
    }
    return list;
});

const updateModalId = computed(() => "image-update-modal-" + props.name);

const changelogLink = computed(() => {
    const labels = envsubstService.value?.labels;
    if (labels && labels[LABEL_IMAGEUPDATES_CHANGELOG]) {
        return labels[LABEL_IMAGEUPDATES_CHANGELOG];
    }
    return "";
});

const urlList = computed(() => {
    const labels = service.value?.labels;
    if (!labels || typeof labels !== "object" || Array.isArray(labels)) {
        return [];
    }
    const entries: { key: string; url: string }[] = [];
    for (const [key, value] of Object.entries(labels)) {
        if (key.startsWith(LABEL_URLS_PREFIX)) {
            entries.push({ key, url: (value as string) || "" });
        }
    }
    return entries;
});

const statusIgnore = computed({
    get() {
        return service.value?.labels?.[LABEL_STATUS_IGNORE] === "true";
    },
    set(val: boolean) {
        ensureLabels();
        if (val) {
            service.value.labels[LABEL_STATUS_IGNORE] = "true";
        } else {
            delete service.value.labels[LABEL_STATUS_IGNORE];
        }
    },
});

const imageUpdatesCheck = computed({
    get() {
        return service.value?.labels?.[LABEL_IMAGEUPDATES_CHECK] !== "false";
    },
    set(val: boolean) {
        ensureLabels();
        if (val) {
            delete service.value.labels[LABEL_IMAGEUPDATES_CHECK];
        } else {
            service.value.labels[LABEL_IMAGEUPDATES_CHECK] = "false";
        }
    },
});

const changelogUrl = computed({
    get() {
        return service.value?.labels?.[LABEL_IMAGEUPDATES_CHANGELOG] || "";
    },
    set(val: string) {
        ensureLabels();
        if (val) {
            service.value.labels[LABEL_IMAGEUPDATES_CHANGELOG] = val;
        } else {
            delete service.value.labels[LABEL_IMAGEUPDATES_CHANGELOG];
        }
    },
});

const bgStyle = computed(() => {
    if (status.value === "running" || status.value === "healthy") {
        return "bg-primary";
    } else if (status.value === "unhealthy") {
        return "bg-danger";
    }
    return "bg-secondary";
});

const logRouteLink = computed(() => {
    return {
        name: "containerLogs",
        params: {
            containerName: containerName.value,
        },
    };
});

const containerName = computed(() => {
    if (props.serviceStatus && props.serviceStatus[0]) {
        return props.serviceStatus[0].name;
    }
    return stackName.value + "-" + props.name + "-1";
});

const inspectRouteLink = computed(() => {
    return {
        name: "containerDetail",
        params: {
            containerName: containerName.value,
        },
    };
});

const terminalRouteLink = computed(() => {
    return {
        name: "containerShell",
        params: {
            containerName: containerName.value,
            type: "bash",
        },
    };
});

const imageName = computed(() => {
    if (envsubstService.value.image) {
        return envsubstService.value.image.split(":")[0];
    }
    return "";
});

const imageTag = computed(() => {
    if (envsubstService.value.image) {
        const tag = envsubstService.value.image.split(":")[1];
        return tag || "latest";
    }
    return "";
});

const statsInstances = computed(() => {
    if (!props.serviceStatus) {
        return [];
    }
    return props.serviceStatus
        .map((s: any) => props.dockerStats[s.name])
        .filter((s: any) => !!s)
        .sort((a: any, b: any) => a.Name.localeCompare(b.Name));
});

const started = computed(() => status.value === "running" || status.value === "healthy");

const status = computed(() => {
    if (!props.serviceStatus) {
        return "N/A";
    }
    return props.serviceStatus[0].status;
});

// Methods
function parsePort(port: any) {
    if (composeStack.endpoint) {
        return parseDockerPort(port, composeStack.primaryHostname);
    }
    const hostname = info.value.primaryHostname || location.hostname;
    return parseDockerPort(port, hostname);
}

function remove() {
    delete jsonConfig.services[props.name];
}

function startService() {
    emit("start-service", props.name);
}

function stopService() {
    emit("stop-service", props.name);
}

function restartService() {
    emit("restart-service", props.name);
}

function recreateService() {
    emit("restart-service", props.name);
}

function resetUpdateDialog() {
    updateDialogData.pruneAfterUpdate = false;
    updateDialogData.pruneAllAfterUpdate = false;
}

function doUpdateService() {
    updateModalRef.value?.hide();

    startComposeAction();
    emitAgent(endpoint.value, "updateService", composeStack.name, props.name, updateDialogData.pruneAfterUpdate, updateDialogData.pruneAllAfterUpdate, (res: any) => {
        stopComposeAction();
        toastRes(res);
    });
}

function skipCurrentUpdate() {
    updateModalRef.value?.hide();
}

function ensureLabels() {
    if (!service.value.labels) {
        service.value.labels = {};
    }
}

function addUrl() {
    ensureLabels();
    let i = 0;
    let key;
    do {
        key = LABEL_URLS_PREFIX + i;
        i++;
    } while (service.value.labels[key] !== undefined);
    service.value.labels[key] = "";
}

function removeUrl(key: string) {
    delete service.value.labels[key];
}

function updateUrl(key: string, value: string) {
    service.value.labels[key] = value;
}
</script>

<style scoped lang="scss">
@import "../styles/vars";

.container {
    max-width: 100%;

    .image {
        font-size: 0.8rem;
        color: #6c757d;
        .tag {
            color: #33383b;
        }

        .image-link {
            text-decoration: none;
            color: inherit;

            &:hover {
                text-decoration: underline;
            }

            .dark & .tag {
                color: $dark-font-color;
            }
        }
    }

    .status {
        font-size: 0.8rem;
        color: #6c757d;
    }

    .notification {
        font-size: 1rem;
        color: $danger;
    }

    .function {
        align-content: center;
        display: flex;
        height: 100%;
        width: 100%;
        align-items: center;
        justify-content: end;
    }

    .stats-select {
        cursor: pointer;
        font-size: 1rem;
        color: #6c757d;
    }

    .stats {
        font-size: 0.8rem;
        color: #6c757d;
    }

    .service-actions .btn {
        width: 45px;
        padding-left: 0;
        padding-right: 0;
        text-align: center;
    }

    .url-list {
        background-color: $dark-bg2;

        li {
            display: flex;
            align-items: center;
            padding: 10px 0 10px 10px;

            .domain-input {
                flex-grow: 1;
                background-color: $dark-bg2;
                border: none;
                color: $dark-font-color;
                outline: none;

                &::placeholder {
                    color: #1d2634;
                }
            }
        }
    }
}
</style>
