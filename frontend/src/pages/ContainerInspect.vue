<template>
    <transition name="slide-fade" appear>
        <div>
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
    </transition>
</template>

<script>
import CodeMirror from "vue-codemirror6";
import { yaml as yamlLang } from "@codemirror/lang-yaml";
import { dracula as editorTheme } from "thememirror";
import { lineNumbers } from "@codemirror/view";
import yaml from "yaml";

export default {
    components: {
        CodeMirror,
    },
    data() {
        return {
            inspectData: "fetching ...",
        };
    },
    computed: {
        stackName() {
            return this.$route.params.stackName;
        },
        endpoint() {
            return this.$route.params.endpoint || "";
        },
        containerName() {
            return this.$route.params.containerName;
        },
    },
    setup() {
        const extensionsYAML = [
            editorTheme,
            yamlLang(),
            lineNumbers(),
        ];

        return { extensionsYAML };
    },
    mounted() {
        this.$root.emitAgent(this.endpoint, "containerInspect", this.containerName, (res) => {
            if (res.ok) {
                const inspectObj = JSON.parse(res.inspectData);
                if (inspectObj) {
                    this.inspectData = yaml.stringify(inspectObj, { lineWidth: 0 });
                }
            }
        });
    },
    methods: {
    }
};
</script>

<style scoped lang="scss">

.editor-box {
    font-family: 'JetBrains Mono', monospace;
    font-size: 14px;
    height: 500px;
}

</style>
