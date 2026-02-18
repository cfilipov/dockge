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

<script>
import { defineComponent } from "vue";
import { PROGRESS_TERMINAL_ROWS } from "../../../common/util-common";

export default defineComponent({
    props: {
        name: {
            type: String,
            required: true
        },
        endpoint: {
            type: String,
            required: true
        },
        rows: {
            type: Number,
            default: PROGRESS_TERMINAL_ROWS
        },
        autoHideTimeout: {
            type: Number,
            default: 10000
        }
    },

    data() {
        return {
            showProgressTerminal: false,
            autoHideTimer: null,
        };
    },

    methods: {
        show() {
            const term = this.$refs.progressTerminal;
            if (term) {
                term.bind(this.endpoint, this.name);
                if (term.terminal) {
                    term.terminal.clear();
                }
            }
            this.showProgressTerminal = true;
            clearTimeout(this.autoHideTimer);
        },

        hideWithTimeout() {
            if (this.autoHideTimeout > 0) {
                this.autoHideTimer = setTimeout(() => {
                    this.showProgressTerminal = false;
                }, this.autoHideTimeout);
            }
        },
    }
});
</script>

<style lang="scss" scoped>
.progress-terminal-header {
    cursor: pointer;
    user-select: none;
}
</style>
