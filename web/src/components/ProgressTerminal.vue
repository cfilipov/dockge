<template>
    <transition name="slide-fade" appear>
        <div v-show="visible" class="progress-terminal position-relative" role="region" aria-label="Progress">
            <button class="dismiss-button" :title="$t('Close')" @click="hide">
                <font-awesome-icon icon="times" />
            </button>
            <Terminal
                ref="progressTerminal"
                :name="name"
                :rows="rows"
                @has-buffer="autoShow"
            ></Terminal>
        </div>
    </transition>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import { PROGRESS_TERMINAL_ROWS } from "../common/util-common";

const props = withDefaults(defineProps<{
    name: string;
    rows?: number;
}>(), {
    rows: PROGRESS_TERMINAL_ROWS,
});

const visible = ref(false);
const progressTerminal = ref<InstanceType<any>>();

// Hide when the terminal name changes (Vue Router reuses the component
// on navigation). autoShow will re-show if the new name has buffered content.
watch(() => props.name, () => {
    visible.value = false;
});

function show() {
    const term = progressTerminal.value;
    if (term) {
        term.bind(props.name);
        if (term.terminal) {
            term.terminal.clear();
        }
    }
    visible.value = true;
}

function hide() {
    visible.value = false;
}

function autoShow() {
    visible.value = true;
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
