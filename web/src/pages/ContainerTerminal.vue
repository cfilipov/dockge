<template>
    <transition name="slide-fade" appear>
        <div>
            <h1 class="mb-3">{{$t("terminal")}} - {{ serviceName }} ({{ stackName }})</h1>

            <div class="mb-3">
                <router-link :to="sh" class="btn btn-normal me-2">{{ $t("Switch to sh") }}</router-link>
            </div>

            <Terminal class="terminal" :rows="20" mode="interactive" :name="terminalName" :stack-name="stackName" :service-name="serviceName" :shell="shell" :endpoint="endpoint"></Terminal>
        </div>
    </transition>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useRoute } from "vue-router";
import { getContainerExecTerminalName } from "../common/util-common";

const route = useRoute();

const stackName = computed(() => route.params.stackName as string);
const endpoint = computed(() => (route.params.endpoint as string) || "");
const shell = computed(() => route.params.type as string);
const serviceName = computed(() => route.params.serviceName as string);
const terminalName = computed(() => getContainerExecTerminalName(endpoint.value, stackName.value, serviceName.value, 0));
const sh = computed(() => {
    const ep = route.params.endpoint;
    const data: any = {
        name: "containerTerminal",
        params: {
            stackName: stackName.value,
            serviceName: serviceName.value,
            type: "sh",
        },
    };
    if (ep) {
        data.name = "containerTerminalEndpoint";
        data.params.endpoint = ep;
    }
    return data;
});
</script>

<style scoped lang="scss">
.terminal {
    height: 410px;
}
</style>
