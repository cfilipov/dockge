<template>
    <transition name="slide-fade" appear>
        <div v-show="visible" class="progress-terminal position-relative">
            <button class="dismiss-button" :title="$t('Close')" @click="hide">
                <font-awesome-icon icon="times" />
            </button>
            <Terminal
                ref="progressTerminal"
                :name="name"
                :endpoint="endpoint"
                :rows="rows"
            ></Terminal>
        </div>
    </transition>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { PROGRESS_TERMINAL_ROWS } from "../common/util-common";

const props = withDefaults(defineProps<{
    name: string;
    endpoint: string;
    rows?: number;
}>(), {
    rows: PROGRESS_TERMINAL_ROWS,
});

const visible = ref(false);
const progressTerminal = ref<InstanceType<any>>();

function show() {
    const term = progressTerminal.value;
    if (term) {
        term.bind(props.endpoint, props.name);
        if (term.terminal) {
            term.terminal.clear();
        }
    }
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
