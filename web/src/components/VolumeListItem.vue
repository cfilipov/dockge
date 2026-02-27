<template>
    <router-link :to="{ name: 'volumeDetail', params: { volumeName: volume.name } }" class="item" :title="volume.name">
        <span class="badge rounded-pill me-2" :class="badgeClass">{{ badgeLabel }}</span>
        <div class="title">
            <span class="me-2">{{ volume.name }}</span>
        </div>
    </router-link>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";

const { t } = useI18n();

const props = defineProps<{
    volume: Record<string, any>;
}>();

const inUse = computed(() => (props.volume.containers ?? 0) > 0);

const badgeClass = computed(() => {
    return inUse.value ? "bg-success" : "bg-warning";
});
const badgeLabel = computed(() => {
    return inUse.value ? t("volumeInUse") : t("volumeUnused");
});
</script>

<style lang="scss" scoped>
@import "../styles/list-item";
</style>
