import { ref } from "vue";

/**
 * Per-tab view mode preference — stacks and containers each remember their
 * own parsed/raw toggle independently.
 *
 * Compose.vue uses the "stacks" scope; ContainerInspect.vue uses "containers".
 * Layout reads the appropriate scope to reset on tab re-click.
 */

const stacksRawMode = ref(false);
const containersRawMode = ref(false);

export function useViewMode(scope: "stacks" | "containers" = "stacks") {
    const isRawMode = scope === "stacks" ? stacksRawMode : containersRawMode;

    function setRawMode(raw: boolean) {
        isRawMode.value = raw;
    }

    return { isRawMode, setRawMode };
}
