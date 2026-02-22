<template>
    <div class="shadow-box">
        <div class="progress-terminal-header mb-1" @click="showProgressTerminal = !showProgressTerminal">
            <font-awesome-icon :icon="showProgressTerminal ? 'chevron-down' : 'chevron-right'" class="me-2" />
            {{ $t("terminal") }}
        </div>
        <transition name="slide-fade" appear>
            <Terminal
                v-show="showProgressTerminal"
                ref="progressTerminal"
                class="terminal"
                :name="name"
                :endpoint="endpoint"
                :rows="rows"
            ></Terminal>
        </transition>
    </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { PROGRESS_TERMINAL_ROWS } from "../../../common/util-common";

const props = withDefaults(defineProps<{
    name: string;
    endpoint: string;
    rows?: number;
    autoHideTimeout?: number;
}>(), {
    rows: PROGRESS_TERMINAL_ROWS,
    autoHideTimeout: 10000,
});

const showProgressTerminal = ref(false);
let autoHideTimer: ReturnType<typeof setTimeout> | null = null;
const progressTerminal = ref<InstanceType<any>>();

function show() {
    const term = progressTerminal.value;
    if (term) {
        term.bind(props.endpoint, props.name);
        if (term.terminal) {
            term.terminal.clear();
        }
    }
    showProgressTerminal.value = true;
    if (autoHideTimer) {
        clearTimeout(autoHideTimer);
    }
}

function hideWithTimeout() {
    if (props.autoHideTimeout > 0) {
        autoHideTimer = setTimeout(() => {
            showProgressTerminal.value = false;
        }, props.autoHideTimeout);
    }
}

defineExpose({ show, hideWithTimeout });
</script>

<style lang="scss" scoped>
.progress-terminal-header {
    cursor: pointer;
    user-select: none;
}
</style>
