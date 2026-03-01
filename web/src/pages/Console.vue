<template>
    <transition name="slide-fade" appear>
        <div v-if="!processing">
            <h1 class="mb-3">{{ $t("console") }}</h1>

            <Terminal v-if="enableConsole" class="terminal" :rows="20" mode="interactive" :mainTerminal="true" name="console"></Terminal>

            <div v-else class="alert alert-warning shadow-box" role="alert">
                <h4 class="alert-heading">{{ $t("Console is not enabled") }}</h4>
                <p v-html="$t('ConsoleNotEnabledMSG1')"></p>
                <p v-html="$t('ConsoleNotEnabledMSG2')"></p>
                <p v-html="$t('ConsoleNotEnabledMSG3')"></p>
            </div>
        </div>
    </transition>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import { useSocket } from "../composables/useSocket";

const { emit } = useSocket();

const processing = ref(true);
const enableConsole = ref(false);

onMounted(() => {
    emit("checkMainTerminal", (res: any) => {
        enableConsole.value = res.ok;
        processing.value = false;
    });
});
</script>

<style scoped lang="scss">
.terminal {
    height: 410px;
}
</style>
