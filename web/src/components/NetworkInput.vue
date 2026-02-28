<template>
    <div>
        <h5>{{ $t("Internal Networks") }}</h5>
        <ul class="list-group">
            <li v-for="(networkRow, index) in networkList" :key="index" class="list-group-item">
                <input v-model="networkRow.key" type="text" class="no-bg domain-input" :placeholder="$t(`Network name...`)" />
                <font-awesome-icon icon="times" class="action remove ms-2 me-3 text-danger" @click="remove(index)" />
            </li>
        </ul>

        <button class="btn btn-normal btn-sm mt-3 me-2" @click="addField">{{ $t("addInternalNetwork") }}</button>

        <h5 class="mt-3">{{ $t("External Networks") }}</h5>

        <div v-if="externalNetworkList.length === 0">
            {{ $t("No External Networks") }}
        </div>

        <div v-for="(networkName, index) in externalNetworkList" :key="networkName" class="form-check form-switch my-3">
            <input :id=" 'external-network' + index" v-model="selectedExternalList[networkName]" class="form-check-input" type="checkbox">

            <label class="form-check-label" :for=" 'external-network' +index">
                {{ networkName }}
            </label>

            <span v-if="false" class="text-danger ms-2 delete">Delete</span>
        </div>

        <div v-if="false" class="input-group mb-3">
            <input
                placeholder="New external network name..."
                class="form-control"
                @keyup.enter="createExternelNetwork"
            />
            <button class="btn btn-normal btn-sm  me-2" type="button">
                {{ $t("createExternalNetwork") }}
            </button>
        </div>

        <div v-if="false">
            <button class="btn btn-primary btn-sm mt-3 me-2" @click="applyToYAML">{{ $t("applyToYAML") }}</button>
        </div>
    </div>
</template>

<script setup lang="ts">
import { ref, reactive, inject, watch, onMounted, type Ref } from "vue";
import { useNetworkStore } from "../stores/networkStore";
import { useAppToast } from "../composables/useAppToast";

const networkStore = useNetworkStore();
const { toastRes } = useAppToast();

const jsonConfig = inject<Record<string, any>>("jsonConfig")!;
const stack = inject<Record<string, any>>("composeStack")!;
const editorFocus = inject<Ref<boolean>>("editorFocus")!;
const endpoint = inject<Ref<string>>("composeEndpoint")!;

const networkList = ref<Array<{ key: string; value: any }>>([]);
const externalList = reactive<Record<string, any>>({});
const selectedExternalList = reactive<Record<string, boolean>>({});
const externalNetworkList = ref<string[]>([]);

function loadNetworkList() {
    networkList.value = [];
    // Clear externalList
    for (const key in externalList) {
        delete externalList[key];
    }

    for (const key in jsonConfig.networks) {
        let obj = {
            key: key,
            value: jsonConfig.networks[key],
        };

        if (obj.value && obj.value.external) {
            externalList[key] = Object.assign({}, obj.value);
        } else {
            networkList.value.push(obj);
        }
    }

    // Restore selectedExternalList
    for (const key in selectedExternalList) {
        delete selectedExternalList[key];
    }
    for (const networkName in externalList) {
        selectedExternalList[networkName] = true;
    }
}

function loadExternalNetworkList() {
    externalNetworkList.value = networkStore.networks
        .map(n => n.name)
        .filter(n => {
            // Filter out this stack networks
            if (n.startsWith(stack.name + "_")) {
                return false;
            }
            // They should be not supported.
            // https://docs.docker.com/compose/compose-file/06-networks/#host-or-none
            if (n === "none" || n === "host" || n === "bridge") {
                return false;
            }
            return true;
        });
}

function addField() {
    networkList.value.push({
        key: "",
        value: {},
    });
}

function remove(index: number) {
    networkList.value.splice(index, 1);
    applyToYAML();
}

function applyToYAML() {
    if (editorFocus.value) {
        return;
    }

    jsonConfig.networks = {};

    // Internal networks
    for (const networkRow of networkList.value) {
        jsonConfig.networks[networkRow.key] = networkRow.value;
    }

    // External networks
    for (const networkName in externalList) {
        jsonConfig.networks[networkName] = externalList[networkName];
    }

    console.debug("applyToYAML", jsonConfig.networks);
}

watch(() => jsonConfig.networks, () => {
    if (editorFocus.value) {
        console.debug("jsonConfig.networks changed");
        loadNetworkList();
    }
}, { deep: true });

watch(selectedExternalList, () => {
    for (const networkName in selectedExternalList) {
        const enable = selectedExternalList[networkName];

        if (enable) {
            if (!externalList[networkName]) {
                externalList[networkName] = {};
            }
            externalList[networkName].external = true;
        } else {
            delete externalList[networkName];
        }
    }
    applyToYAML();
}, { deep: true });

watch(networkList, () => {
    applyToYAML();
}, { deep: true });

onMounted(() => {
    loadNetworkList();
    loadExternalNetworkList();
});
</script>

<style lang="scss" scoped>
@import "../styles/vars.scss";

.list-group {
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

.delete {
    text-decoration: underline;
    font-size: 13px;
    cursor: pointer;
}
</style>
