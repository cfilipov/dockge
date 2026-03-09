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
import { useViewMode } from "../composables/useViewMode";

const { t } = useI18n();
const route = useRoute();
const { getContainersSubView } = useViewMode("containers");

const props = defineProps<{
    container: Record<string, any>;
}>();

const itemLink = computed(() => {
    const name = props.container.name;
    const subView = getContainersSubView();
    if (subView === "raw") {
        return { name: "containerRaw", params: { containerName: name } };
    }
    if (subView === "logs") {
        return { name: "containerLogs", params: { containerName: name } };
    }
    if (subView === "shell") {
        const type = (route.params.type as string) || "bash";
        return { name: "containerShell", params: { containerName: name, type } };
    }
    return { name: "containerDetail", params: { containerName: name } };
});

const tooltip = computed(() => {
    const name = props.container.name;
    const subView = getContainersSubView();
    if (subView === "logs") return t("tooltipContainerLogs", [name]);
    if (subView === "shell") return t("tooltipContainerShell", [name]);
    return t("tooltipContainerInspect", [name]);
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
