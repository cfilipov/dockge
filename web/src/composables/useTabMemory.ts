import { ref, watch } from "vue";
import { useRoute } from "vue-router";

// Module-level refs — singleton state shared across all callers
const lastStack = ref("");
const lastContainer = ref("");
const lastImage = ref("");
const lastNetwork = ref("");
const lastVolume = ref("");

let watcherInstalled = false;

/**
 * Remembers the last selected item for each tab category so that
 * tab links can restore the previous selection when no smarter
 * cross-tab guess is available.
 *
 * Call once from Layout.vue — the route watcher is installed only once.
 */
export function useTabMemory() {
    if (!watcherInstalled) {
        watcherInstalled = true;
        const route = useRoute();

        watch(() => route.params, (params) => {
            const stack = params.stackName as string;
            const container = params.containerName as string;
            const image = params.imageRef as string;
            const network = params.networkName as string;
            const volume = params.volumeName as string;

            if (stack) lastStack.value = stack;
            if (container) lastContainer.value = container;
            if (image) lastImage.value = image;
            if (network) lastNetwork.value = network;
            if (volume) lastVolume.value = volume;
        }, { immediate: true });
    }

    return {
        lastStack,
        lastContainer,
        lastImage,
        lastNetwork,
        lastVolume,
    };
}
