<template>
    <span :class="className">{{ statusName }}</span>
</template>

<script>
import { StackStatusInfo } from "../../../common/util-common";

export default {
    props: {
        stack: {
            type: Object,
            default: null,
        },
        fixedWidth: {
            type: Boolean,
            default: false,
        },
    },

    computed: {
        statusInfo() {
            return StackStatusInfo.get(this.stack?.status);
        },

        uptime() {
            return this.$t("notAvailableShort");
        },

        color() {
            return this.statusInfo.badgeColor;
        },

        statusName() {
            return this.$t(this.statusInfo.label);
        },

        className() {
            let className = `badge rounded-pill bg-${this.color}`;

            if (this.fixedWidth) {
                className += " fixed-width";
            }
            return className;
        },
    },
};
</script>

<style scoped>
.badge {
    min-width: 62px;

}

.fixed-width {
    width: 62px;
    overflow: hidden;
    text-overflow: ellipsis;
}
</style>
