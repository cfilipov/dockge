<template>
    <router-link :to="itemLink" :class="{ 'dim' : !container.isManagedByDockge }" class="item" :title="tooltip">
        <span :class="badgeClass" class="badge rounded-pill me-2">{{ statusLabel }}</span>
        <div class="title">
            <span class="me-2">{{ container.name }}</span>
            <font-awesome-icon v-if="container.recreateNecessary" icon="rocket" class="notification-icon me-2" :title="$t('tooltipIconRecreate')" />
            <font-awesome-icon v-if="container.imageUpdatesAvailable" icon="arrow-up" class="notification-icon me-2" :title="$t('tooltipIconUpdate')" />
        </div>
    </router-link>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import { ContainerStatusInfo } from "../common/util-common";

const { t } = useI18n();
const route = useRoute();

const props = defineProps<{
    container: Record<string, any>;
}>();

const currentTab = computed(() => {
    if (route.path.startsWith("/logs")) return "logs";
    if (route.path.startsWith("/containers")) return "containers";
    return "shell";
});

const itemLink = computed(() => {
    const name = props.container.name;
    if (currentTab.value === "logs") {
        return { name: "containerLogs", params: { containerName: name } };
    }
    if (currentTab.value === "containers") {
        return { name: "containerDetail", params: { containerName: name } };
    }
    return { name: "containerShell", params: { containerName: name, type: "bash" } };
});

const tooltip = computed(() => {
    const name = props.container.name;
    if (currentTab.value === "logs") return t("tooltipContainerLogs", [name]);
    if (currentTab.value === "containers") return t("tooltipContainerInspect", [name]);
    return t("tooltipContainerShell", [name]);
});

const statusInfo = computed(() => ContainerStatusInfo.from(props.container));

const badgeClass = computed(() => `bg-${statusInfo.value.badgeColor}`);
const statusLabel = computed(() => t(statusInfo.value.label));
</script>

<style lang="scss" scoped>
@import "../styles/list-item";

.item .title {
    flex: 1;
    min-width: 0;
}

.dim {
    opacity: 0.5;
}

.notification-icon {
    color: $info;
    font-weight: bold;
}
</style>
