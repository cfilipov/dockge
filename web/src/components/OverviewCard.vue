<template>
    <h4 class="mb-3">{{ $t("containerOverview") }}</h4>
    <div v-if="data" class="shadow-box big-padding mb-3">
        <div class="overview-list" role="region" :aria-label="$t('containerOverview')">
            <slot />
        </div>
    </div>
    <div v-else class="shadow-box big-padding mb-3">
        <p class="text-muted mb-0">{{ fallbackText || (loading ? "Loading..." : "") }}</p>
    </div>
</template>

<script setup lang="ts">
defineProps<{
    data: any;
    loading?: boolean;
    fallbackText?: string;
}>();
</script>

<style scoped lang="scss">
@import "../styles/info-chips";
</style>
