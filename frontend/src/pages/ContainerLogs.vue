<template>
    <transition name="slide-fade" appear>
        <div v-if="containerName">
            <h1 class="mb-3">{{ $t("logs") }} - {{ containerName }}</h1>

            <Terminal class="terminal" :rows="20" mode="displayOnly"
                :name="terminalName" :endpoint="endpoint" />
        </div>
        <div v-else>
            <h1 class="mb-3">{{ $t("logs") }}</h1>
            <div class="shadow-box big-padding">
                <p class="text-muted mb-0">{{ $t("selectContainer") }}</p>
            </div>
        </div>
    </transition>
</template>

<script setup lang="ts">
import { computed, onMounted, watch } from "vue";
import { useRoute } from "vue-router";
import { useSocket } from "../composables/useSocket";

const route = useRoute();
const { emitAgent } = useSocket();

const containerName = computed(() => route.params.containerName as string || "");
const endpoint = computed(() => "");
const terminalName = computed(() => "container-log-by-name--" + containerName.value);

onMounted(() => {
    if (containerName.value) {
        emitAgent(endpoint.value, "joinContainerLogByName", containerName.value, () => {});
    }
});
</script>

<style scoped lang="scss">
.terminal {
    height: 410px;
}
</style>
