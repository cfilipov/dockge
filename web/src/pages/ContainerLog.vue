<template>
    <transition name="slide-fade" appear>
        <div>
            <h1 class="mb-3">{{ $t("log") }} - {{ serviceName }} ({{ stackName }})</h1>

            <Terminal class="terminal" :rows="20" mode="displayOnly" :name="terminalName" :stack-name="stackName" :service-name="serviceName"></Terminal>
        </div>
    </transition>
</template>

<script setup lang="ts">
import { computed, onMounted } from "vue";
import { useRoute } from "vue-router";
import { getContainerLogName } from "../common/util-common";
import { useSocket } from "../composables/useSocket";

const route = useRoute();
const { emit } = useSocket();

const stackName = computed(() => route.params.stackName as string);
const serviceName = computed(() => route.params.serviceName as string);
const terminalName = computed(() => getContainerLogName(stackName.value, serviceName.value));

onMounted(() => {
    emit("joinContainerLog", stackName.value, serviceName.value, () => {});
});
</script>

<style scoped lang="scss">
.terminal {
    height: 410px;
}
</style>
