<template>
    <transition name="slide-fade" appear>
        <div>
            <h1 class="mb-3">{{ $t("console") }}</h1>

            <Terminal class="terminal" :rows="20" mode="interactive"
                name="console" channel="terminal" terminal-type="console" />
        </div>
    </transition>
</template>

<script setup lang="ts">
</script>

<style scoped lang="scss">
.terminal {
    height: 410px;
}
</style>
