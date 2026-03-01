<template>
    <div class="shadow-box big-padding mb-3 container" role="region" :aria-label="name">
        <!-- Container name with status badge -->
        <h5 class="mb-3">
            <span v-if="!isEditMode" class="badge rounded-pill me-2" :class="bgStyle">{{ $t(containerStatusInfo.label) }}</span>
            <router-link v-if="!isEditMode && containerExists" :to="inspectRouteLink" class="stack-link">{{ containerName }}</router-link>
            <span v-else-if="!isEditMode">{{ containerName }}</span>
            <template v-else>{{ name }}</template>
        </h5>

        <!-- Container, image, ports chips -->
        <div v-if="!isEditMode" class="network-props">
            <div class="network-chip chip-link" @click="emit('scroll-to-service', name)">
                <span class="chip-label">{{ $t("service") }}</span>
                <code>{{ name }}</code>
            </div>
            <router-link v-if="imageRef" :to="{ name: 'imageDetail', params: { imageRef: imageName + ':' + imageTag } }" class="network-chip chip-link">
                <span class="chip-label">{{ $t("image") }}</span>
                <code>{{ imageName }}:{{ imageTag }}</code>
            </router-link>
            <div v-if="envsubstService.ports && envsubstService.ports.length > 0" class="network-chip">
                <span class="chip-label">{{ $tc("port", 2) }}</span>
                <span>
                    <template v-for="(port, i) in envsubstService.ports" :key="port"><a :href="parsePort(port).url" target="_blank" class="chip-port-link"><code>{{ parsePort(port).display }}</code></a><span v-if="i < envsubstService.ports.length - 1" class="chip-sep">, </span></template>
                </span>
            </div>
        </div>

        <!-- Action/log/shell buttons -->
        <div v-if="!isEditMode" class="d-flex justify-content-end align-items-center mt-3">
            <div v-if="started" class="btn-group service-actions" role="group">
                <router-link class="btn btn-sm btn-normal" :title="$t('tooltipServiceLog', [name])" :aria-label="$t('tooltipServiceLog', [name])" :to="logRouteLink" :disabled="processing"><svg class="svg-icon" :viewBox="icons['file-lines'].viewBox"><path fill="currentColor" :d="icons['file-lines'].path" /></svg></router-link>
                <router-link class="btn btn-sm btn-normal" :title="$t('tooltipServiceTerminal', [name])" :aria-label="$t('tooltipServiceTerminal', [name])" :to="terminalRouteLink" :disabled="processing"><svg class="svg-icon" :viewBox="icons.terminal.viewBox"><path fill="currentColor" :d="icons.terminal.path" /></svg></router-link>
            </div>
            <div class="btn-group service-actions ms-2" role="group">
                <button v-if="!started" type="button" class="btn btn-sm btn-primary" :title="tooltipStart" :aria-label="tooltipStart" :disabled="processing" @click="startService"><svg class="svg-icon" :viewBox="icons.play.viewBox"><path fill="currentColor" :d="icons.play.path" /></svg></button>
                <button v-if="started" type="button" class="btn btn-sm btn-normal" :title="tooltipRestart" :aria-label="tooltipRestart" :disabled="processing" @click="restartService"><svg class="svg-icon" :viewBox="icons.rotate.viewBox"><path fill="currentColor" :d="icons.rotate.path" /></svg></button>
                <button type="button" class="btn btn-sm" :class="serviceRecreateNecessary ? 'btn-info' : 'btn-normal'" :title="tooltipRecreate" :aria-label="tooltipRecreate" :disabled="processing" @click="recreateService"><svg class="svg-icon" :viewBox="icons.rocket.viewBox"><path fill="currentColor" :d="icons.rocket.path" /></svg></button>
                <button type="button" class="btn btn-sm" :class="serviceImageUpdateAvailable ? 'btn-info' : 'btn-normal'" :title="tooltipUpdate" :aria-label="tooltipUpdate" :disabled="processing" @click="emit('update-service', name)"><svg class="svg-icon" :viewBox="icons['cloud-arrow-down'].viewBox"><path fill="currentColor" :d="icons['cloud-arrow-down'].path" /></svg></button>
                <button v-if="started" type="button" class="btn btn-sm btn-normal" :title="tooltipStop" :aria-label="tooltipStop" :disabled="processing" @click="stopService"><svg class="svg-icon" :viewBox="icons.stop.viewBox"><path fill="currentColor" :d="icons.stop.path" /></svg></button>
            </div>
        </div>

        <div v-if="isEditMode" class="mt-2">
            <button class="btn btn-normal me-2" @click="showConfig = !showConfig">
                <svg class="svg-icon" :viewBox="icons.edit.viewBox"><path fill="currentColor" :d="icons.edit.path" /></svg>
                {{ $t("Edit") }}
            </button>
            <button v-if="false" class="btn btn-normal me-2">Rename</button>
            <button class="btn btn-danger me-2" @click="remove">
                <svg class="svg-icon" :viewBox="icons.trash.viewBox"><path fill="currentColor" :d="icons.trash.path" /></svg>
                {{ $t("deleteContainer") }}
            </button>
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
                            <svg class="svg-icon action remove ms-2 me-3 text-danger" :viewBox="icons.times.viewBox" @click="removeUrl(entry.key)"><path fill="currentColor" :d="icons.times.path" /></svg>
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
import { ref, computed, inject, provide, type Ref } from "vue";
import { parseDockerPort, ContainerStatusInfo } from "../common/util-common";
import { containerIcons } from "./container-icons";
import { LABEL_STATUS_IGNORE, LABEL_IMAGEUPDATES_CHECK, LABEL_IMAGEUPDATES_CHANGELOG, LABEL_URLS_PREFIX } from "../common/compose-labels";
import { BFormCheckbox } from "bootstrap-vue-next";
import ArrayInput from "./ArrayInput.vue";
import ArraySelect from "./ArraySelect.vue";
import { useI18n } from "vue-i18n";
import { useSocket } from "../composables/useSocket";
import { useAppToast } from "../composables/useAppToast";

const icons = containerIcons;

const { t } = useI18n();
const { info } = useSocket();
const { toastRes } = useAppToast();

// Injected from Compose.vue
const jsonConfig = inject<Record<string, any>>("jsonConfig")!;
const envsubstJSONConfig = inject<Record<string, any>>("envsubstJSONConfig")!;
const composeStack = inject<Record<string, any>>("composeStack")!;
const startComposeAction = inject<() => void>("startComposeAction")!;
const stopComposeAction = inject<() => void>("stopComposeAction")!;

const props = defineProps<{
    name: string;
    isEditMode?: boolean;
    first?: boolean;
    serviceStatus: any;
    serviceImageUpdateAvailable?: boolean;
    serviceRecreateNecessary?: boolean;
    ports?: any[];
    processing?: boolean;
    isManaged?: boolean;
}>();

const emit = defineEmits<{
    (e: "start-service", name: string): void;
    (e: "stop-service", name: string): void;
    (e: "restart-service", name: string): void;
    (e: "recreate-service", name: string): void;
    (e: "update-service", name: string): void;
    (e: "scroll-to-service", name: string): void;
}>();

const showConfig = ref(false);

// Computed from injected state
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
    if (!envsubstJSONConfig.services?.[props.name]) {
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

const containerStatusInfo = computed(() => {
    if (!props.serviceStatus?.[0]) return ContainerStatusInfo.UNKNOWN;
    return ContainerStatusInfo.from(props.serviceStatus[0]);
});

const bgStyle = computed(() => `bg-${containerStatusInfo.value.badgeColor}`);
const containerExists = computed(() => !!props.serviceStatus?.[0]);

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

const imageRef = computed(() => {
    // Prefer the actual running image from Docker (matches container detail page)
    if (props.serviceStatus?.[0]?.image) {
        return props.serviceStatus[0].image;
    }
    // Fall back to compose YAML (e.g. when container isn't running)
    if (envsubstService.value.image) {
        return envsubstService.value.image;
    }
    return "";
});

const imageName = computed(() => {
    const ref = imageRef.value;
    return ref ? ref.split(":")[0] : "";
});

const imageTag = computed(() => {
    const ref = imageRef.value;
    if (!ref) return "";
    return ref.split(":")[1] || "latest";
});

const started = computed(() => status.value === "running" || status.value === "healthy" || status.value === "unhealthy");

const status = computed(() => {
    if (!props.serviceStatus?.[0]) return "N/A";
    const c = props.serviceStatus[0];
    if (c.health === "unhealthy") return "unhealthy";
    if (c.health === "healthy") return "healthy";
    return c.state || "N/A";
});

// Tooltips: show the actual docker command that will run
const tooltipStart = computed(() => props.isManaged !== false
    ? t("tooltipServiceStart", [props.name])
    : t("tooltipContainerStart", [containerName.value]));
const tooltipStop = computed(() => props.isManaged !== false
    ? t("tooltipServiceStop", [props.name])
    : t("tooltipContainerStop", [containerName.value]));
const tooltipRestart = computed(() => props.isManaged !== false
    ? t("tooltipServiceRestart", [props.name])
    : t("tooltipContainerRestart", [containerName.value]));
const tooltipRecreate = computed(() => props.isManaged !== false
    ? t("tooltipServiceRecreate", [props.name])
    : t("tooltipContainerRecreate", [containerName.value]));
const tooltipUpdate = computed(() => props.isManaged !== false
    ? t("tooltipServiceUpdate", [props.name])
    : t("tooltipContainerUpdate", [imageRef.value, containerName.value]));

// Methods
function parsePort(port: any) {
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
    emit("recreate-service", props.name);
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

.svg-icon {
    display: inline-block;
    height: 1em;
    vertical-align: -0.125em;
    overflow: visible;
    box-sizing: content-box;
    fill: currentColor;
}

.container {
    max-width: 100%;

    .network-props {
        display: flex;
        flex-wrap: wrap;
        gap: 0.5rem;
    }

    .network-chip {
        display: inline-flex;
        align-items: baseline;
        gap: 0.4rem;
        background: rgba(0, 0, 0, 0.06);
        border-radius: 10px;
        padding: 0.3rem 0.6rem;

        .dark & {
            background: $dark-header-bg;
        }

        .chip-label {
            font-size: 0.8em;
            font-weight: 600;
            color: $dark-font-color3;
            text-transform: uppercase;
            white-space: nowrap;
        }

        code {
            font-family: 'JetBrains Mono', monospace;
            font-size: 0.85em;
            color: $primary;
            background: none;
        }
    }

    .chip-link {
        text-decoration: none;
        cursor: pointer;

        &:hover {
            text-decoration: none;

            code {
                text-decoration: underline;
            }
        }
    }

    .chip-port-link {
        text-decoration: none;

        &:hover code {
            text-decoration: underline;
        }
    }

    .chip-sep {
        color: $dark-font-color3;
    }

    .stack-link {
        font-weight: 600;
        text-decoration: none;
        color: inherit;

        &:hover {
            color: lighten($primary, 10%);
        }
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
