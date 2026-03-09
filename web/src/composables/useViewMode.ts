import { ref } from "vue";

/**
 * Per-tab view mode preference — stacks and containers each remember their
 * own parsed/raw toggle independently.
 *
 * Compose.vue uses the "stacks" scope; ContainerInspect.vue uses "containers".
 * Layout reads the appropriate scope to reset on tab re-click.
 */

export type ContainersSubView = "parsed" | "raw" | "logs" | "shell";

const stacksRawMode = ref(false);
const containersSubView = ref<ContainersSubView>("parsed");

export function useViewMode(scope: "stacks" | "containers" = "stacks") {
    const isRawMode = scope === "stacks" ? stacksRawMode : ref(containersSubView.value === "raw");

    function setRawMode(raw: boolean) {
        if (scope === "stacks") {
            stacksRawMode.value = raw;
        } else {
            containersSubView.value = raw ? "raw" : "parsed";
        }
    }

    function getContainersSubView(): ContainersSubView {
        return containersSubView.value;
    }

    function setContainersSubView(view: ContainersSubView) {
        containersSubView.value = view;
    }

    return { isRawMode, setRawMode, getContainersSubView, setContainersSubView };
}
