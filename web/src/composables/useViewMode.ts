import { ref } from "vue";

/**
 * Shared view mode preference — singleton state across all components.
 *
 * When the user toggles "Show UI" / "Show YAML" on any page (stacks or
 * containers), the preference persists until they toggle again.
 *
 * Compose.vue calls setRawMode() when the toggle changes.
 * StackListItem and Layout read isRawMode to build correct URLs.
 */

const isRawMode = ref(false);

function setRawMode(raw: boolean) {
    isRawMode.value = raw;
}

export function useViewMode() {
    return { isRawMode, setRawMode };
}
