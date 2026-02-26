<template>
    <div>
        <h4 class="mb-3 collapsible-heading" @click="expanded = !expanded">
            <font-awesome-icon
                icon="chevron-down"
                class="chevron-icon"
                :class="{ collapsed: !expanded }"
            />
            <slot name="heading" />
        </h4>
        <div v-show="expanded">
            <slot />
        </div>
    </div>
</template>

<script setup lang="ts">
import { ref } from "vue";

const expanded = ref(true);
</script>

<style scoped lang="scss">
@import "../styles/vars.scss";

.collapsible-heading {
    cursor: pointer;
    user-select: none;

    &:hover {
        opacity: 0.8;
    }
}

.chevron-icon {
    font-size: 0.7em;
    margin-right: 0.4rem;
    transition: transform 0.2s $easing-in;
}

.collapsed {
    transform: rotate(-90deg);
}

.collapsible-heading :deep(.section-count) {
    color: $dark-font-color3;
}
</style>
