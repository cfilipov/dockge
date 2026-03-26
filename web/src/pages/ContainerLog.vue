<template>
    <transition name="slide-fade" appear>
        <div>
            <h1 class="mb-3">{{ $t("log") }} - {{ serviceName }} ({{ stackName }})</h1>

            <LogView class="terminal" aria-label="Logs" :name="terminalName"
                terminal-type="container-log" :terminal-params="{ stack: stackName, service: serviceName }" />
        </div>
    </transition>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useRoute } from "vue-router";
import { getContainerLogName } from "../common/util-common";

const route = useRoute();

const stackName = computed(() => route.params.stackName as string);
const serviceName = computed(() => route.params.serviceName as string);
const terminalName = computed(() => getContainerLogName(stackName.value, serviceName.value));
</script>

<style scoped lang="scss">
.terminal {
    height: 410px;
}
</style>
