<template>
    <transition name="slide-fade" appear>
        <div v-if="containerName">
            <h1 class="mb-3">{{ $t("inspect") }} - {{ containerName }}</h1>

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
import { ref, computed, onMounted } from "vue";
import { useRoute } from "vue-router";
import CodeMirror from "vue-codemirror6";
import { yaml as yamlLang } from "@codemirror/lang-yaml";
import { dracula as editorTheme } from "thememirror";
import { lineNumbers } from "@codemirror/view";
import yaml from "yaml";
import { useSocket } from "../composables/useSocket";

const route = useRoute();
const { emitAgent } = useSocket();

const inspectData = ref("fetching ...");

const extensionsYAML = [
    editorTheme,
    yamlLang(),
    lineNumbers(),
];

const endpoint = computed(() => (route.params.endpoint as string) || "");
const containerName = computed(() => route.params.containerName as string || "");

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

.editor-box {
    font-family: 'JetBrains Mono', monospace;
    font-size: 14px;
    height: 500px;
}

</style>
