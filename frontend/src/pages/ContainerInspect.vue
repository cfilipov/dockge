<template>
    <transition name="slide-fade" appear>
        <div>
            <h1 class="mb-3">{{ $t("inspect") }} - {{ containerName }}</h1>

            <div class="shadow-box mb-3 editor-box">
                <code-mirror
                    v-model="inspectData"
                    :extensions="extensionsJSON"
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
import { json } from "@codemirror/lang-json";
import { dracula as editorTheme } from "thememirror";
import { lineNumbers } from "@codemirror/view";

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
        const extensionsJSON = [
            editorTheme,
            json(),
            lineNumbers(),
        ];

        return { extensionsJSON };
    },
    mounted() {
        this.$root.emitAgent(this.endpoint, "containerInspect", this.containerName, (res) => {
            if (res.ok) {
                const inspectObj = JSON.parse(res.inspectData);
                if (inspectObj) {
                    this.inspectData = JSON.stringify(inspectObj, undefined, 2);
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
