<template>
    <router-link :to="{ name: 'imageDetail', params: { imageRef: routeRef } }" class="item" :title="displayName">
        <span class="badge rounded-pill me-2" :class="badgeClass">{{ badgeLabel }}</span>
        <div class="title">
            <span class="me-2">{{ displayName }}</span>
        </div>
    </router-link>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";

const { t } = useI18n();

const props = defineProps<{
    image: Record<string, any>;
}>();

const isDangling = computed(() => props.image.dangling === true);

const displayName = computed(() => {
    if (props.image.repoTags && props.image.repoTags.length > 0) {
        return props.image.repoTags[0];
    }
    // Show truncated sha256 ID for dangling images
    const id = props.image.id || "";
    if (id.startsWith("sha256:")) {
        return id.substring(0, 19);
    }
    return id.substring(0, 12) || "<none>";
});

// Use image ID for dangling images (no tag to route with)
const routeRef = computed(() => {
    if (isDangling.value) {
        return props.image.id || "";
    }
    if (props.image.repoTags && props.image.repoTags.length > 0) {
        return props.image.repoTags[0];
    }
    return props.image.id || "";
});

const inUse = computed(() => (props.image.containers ?? 0) > 0);

const badgeClass = computed(() => {
    if (isDangling.value) return "bg-secondary";
    return inUse.value ? "bg-success" : "bg-warning";
});
const badgeLabel = computed(() => {
    if (isDangling.value) return t("imageDangling");
    return inUse.value ? t("imageInUse") : t("imageUnused");
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
</style>
