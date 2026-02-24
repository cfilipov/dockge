<template>
    <router-link :to="{ name: 'networkDetail', params: { networkName: network.name } }" class="item" :title="network.name">
        <span class="badge rounded-pill me-2" :class="badgeClass">{{ badgeLabel }}</span>
        <div class="title">
            <span class="me-2">{{ network.name }}</span>
        </div>
    </router-link>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";

const { t } = useI18n();

const props = defineProps<{
    network: Record<string, any>;
}>();

const inUse = computed(() => (props.network.containers ?? 0) > 0);

const badgeClass = computed(() => inUse.value ? "bg-success" : "bg-warning");
const badgeLabel = computed(() => inUse.value ? t("networkInUse") : t("networkUnused"));
</script>

<style lang="scss" scoped>
@import "../styles/vars.scss";

.item {
    text-decoration: none;
    color: inherit;
    display: flex;
    align-items: center;
    min-height: 52px;
    border-radius: 10px;
    transition: all ease-in-out 0.15s;
    width: 100%;
    padding: 5px 8px;
    &:hover {
        background-color: $highlight-white;
    }
    &.active {
        background-color: $highlight-white;
    }
    .title {
        margin-top: -4px;
    }
}

.badge {
    min-width: 62px;
    width: 62px;
    overflow: hidden;
    text-overflow: ellipsis;
}
</style>
