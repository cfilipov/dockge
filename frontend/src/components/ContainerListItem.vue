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
import { StackStatusInfo } from "../../../common/util-common";

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

function getContainerStatusLabel(c: Record<string, any>): string {
    if (c.state === "running" && c.health === "unhealthy") return "unhealthy";
    if (c.state === "running") return "active";
    if (c.state === "exited" || c.state === "dead") return "exited";
    if (c.state === "paused") return "active";
    if (c.state === "created") return "down";
    return "down";
}

const statusInfo = computed(() => {
    const label = getContainerStatusLabel(props.container);
    return StackStatusInfo.ALL.find(i => i.label === label) ?? StackStatusInfo.ALL[0];
});

const badgeClass = computed(() => `bg-${statusInfo.value.badgeColor}`);
const statusLabel = computed(() => t(statusInfo.value.label));
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
    min-width: 62px;
    width: 62px;
    overflow: hidden;
    text-overflow: ellipsis;
}

.dim {
    opacity: 0.5;
}

.notification-icon {
    color: $info;
    font-weight: bold;
}
</style>
