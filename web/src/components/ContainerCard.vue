<template>
    <div class="shadow-box big-padding mb-3" role="region" :aria-label="container.name">
        <h5 class="mb-3">
            <span class="badge rounded-pill me-2" :class="'bg-' + statusInfo.badgeColor">{{ $t(statusInfo.label) }}</span>
            <router-link :to="{ name: 'containerDetail', params: { containerName: container.name } }" class="stack-link">{{ container.name }}</router-link>
        </h5>
        <div class="info-chips">
            <div class="info-chip">
                <span class="chip-label">{{ $t("containerID") }}</span>
                <code :title="container.containerId">{{ container.containerId.substring(0, 12) }}</code>
            </div>
            <slot />
        </div>
    </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { ContainerStatusInfo } from "../common/util-common";
import type { ContainerBroadcast } from "../stores/containerStore";

const props = defineProps<{
    container: ContainerBroadcast;
}>();

const statusInfo = computed(() => ContainerStatusInfo.from(props.container));
</script>

<style scoped lang="scss">
@import "../styles/info-chips";
</style>
