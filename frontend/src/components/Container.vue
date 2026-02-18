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
                    :title="$t('tooltipServiceRecreate')"
                    :disabled="processing"
                    @click="recreateService"
                >
                    <font-awesome-icon icon="rocket" />
                </button>

                <button
                    v-if="!isEditMode && serviceImageUpdateAvailable"
                    v-b-modal="updateModalId"
                    class="btn btn-sm btn-info me-2"
                    :title="$t('tooltipServiceUpdate')"
                    :disabled="processing"
                >
                    <font-awesome-icon icon="arrow-up" />
                </button>

                <!-- Image update modal -->
                <BModal :id="updateModalId" :ref="updateModalId" :title="$tc('imageUpdate', 1)">
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
                        <button class="btn btn-primary" :title="$t('tooltipDoServiceUpdate')" @click="updateService">
                            <font-awesome-icon icon="cloud-arrow-down" class="me-1" />{{ $t("updateStack") }}
                        </button>
                    </template>
                </BModal>

                <div v-if="!isEditMode" class="btn-group service-actions me-2" role="group">
                    <router-link v-if="started" class="btn btn-sm btn-normal" :title="$t('tooltipServiceLog')" :to="logRouteLink" :disabled="processing"><font-awesome-icon icon="file-lines" /></router-link>
                    <router-link v-if="started" class="btn btn-sm btn-normal" :title="$t('tooltipServiceInspect')" :to="inspectRouteLink" :disabled="processing"><font-awesome-icon icon="info-circle" /></router-link>
                    <router-link v-if="started" class="btn btn-sm btn-normal" :title="$t('tooltipServiceTerminal')" :to="terminalRouteLink" :disabled="processing"><font-awesome-icon icon="terminal" /></router-link>
                </div>

                <div v-if="!isEditMode" class="btn-group service-actions" role="group">
                    <button v-if="!started" type="button" class="btn btn-sm btn-success" :title="$t('tooltipServiceStart')" :disabled="processing" @click="startService"><font-awesome-icon icon="play" /></button>
                    <button v-if="started" type="button" class="btn btn-sm btn-danger" :title="$t('tooltipServiceStop')" :disabled="processing" @click="stopService"><font-awesome-icon icon="stop" /></button>
                    <button v-if="started" type="button" class="btn btn-sm btn-warning" :title="$t('tooltipServiceRestart')" :disabled="processing" @click="restartService"><font-awesome-icon icon="rotate" /></button>
                </div>
            </div>
        </div>
        <div v-if="!isEditMode" class="row">
            <div class="d-flex flex-wrap justify-content-between gap-3 mb-2">
                <div class="image">
                    <span class="me-1">{{ imageName }}:</span><span class="tag">{{ imageTag }}</span>
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

                    <!-- TODO: Search online: https://hub.docker.com/api/content/v1/products/search?q=louislam%2Fuptime&source=community&page=1&page_size=4 -->
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
            </div>
        </transition>
    </div>
</template>

<script>
import { defineComponent } from "vue";
import { FontAwesomeIcon } from "@fortawesome/vue-fontawesome";
import { parseDockerPort } from "../../../common/util-common";
import { LABEL_IMAGEUPDATES_CHANGELOG } from "../../../common/compose-labels";
import { BModal, BForm, BFormCheckbox } from "bootstrap-vue-next";
import DockerStat from "./DockerStat.vue";

export default defineComponent({
    components: {
        FontAwesomeIcon,
        DockerStat,
        BModal,
        BForm,
        BFormCheckbox,
    },
    props: {
        name: {
            type: String,
            required: true,
        },
        isEditMode: {
            type: Boolean,
            default: false,
        },
        first: {
            type: Boolean,
            default: false,
        },
        serviceStatus: {
            type: Object,
            default: null,
        },
        serviceImageUpdateAvailable: {
            type: Boolean,
            default: false,
        },
        serviceRecreateNecessary: {
            type: Boolean,
            default: false,
        },
        dockerStats: {
            type: Object,
            default: null,
        },
        ports: {
            type: Array,
            default: null
        },
        processing: {
            type: Boolean,
            default: false,
        }
    },
    emits: [
        "start-service",
        "stop-service",
        "restart-service",
    ],
    data() {
        return {
            showConfig: false,
            expandedStats: false,
            updateDialogData: {
                pruneAfterUpdate: false,
                pruneAllAfterUpdate: false,
            },
        };
    },
    computed: {

        networkList() {
            let list = [];
            for (const networkName in this.jsonObject.networks) {
                list.push(networkName);
            }
            return list;
        },

        updateModalId() {
            return "image-update-modal-" + this.name;
        },

        changelogLink() {
            const labels = this.service?.labels;
            if (labels && labels[LABEL_IMAGEUPDATES_CHANGELOG]) {
                return labels[LABEL_IMAGEUPDATES_CHANGELOG];
            }
            return "";
        },

        bgStyle() {
            if (this.status === "running" || this.status === "healthy") {
                return "bg-primary";
            } else if (this.status === "unhealthy") {
                return "bg-danger";
            } else {
                return "bg-secondary";
            }
        },

        logRouteLink() {
            if (this.endpoint) {
                return {
                    name: "containerLogEndpoint",
                    params: {
                        endpoint: this.endpoint,
                        stackName: this.stackName,
                        serviceName: this.name,
                    },
                };
            } else {
                return {
                    name: "containerLog",
                    params: {
                        stackName: this.stackName,
                        serviceName: this.name,
                    },
                };
            }
        },

        containerName() {
            if (this.serviceStatus && this.serviceStatus[0]) {
                return this.serviceStatus[0].name;
            }
            return this.stackName + "-" + this.name + "-1";
        },

        inspectRouteLink() {
            if (this.endpoint) {
                return {
                    name: "containerInspectEndpoint",
                    params: {
                        endpoint: this.endpoint,
                        containerName: this.containerName,
                    },
                };
            } else {
                return {
                    name: "containerInspect",
                    params: {
                        containerName: this.containerName,
                    },
                };
            }
        },

        terminalRouteLink() {
            if (this.endpoint) {
                return {
                    name: "containerTerminalEndpoint",
                    params: {
                        endpoint: this.endpoint,
                        stackName: this.stackName,
                        serviceName: this.name,
                        type: "bash",
                    },
                };
            } else {
                return {
                    name: "containerTerminal",
                    params: {
                        stackName: this.stackName,
                        serviceName: this.name,
                        type: "bash",
                    },
                };
            }
        },

        endpoint() {
            return this.$parent.$parent.endpoint;
        },

        stack() {
            return this.$parent.$parent.stack;
        },

        stackName() {
            return this.$parent.$parent.stack.name;
        },

        service() {
            if (!this.jsonObject.services[this.name]) {
                return {};
            }
            return this.jsonObject.services[this.name];
        },

        serviceCount() {
            return Object.keys(this.jsonObject.services).length;
        },

        jsonObject() {
            return this.$parent.$parent.jsonConfig;
        },

        envsubstJSONConfig() {
            return this.$parent.$parent.envsubstJSONConfig;
        },

        envsubstService() {
            if (!this.envsubstJSONConfig.services[this.name]) {
                return {};
            }
            return this.envsubstJSONConfig.services[this.name];
        },

        imageName() {
            if (this.envsubstService.image) {
                return this.envsubstService.image.split(":")[0];
            } else {
                return "";
            }
        },

        imageTag() {
            if (this.envsubstService.image) {
                let tag = this.envsubstService.image.split(":")[1];

                if (tag) {
                    return tag;
                } else {
                    return "latest";
                }
            } else {
                return "";
            }
        },
        statsInstances() {
            if (!this.serviceStatus) {
                return [];
            }

            return this.serviceStatus
                .map(s => this.dockerStats[s.name])
                .filter(s => !!s)
                .sort((a, b) => a.Name.localeCompare(b.Name));
        },
        started() {
            return this.status === "running" || this.status === "healthy";
        },
        status() {
            if (!this.serviceStatus) {
                return "N/A";
            }
            return this.serviceStatus[0].status;
        }
    },
    mounted() {
        if (this.first) {
            //this.showConfig = true;
        }
    },
    methods: {
        parsePort(port) {
            if (this.stack.endpoint) {
                return parseDockerPort(port, this.stack.primaryHostname);
            } else {
                let hostname = this.$root.info.primaryHostname || location.hostname;
                return parseDockerPort(port, hostname);
            }
        },
        remove() {
            delete this.jsonObject.services[this.name];
        },
        startService() {
            this.$emit("start-service", this.name);
        },
        stopService() {
            this.$emit("stop-service", this.name);
        },
        restartService() {
            this.$emit("restart-service", this.name);
        },
        recreateService() {
            this.$emit("restart-service", this.name);
        },
        resetUpdateDialog() {
            this.updateDialogData = {
                pruneAfterUpdate: false,
                pruneAllAfterUpdate: false,
            };
        },
        updateService() {
            this.$refs[this.updateModalId].hide();

            this.$parent.$parent.startComposeAction();
            this.$root.emitAgent(this.endpoint, "updateService", this.stack.name, this.name, this.updateDialogData.pruneAfterUpdate, this.updateDialogData.pruneAllAfterUpdate, (res) => {
                this.$parent.$parent.stopComposeAction();
                this.$root.toastRes(res);
            });
        },
        skipCurrentUpdate() {
            this.$refs[this.updateModalId].hide();
        }
    }
});
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
}
</style>
