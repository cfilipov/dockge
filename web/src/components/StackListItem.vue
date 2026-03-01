<template>
    <router-link :to="url" :class="{ 'dim' : !stack.isManagedByDockge }" class="item">
        <Uptime :stack="stack" class="me-2" />
        <div class="title">
            <span class="me-2">{{ stackName }}</span>
            <font-awesome-icon v-if="stack.started && stack.recreateNecessary" icon="rocket" class="notification-icon me-2" :title="$t('tooltipIconRecreate')" />
            <font-awesome-icon v-if="stack.imageUpdatesAvailable" icon="arrow-up" class="notification-icon me-2" :title="$t('tooltipIconUpdate')" />
        </div>
    </router-link>
</template>

<script setup lang="ts">
import { ref, computed } from "vue";

const props = withDefaults(defineProps<{
    stack: Record<string, any>;
    isSelectMode?: boolean;
    depth?: number;
    isSelected?: (id: any) => boolean;
    select?: (id: any) => void;
    deselect?: (id: any) => void;
}>(), {
    isSelectMode: false,
    depth: 0,
    isSelected: () => false,
    select: () => {},
    deselect: () => {},
});

const isCollapsed = ref(true);

const url = computed(() => `/stacks/${props.stack.name}`);

const depthMargin = computed(() => ({
    marginLeft: `${31 * props.depth}px`,
}));

const stackName = computed(() => props.stack.name);

function changeCollapsed() {
    isCollapsed.value = !isCollapsed.value;
    let storage = window.localStorage.getItem("stackCollapsed");
    let storageObject: Record<string, any> = {};
    if (storage !== null) {
        storageObject = JSON.parse(storage);
    }
    storageObject[`stack_${props.stack.id}`] = isCollapsed.value;
    window.localStorage.setItem("stackCollapsed", JSON.stringify(storageObject));
}

function toggleSelection() {
    if (props.isSelected(props.stack.id)) {
        props.deselect(props.stack.id);
    } else {
        props.select(props.stack.id);
    }
}
</script>

<style lang="scss" scoped>
@import "../styles/list-item";

.small-padding {
    padding-left: 5px !important;
    padding-right: 5px !important;
}

.collapse-padding {
    padding-left: 8px !important;
    padding-right: 2px !important;
}

.item {
    &.disabled {
        opacity: 0.3;
    }

}

.collapsed {
    transform: rotate(-90deg);
}

.animated {
    transition: all 0.2s $easing-in;
}

.select-input-wrapper {
    float: left;
    margin-top: 15px;
    margin-left: 3px;
    margin-right: 10px;
    padding-left: 4px;
    position: relative;
    z-index: 15;
}

.dim {
    opacity: 0.5;
}

.notification-icon {
    color: $info;
    font-weight: bold;
}

</style>
