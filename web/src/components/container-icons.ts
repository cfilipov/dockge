/**
 * Pre-resolved SVG icon data for Container.vue action buttons.
 *
 * Instead of 708 reactive <font-awesome-icon> Vue component instances
 * (6 per container card × 120 cards), Container.vue renders plain <svg>
 * elements using these pre-extracted paths. Zero Vue component overhead,
 * zero extra bundle size (reuses data already in @fortawesome/free-solid-svg-icons).
 */
import {
    faPlay,
    faStop,
    faRotate,
    faRocket,
    faCloudArrowDown,
    faFileLines,
    faTerminal,
    faEdit,
    faTrash,
    faTimes,
} from "@fortawesome/free-solid-svg-icons";
import type { IconDefinition } from "@fortawesome/fontawesome-svg-core";

interface IconData {
    viewBox: string;
    path: string;
}

function extract(icon: IconDefinition): IconData {
    return {
        viewBox: `0 0 ${icon.icon[0]} ${icon.icon[1]}`,
        path: icon.icon[4] as string,
    };
}

export const containerIcons = {
    "play": extract(faPlay),
    "stop": extract(faStop),
    "rotate": extract(faRotate),
    "rocket": extract(faRocket),
    "cloud-arrow-down": extract(faCloudArrowDown),
    "file-lines": extract(faFileLines),
    "terminal": extract(faTerminal),
    "edit": extract(faEdit),
    "trash": extract(faTrash),
    "times": extract(faTimes),
} as const;
