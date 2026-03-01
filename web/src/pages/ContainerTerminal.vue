<template>
    <transition name="slide-fade" appear>
        <div>
            <h1 class="mb-3">{{$t("terminal")}} - {{ serviceName }} ({{ stackName }})</h1>

            <div class="mb-3">
                <router-link :to="sh" class="btn btn-normal me-2">{{ $t("Switch to sh") }}</router-link>
            </div>

            <Terminal class="terminal" :rows="20" mode="interactive" :name="terminalName" :stack-name="stackName" :service-name="serviceName" :shell="shell"></Terminal>
        </div>
    </transition>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useRoute } from "vue-router";
import { getContainerExecTerminalName } from "../common/util-common";

const route = useRoute();

const stackName = computed(() => route.params.stackName as string);
const shell = computed(() => route.params.type as string);
const serviceName = computed(() => route.params.serviceName as string);
const terminalName = computed(() => getContainerExecTerminalName(stackName.value, serviceName.value, 0));
const sh = computed(() => ({
    name: "containerTerminal",
    params: {
        stackName: stackName.value,
        serviceName: serviceName.value,
        type: "sh",
    },
}));
</script>

<style scoped lang="scss">
.terminal {
    height: 410px;
}
</style>
