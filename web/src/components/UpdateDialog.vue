<template>
    <BModal v-model="visible" :title="$t('updateStack')" :close-on-esc="true" @show="onShow" @hidden="onHidden">
        <p class="mb-3" v-html="$t('updateStackMsg')"></p>

        <div v-if="changelogLinks.length > 0" class="mb-3">
            <h5>{{ $t("changelog") }}</h5>
            <div v-for="link in changelogLinks" :key="link.service">
                <strong>{{ link.service }}:</strong>{{ " " }}
                <a :href="link.url" target="_blank">{{ link.url }}</a>
            </div>
        </div>

        <BForm>
            <BFormCheckbox v-model="pruneAfterUpdate" switch><span v-html="$t('pruneAfterUpdate')"></span></BFormCheckbox>
            <div style="margin-left: 2.5rem;">
                <BFormCheckbox v-model="pruneAllAfterUpdate" :checked="pruneAfterUpdate && pruneAllAfterUpdate" :disabled="!pruneAfterUpdate"><span v-html="$t('pruneAllAfterUpdate')"></span></BFormCheckbox>
            </div>
        </BForm>

        <template #footer>
            <button v-if="showIgnore" class="btn btn-normal" :title="$t('tooltipServiceUpdateIgnore')" @click="doIgnore">
                <font-awesome-icon icon="ban" class="me-1" />{{ $t("ignoreUpdate") }}
            </button>
            <button class="btn btn-primary" @click="doUpdate">
                <font-awesome-icon icon="cloud-arrow-down" class="me-1" />{{ $t("updateStack") }}
            </button>
        </template>
    </BModal>
</template>

<script setup lang="ts">
import { ref, computed } from "vue";
import { BModal, BForm, BFormCheckbox } from "bootstrap-vue-next";
import { FontAwesomeIcon } from "@fortawesome/vue-fontawesome";
import { LABEL_IMAGEUPDATES_CHANGELOG } from "../common/compose-labels";
import { useSocket } from "../composables/useSocket";
import yamlLib from "yaml";

const props = withDefaults(defineProps<{
    modelValue: boolean;
    stackName: string;
    endpoint: string;
    serviceName?: string;
    composeYaml?: string;
    showIgnore?: boolean;
}>(), {
    showIgnore: false,
});

const emit = defineEmits<{
    (e: "update:modelValue", value: boolean): void;
    (e: "update", data: { pruneAfterUpdate: boolean; pruneAllAfterUpdate: boolean }): void;
    (e: "ignore"): void;
}>();

const { emitAgent } = useSocket();

const visible = computed({
    get: () => props.modelValue,
    set: (val: boolean) => emit("update:modelValue", val),
});

const pruneAfterUpdate = ref(false);
const pruneAllAfterUpdate = ref(false);
const fetchedYAML = ref("");

const effectiveYAML = computed(() => props.composeYaml || fetchedYAML.value);

const changelogLinks = computed(() => {
    const yaml = effectiveYAML.value;
    if (!yaml) return [];

    try {
        const doc = yamlLib.parse(yaml);
        const services = doc?.services;
        if (!services) return [];

        const links: { service: string; url: string }[] = [];

        for (const [name, svc] of Object.entries(services) as [string, any][]) {
            if (props.serviceName && name !== props.serviceName) continue;
            const url = getLabelValue(svc?.labels, LABEL_IMAGEUPDATES_CHANGELOG);
            if (url) {
                links.push({ service: name, url });
            }
        }

        return links;
    } catch {
        return [];
    }
});

function getLabelValue(labels: any, key: string): string {
    if (!labels) return "";
    if (Array.isArray(labels)) {
        const prefix = key + "=";
        const entry = labels.find((l: string) => typeof l === "string" && l.startsWith(prefix));
        return entry ? entry.substring(prefix.length) : "";
    }
    return labels[key] || "";
}

function onShow() {
    pruneAfterUpdate.value = false;
    pruneAllAfterUpdate.value = false;
    fetchedYAML.value = "";

    if (!props.composeYaml && props.stackName) {
        emitAgent(props.endpoint, "getStack", props.stackName, (res: any) => {
            if (res.ok && res.stack) {
                fetchedYAML.value = res.stack.composeYAML || "";
            }
        });
    }
}

function onHidden() {
    pruneAfterUpdate.value = false;
    pruneAllAfterUpdate.value = false;
    fetchedYAML.value = "";
}

function doUpdate() {
    emit("update:modelValue", false);
    emit("update", {
        pruneAfterUpdate: pruneAfterUpdate.value,
        pruneAllAfterUpdate: pruneAllAfterUpdate.value,
    });
}

function doIgnore() {
    emit("update:modelValue", false);
    emit("ignore");
}
</script>
