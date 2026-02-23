<template>
    <div class="container mt-3">
        <h1 class="mb-3">{{ title }}</h1>
        <div class="shadow-box big-padding">
            <p class="text-muted mb-0">Coming soon.</p>
        </div>
    </div>
</template>

<script setup lang="ts">
defineProps<{
    title: string;
}>();
</script>
