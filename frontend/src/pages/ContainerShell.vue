<template>
    <transition name="slide-fade" appear>
        <div v-if="containerName">
            <h1 class="mb-3">{{ $t("shell") }} - {{ containerName }}</h1>

            <div class="mb-3">
                <router-link :to="switchShellLink" class="btn btn-normal me-2">{{ $t(switchShellLabel) }}</router-link>
            </div>

            <Terminal class="terminal" :rows="20" mode="interactive"
                :name="terminalName" :endpoint="endpoint"
                :container-name="containerName" :shell="shell" />
        </div>
        <div v-else>
            <h1 class="mb-3">{{ $t("shell") }}</h1>
            <div class="shadow-box big-padding">
                <p class="text-muted mb-0">{{ $t("selectContainer") }}</p>
            </div>
        </div>
    </transition>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useRoute } from "vue-router";

const route = useRoute();

const containerName = computed(() => route.params.containerName as string || "");
const endpoint = computed(() => "");
const shell = computed(() => (route.params.type as string) || "bash");
const terminalName = computed(() => "container-exec-by-name--" + containerName.value);

const alternateShell = computed(() => shell.value === "bash" ? "sh" : "bash");
const switchShellLabel = computed(() => shell.value === "bash" ? "Switch to sh" : "Switch to bash");
const switchShellLink = computed(() => ({
    name: "containerShell",
    params: {
        containerName: containerName.value,
        type: alternateShell.value,
    },
}));
</script>

<style scoped lang="scss">
.terminal {
    height: 410px;
}
</style>
