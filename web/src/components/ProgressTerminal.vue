<template>
    <transition name="slide-fade" appear>
        <div v-if="visible" class="progress-terminal position-relative" role="region" aria-label="Progress">
            <button class="dismiss-button" :title="$t('Close')" @click="hide">
                <font-awesome-icon icon="times" />
            </button>
            <Terminal
                ref="progressTerminal"
                :name="terminalName"
                :rows="rows"
                :terminal-type="terminalType"
                :terminal-params="terminalParams"
            ></Terminal>
        </div>
    </transition>
</template>

<script setup lang="ts">
import { ref, computed } from "vue";
import { PROGRESS_TERMINAL_ROWS } from "../common/util-common";

const props = withDefaults(defineProps<{
    terminalType: string;
    terminalParams: Record<string, string>;
    rows?: number;
}>(), {
    rows: PROGRESS_TERMINAL_ROWS,
});

const visible = ref(false);

const terminalName = computed(() => {
    if (props.terminalType === "compose") {
        return "compose-" + (props.terminalParams.stack || "");
    }
    if (props.terminalType === "container-action") {
        return "container-" + (props.terminalParams.container || "");
    }
    return props.terminalType;
});

function show() {
    visible.value = true;
}

function hide() {
    visible.value = false;
}

defineExpose({ show, hide });
</script>

<style lang="scss" scoped>
.dismiss-button {
    all: unset;
    position: absolute;
    right: 15px;
    top: 15px;
    z-index: 10;
    cursor: pointer;

    svg {
        width: 20px;
        height: 20px;
    }

    .dark &:hover {
        color: white;
    }
}
</style>
