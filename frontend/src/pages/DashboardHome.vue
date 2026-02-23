<template>
    <transition ref="tableContainerRef" name="slide-fade" appear>
        <div v-if="$route.name === 'DashboardHome'">
            <h1 class="mb-3">
                {{ $t("stacks") }}
            </h1>

            <div class="row first-row">
                <!-- Left -->
                <div class="col-md-7">
                    <!-- Stats -->
                    <div class="shadow-box big-padding text-center mb-4">
                        <div class="row">
                            <div class="col">
                                <h3>{{ $t("active") }}</h3>
                                <span class="num active">{{ activeNum }}</span>
                            </div>
                            <div class="col">
                                <h3>{{ $t("partially") }}</h3>
                                <span class="num partially">{{ partiallyNum }}</span>
                            </div>
                            <div class="col">
                                <h3>{{ $t("unhealthy") }}</h3>
                                <span class="num unhealthy">{{ unhealthyNum }}</span>
                            </div>
                            <div class="col">
                                <h3>{{ $t("exited") }}</h3>
                                <span class="num exited">{{ exitedNum }}</span>
                            </div>
                            <div class="col">
                                <h3>{{ $t("down") }}</h3>
                                <span class="num inactive">{{ downNum }}</span>
                            </div>
                            <div class="col">
                                <h3>{{ $t("updates") }}</h3>
                                <span class="num update-available">{{ updateAvailableNum }}</span>
                            </div>
                        </div>
                    </div>

                    <!-- Docker Run -->
                    <h2 class="mb-3">{{ $t("Docker Run") }}</h2>
                    <div class="mb-3">
                        <textarea id="name" v-model="dockerRunCommand" type="text" class="form-control docker-run shadow-box" required placeholder="docker run ..."></textarea>
                    </div>

                    <button class="btn-normal btn mb-4" @click="convertDockerRun">{{ $t("Convert to Compose") }}</button>
                </div>
                <!-- Right -->
                <div class="col-md-5">
                    <!-- Agent List -->
                    <div class="shadow-box big-padding">
                        <h4 class="mb-3">{{ $tc("dockgeAgent", 2) }} <span class="badge bg-warning" style="font-size: 12px;">beta</span></h4>

                        <div v-for="(agent, endpoint) in agentList" :key="endpoint" class="mb-3 agent">
                            <!-- Agent Status -->
                            <template v-if="agentStatusList[endpoint]">
                                <span v-if="agentStatusList[endpoint] === 'online'" class="badge bg-primary me-2">{{ $t("agentOnline") }}</span>
                                <span v-else-if="agentStatusList[endpoint] === 'offline'" class="badge bg-danger me-2">{{ $t("agentOffline") }}</span>
                                <span v-else class="badge bg-secondary me-2">{{ $t(agentStatusList[endpoint]) }}</span>
                            </template>

                            <!-- Agent Display Name -->
                            <template v-if="agentStatusList[endpoint]">
                                <span v-if="endpoint === '' && agent.name === ''" class="badge bg-secondary me-2">Controller</span>
                                <span v-else-if="agent.name === ''" :href="agent.url" class="me-2">{{ endpoint }}</span>
                                <span v-else :href="agent.url" class="me-2">{{ agent.name }}</span>
                            </template>

                            <!-- Edit Name  -->
                            <font-awesome-icon icon="pen-to-square" @click="showEditAgentNameDialog[agent.name] = !showEditAgentNameDialog[agent.Name]" />

                            <!-- Edit Dialog -->
                            <BModal v-model="showEditAgentNameDialog[agent.name]" :no-close-on-backdrop="true" :close-on-esc="true" :okTitle="$t('Update Name')" okVariant="info" @ok="updateName(agent.url, agent.updatedName)">
                                <label for="Update Name" class="form-label">Current value: {{ $t(agent.name) }}</label>
                                <input id="updatedName" v-model="agent.updatedName" type="text" class="form-control" optional>
                            </BModal>

                            <!-- Remove Button -->
                            <font-awesome-icon v-if="endpoint !== ''" class="ms-2 remove-agent" icon="trash" @click="showRemoveAgentDialog[agent.url] = !showRemoveAgentDialog[agent.url]" />

                            <!-- Remove Agent Dialog -->
                            <BModal v-model="showRemoveAgentDialog[agent.url]" :okTitle="$t('removeAgent')" okVariant="danger" @ok="removeAgent(agent.url)">
                                <p>{{ agent.url }}</p>
                                {{ $t("removeAgentMsg") }}
                            </BModal>
                        </div>

                        <button v-if="!showAgentForm" class="btn btn-normal" @click="showAgentForm = !showAgentForm">{{ $t("addAgent") }}</button>

                        <!-- Add Agent Form -->
                        <form v-if="showAgentForm" @submit.prevent="addAgent">
                            <div class="mb-3">
                                <label for="url" class="form-label">{{ $t("dockgeURL") }}</label>
                                <input id="url" v-model="agent.url" type="url" class="form-control" required placeholder="http://">
                            </div>

                            <div class="mb-3">
                                <label for="username" class="form-label">{{ $t("Username") }}</label>
                                <input id="username" v-model="agent.username" type="text" class="form-control" required>
                            </div>

                            <div class="mb-3">
                                <label for="password" class="form-label">{{ $t("Password") }}</label>
                                <input id="password" v-model="agent.password" type="password" class="form-control" required autocomplete="new-password">
                            </div>

                            <div class="mb-3">
                                <label for="name" class="form-label">{{ $t("Friendly Name") }}</label>
                                <input id="name" v-model="agent.name" type="text" class="form-control" optional>
                            </div>

                            <button type="submit" class="btn btn-primary" :disabled="connectingAgent">
                                <template v-if="connectingAgent">{{ $t("connecting") }}</template>
                                <template v-else>{{ $t("connect") }}</template>
                            </button>
                        </form>
                    </div>
                </div>
            </div>
        </div>
    </transition>
    <router-view ref="child" />
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch, onMounted, onBeforeUnmount, nextTick } from "vue";
import { useRouter } from "vue-router";
import { statusNameShort } from "../../../common/util-common";
import { useSocket } from "../composables/useSocket";
import { useAppToast } from "../composables/useAppToast";

defineProps<{
    calculatedHeight?: number;
}>();

const router = useRouter();
const { completeStackList, allAgentStackList, agentList, agentStatusList, composeTemplate, getSocket } = useSocket();
const { toastRes } = useAppToast();

const page = ref(1);
const perPage = ref(25);
const initialPerPage = ref(25);
const importantHeartBeatListLength = ref(0);
const displayedRecords = ref<any[]>([]);
const dockerRunCommand = ref("");
const showAgentForm = ref(false);
const showRemoveAgentDialog = reactive<Record<string, boolean>>({});
const showEditAgentNameDialog = reactive<Record<string, boolean>>({});
const connectingAgent = ref(false);
const agent = reactive({
    url: "http://",
    username: "",
    password: "",
    name: "",
    updatedName: "",
});
const tableContainerRef = ref<HTMLElement>();

function getStatusNum(statusName: string) {
    let num = 0;
    for (let stackName in completeStackList.value) {
        const stack = (completeStackList.value as any)[stackName];
        if (statusNameShort(stack.status) === statusName) {
            num += 1;
        }
    }
    return num;
}

const activeNum = computed(() => getStatusNum("active"));
const partiallyNum = computed(() => getStatusNum("partially"));
const unhealthyNum = computed(() => getStatusNum("unhealthy"));
const downNum = computed(() => getStatusNum("down"));
const exitedNum = computed(() => getStatusNum("exited"));
const updateAvailableNum = computed(() => {
    let num = 0;
    for (let stackName in completeStackList.value) {
        const stack = (completeStackList.value as any)[stackName];
        if (stack.imageUpdatesAvailable) {
            num += 1;
        }
    }
    return num;
});

function addAgent() {
    connectingAgent.value = true;
    getSocket().emit("addAgent", agent, (res: any) => {
        toastRes(res);

        if (res.ok) {
            showAgentForm.value = false;
            Object.assign(agent, {
                url: "http://",
                username: "",
                password: "",
                name: "",
                updatedName: "",
            });
        }

        connectingAgent.value = false;
    });
}

function removeAgent(url: string) {
    getSocket().emit("removeAgent", url, (res: any) => {
        if (res.ok) {
            toastRes(res);

            let urlObj = new URL(url);
            let endpoint = urlObj.host;

            // Remove the stack list and status list of the removed agent
            delete allAgentStackList.value[endpoint];
        }
    });
}

function updateName(url: string, updatedName: string) {
    getSocket().emit("updateAgent", url, updatedName, (res: any) => {
        toastRes(res);

        if (res.ok) {
            showAgentForm.value = false;
            Object.assign(agent, {
                updatedName: "",
            });
        }
    });
}

function convertDockerRun() {
    if (dockerRunCommand.value.trim() === "docker run") {
        throw new Error("Please enter a docker run command");
    }

    getSocket().emit("composerize", dockerRunCommand.value, (res: any) => {
        if (res.ok) {
            composeTemplate.value = res.composeTemplate;
            router.push("/stacks/compose");
        } else {
            toastRes(res);
        }
    });
}

function onNewImportantHeartbeat(heartbeat: any) {
    if (page.value === 1) {
        displayedRecords.value.unshift(heartbeat);
        if (displayedRecords.value.length > perPage.value) {
            displayedRecords.value.pop();
        }
        importantHeartBeatListLength.value += 1;
    }
}

function getImportantHeartbeatListLength() {
    getSocket().emit("monitorImportantHeartbeatListCount", null, (res: any) => {
        if (res.ok) {
            importantHeartBeatListLength.value = res.count;
            getImportantHeartbeatListPaged();
        }
    });
}

function getImportantHeartbeatListPaged() {
    const offset = (page.value - 1) * perPage.value;
    getSocket().emit("monitorImportantHeartbeatListPaged", null, offset, perPage.value, (res: any) => {
        if (res.ok) {
            displayedRecords.value = res.data;
        }
    });
}

function updatePerPage() {
    const tableContainer = tableContainerRef.value;
    if (!tableContainer) return;
    const tableContainerHeight = tableContainer.offsetHeight;
    const availableHeight = window.innerHeight - tableContainerHeight;
    const additionalPerPage = Math.floor(availableHeight / 58);

    if (additionalPerPage > 0) {
        perPage.value = Math.max(initialPerPage.value, perPage.value + additionalPerPage);
    } else {
        perPage.value = initialPerPage.value;
    }
}

watch(perPage, () => {
    nextTick(() => {
        getImportantHeartbeatListPaged();
    });
});

watch(page, () => {
    getImportantHeartbeatListPaged();
});

onMounted(() => {
    initialPerPage.value = perPage.value;
    window.addEventListener("resize", updatePerPage);
    updatePerPage();
});

onBeforeUnmount(() => {
    window.removeEventListener("resize", updatePerPage);
});
</script>

<style lang="scss" scoped>
@import "../styles/vars";

.num {
    font-size: 30px;

    font-weight: bold;
    display: block;

    &.active {
        color: $primary;
    }

    &.partially {
        color: $info;
    }

    &.unhealthy {
        color: $danger;
    }

    &.exited {
        color: $warning;
    }

    &.update-available {
        color: $info;
    }
}

.shadow-box {
    padding: 20px;
}

table {
    font-size: 14px;

    tr {
        transition: all ease-in-out 0.2ms;
    }

    @media (max-width: 550px) {
        table-layout: fixed;
        overflow-wrap: break-word;
    }
}

.docker-run {
    border: none;
    font-family: 'JetBrains Mono', monospace;
    font-size: 15px;
}

.first-row .shadow-box {

}

.remove-agent {
    cursor: pointer;
    color: rgba(255, 255, 255, 0.3);
}

.agent {
    a {
        text-decoration: none;
    }
}

</style>
