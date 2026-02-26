<template>
    <span :class="className">{{ statusName }}</span>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { StackStatusInfo } from "../common/util-common";

const { t } = useI18n();

const props = defineProps<{
    stack: Record<string, any> | null;
}>();

const statusInfo = computed(() => StackStatusInfo.get(props.stack?.status));
const color = computed(() => statusInfo.value.badgeColor);
const statusName = computed(() => t(statusInfo.value.label));

const className = computed(() => `badge rounded-pill bg-${color.value}`);
</script>

<style scoped>
.badge {
    white-space: nowrap;
}
</style>
