<template>
    <transition ref="tableContainerRef" name="slide-fade" appear>
        <div v-if="$route.name === 'DashboardHome'">
            <h1 class="mb-3">
                {{ $t("stacks") }}
            </h1>

            <div class="row first-row">
                <!-- Left -->
                <div class="col-md-7">
                    <!-- Stats -->
                    <div class="shadow-box big-padding text-center mb-4">
                        <div class="row">
                            <div class="col">
                                <h3>{{ $t("active") }}</h3>
                                <span class="num active">{{ activeNum }}</span>
                            </div>
                            <div class="col">
                                <h3>{{ $t("partially") }}</h3>
                                <span class="num partially">{{ partiallyNum }}</span>
                            </div>
                            <div class="col">
                                <h3>{{ $t("unhealthy") }}</h3>
                                <span class="num unhealthy">{{ unhealthyNum }}</span>
                            </div>
                            <div class="col">
                                <h3>{{ $t("exited") }}</h3>
                                <span class="num exited">{{ exitedNum }}</span>
                            </div>
                            <div class="col">
                                <h3>{{ $t("down") }}</h3>
                                <span class="num inactive">{{ downNum }}</span>
                            </div>
                            <div class="col">
                                <h3>{{ $t("updates") }}</h3>
                                <span class="num update-available">{{ updateAvailableNum }}</span>
                            </div>
                        </div>
                    </div>

                    <!-- Docker Run -->
                    <h2 class="mb-3">{{ $t("Docker Run") }}</h2>
                    <div class="mb-3">
                        <textarea id="name" v-model="dockerRunCommand" type="text" class="form-control docker-run shadow-box" required placeholder="docker run ..."></textarea>
                    </div>

                    <button class="btn-normal btn mb-4" @click="convertDockerRun">{{ $t("Convert to Compose") }}</button>
                </div>
            </div>
        </div>
    </transition>
    <router-view ref="child" />
</template>

<script setup lang="ts">
import { ref, computed } from "vue";
import { useRouter } from "vue-router";
import { statusNameShort } from "../common/util-common";
import { useSocket } from "../composables/useSocket";
import { useStackStore } from "../stores/stackStore";
import { useAppToast } from "../composables/useAppToast";

defineProps<{
    calculatedHeight?: number;
}>();

const router = useRouter();
const stackStore = useStackStore();
const { composeTemplate } = useSocket();
const { toastRes } = useAppToast();

const dockerRunCommand = ref("");
const tableContainerRef = ref<HTMLElement>();

const statusCounts = computed(() => {
    const counts: Record<string, number> = { active: 0, partially: 0, unhealthy: 0, down: 0, exited: 0, updateAvailable: 0 };

    for (const stack of stackStore.allStacks) {
        const short = statusNameShort(stack.status);
        if (short in counts) {
            counts[short]++;
        }
        if (stack.imageUpdatesAvailable) {
            counts.updateAvailable++;
        }
    }

    return counts;
});

const activeNum = computed(() => statusCounts.value.active);
const partiallyNum = computed(() => statusCounts.value.partially);
const unhealthyNum = computed(() => statusCounts.value.unhealthy);
const downNum = computed(() => statusCounts.value.down);
const exitedNum = computed(() => statusCounts.value.exited);
const updateAvailableNum = computed(() => statusCounts.value.updateAvailable);

async function convertDockerRun() {
    const cmd = dockerRunCommand.value.trim();
    if (!cmd || cmd === "docker run") {
        toastRes({ ok: false, msg: "Please enter a docker run command" });
        return;
    }

    try {
        const { default: composerize } = await import("composerize");
        composeTemplate.value = composerize(cmd);
        router.push("/stacks/new");
    } catch (e: unknown) {
        toastRes({ ok: false, msg: (e instanceof Error ? e.message : String(e)) || "Failed to convert docker run command" });
    }
}

</script>

<style lang="scss" scoped>
@import "../styles/vars";

.num {
    font-size: 30px;

    font-weight: bold;
    display: block;

    &.active {
        color: $primary;
    }

    &.partially {
        color: $info;
    }

    &.unhealthy {
        color: $danger;
    }

    &.exited {
        color: $warning;
    }

    &.update-available {
        color: $info;
    }
}

.shadow-box {
    padding: 20px;
}

table {
    font-size: 14px;

    tr {
        transition: all ease-in-out 0.2ms;
    }

    @media (max-width: 550px) {
        table-layout: fixed;
        overflow-wrap: break-word;
    }
}

.docker-run {
    border: none;
    font-family: 'JetBrains Mono', monospace;
    font-size: 15px;
}

.first-row .shadow-box {

}


</style>
