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

const { t } = useI18n();
const route = useRoute();

const props = defineProps<{
    container: Record<string, any>;
}>();

const isLogsTab = computed(() => route.path.startsWith("/logs"));

const itemLink = computed(() => {
    if (isLogsTab.value) {
        return {
            name: "containerLogs",
            params: { containerName: props.container.name },
        };
    }
    return {
        name: "containerShell",
        params: { containerName: props.container.name, type: "bash" },
    };
});

const tooltip = computed(() => {
    if (isLogsTab.value) {
        return t("tooltipContainerLogs", [props.container.name]);
    }
    return t("tooltipContainerShell", [props.container.name]);
});

const badgeClass = computed(() => {
    const state = props.container.state;
    const health = props.container.health;

    if (state === "running") {
        if (health === "unhealthy") return "bg-danger";
        return "bg-primary";
    }
    if (state === "exited" || state === "dead") return "bg-warning";
    if (state === "paused") return "bg-info";
    return "bg-dark"; // created or unknown
});

const statusLabel = computed(() => {
    const state = props.container.state;
    const health = props.container.health;

    if (state === "running" && health === "unhealthy") return t("containerUnhealthy");
    if (state === "running") return t("containerRunning");
    if (state === "exited") return t("containerExited");
    if (state === "dead") return t("containerDead");
    if (state === "paused") return t("containerPaused");
    if (state === "created") return t("containerCreated");
    return state || "unknown";
});
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

.dim {
    opacity: 0.5;
}

.notification-icon {
    color: $info;
    font-weight: bold;
}
</style>
