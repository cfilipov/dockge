<template>
    <div class="toast-body-content">
        <button
            class="copy-btn"
            @click.stop="copy"
            :title="copied ? 'Copied!' : 'Copy to clipboard'"
        >
            <font-awesome-icon :icon="copied ? 'check' : 'copy'" />
        </button>
        <span>{{ message }}</span>
    </div>
</template>

<script setup lang="ts">
import { ref } from "vue";

const props = defineProps<{
    message: string;
}>();

const copied = ref(false);

function copy() {
    navigator.clipboard.writeText(props.message);
    copied.value = true;
    setTimeout(() => {
        copied.value = false;
    }, 2000);
}
</script>

<style scoped lang="scss">
.toast-body-content {
    display: flex;
    align-items: start;
    gap: 8px;
}

.copy-btn {
    background: none;
    border: none;
    color: inherit;
    opacity: 0.5;
    padding: 0;
    cursor: pointer;
    font-size: 0.85em;
    flex-shrink: 0;
    margin-top: 2px;

    &:hover {
        opacity: 1;
    }
}
</style>
