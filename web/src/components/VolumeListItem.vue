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
@import "../styles/vars.scss";

.item {
    text-decoration: none;
    color: inherit;
    display: flex;
    align-items: center;
    min-height: 46px;
    border-radius: 10px;
    transition: none;
    width: 100%;
    padding: 5px 8px;
    margin: 3px 0;
    overflow: hidden;
    min-width: 0;
    &:hover {
        background-color: $highlight-white;
    }
    &.active {
        background-color: $highlight-white;
        border-left: 4px solid $primary;
        border-top-left-radius: 0;
        border-bottom-left-radius: 0;
    }
    .title {
        margin-top: -4px;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }
}

.badge {
    white-space: nowrap;
}
</style>
