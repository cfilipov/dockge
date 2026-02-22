<template>
    <div>
        <div v-if="settingsLoaded" class="my-4">
            <form class="my-4" autocomplete="off" @submit.prevent="saveGeneral">
                <div class="shadow-box mb-3 editor-box edit-mode">
                    <code-mirror
                        ref="editor"
                        v-model="settings.globalENV"
                        :extensions="extensionsEnv"
                        minimal
                        :wrap="true"
                        :dark="true"
                        :tab="true"
                        :hasFocus="editorFocus"
                        @change="onChange"
                    />
                </div>

                <div class="my-4">
                    <div>
                        <button class="btn btn-primary" type="submit">
                            {{ $t("Save") }}
                        </button>
                    </div>
                </div>
            </form>
        </div>
    </div>
</template>

<script setup lang="ts">
import { ref, inject, type Ref } from "vue";
import CodeMirror from "vue-codemirror6";
import { python } from "@codemirror/lang-python";
import { dracula as editorTheme } from "thememirror";
import { lineNumbers, EditorView } from "@codemirror/view";

const settings = inject<Ref<Record<string, any>>>("settings")!;
const saveSettings = inject<(callback?: () => void, currentPassword?: string) => void>("saveSettings")!;
const settingsLoaded = inject<Ref<boolean>>("settingsLoaded")!;

const editorFocus = ref(false);

const focusEffectHandler = (state: any, focusing: boolean) => {
    editorFocus.value = focusing;
    return null;
};

const extensionsEnv = [
    editorTheme,
    python(),
    lineNumbers(),
    EditorView.focusChangeEffect.of(focusEffectHandler),
];

function saveGeneral() {
    saveSettings();
}

function onChange() {
    // hook for future live validation if desired
}
</script>

<style scoped lang="scss">
.editor-box {
    font-family: 'JetBrains Mono', monospace;
    font-size: 14px;

    &.edit-mode {
        background-color: #2c2f38 !important;
    }
}
</style>
