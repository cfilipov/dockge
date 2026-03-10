import type { IconDefinition } from "@fortawesome/fontawesome-svg-core";
import {
	faPlay,
	faStop,
	faRotate,
	faRocket,
	faFloppyDisk,
	faPen,
	faCloudArrowDown,
	faTrash,
	faMagnifyingGlass,
	faFileLines,
	faTerminal,
} from "@fortawesome/free-solid-svg-icons";
import * as m from "$lib/paraglide/messages";

export type ActionType =
	| "start"
	| "stop"
	| "restart"
	| "deploy"
	| "saveDraft"
	| "edit"
	| "update"
	| "updateAvailable"
	| "recreate"
	| "recreateNecessary"
	| "down"
	| "delete"
	| "forceDelete"
	| "checkUpdates"
	| "logs"
	| "terminal";

export type ActionColor = "brand" | "secondary" | "gray" | "purple" | "red";

export interface ActionDef {
	icon: IconDefinition;
	label: () => string;
	color: ActionColor;
}

export const actionDefs: Record<ActionType, ActionDef> = {
	start: { icon: faPlay, label: () => m.start(), color: "brand" },
	stop: { icon: faStop, label: () => m.stop(), color: "gray" },
	restart: { icon: faRotate, label: () => m.restart(), color: "gray" },
	deploy: { icon: faRocket, label: () => m.deploy(), color: "brand" },
	saveDraft: { icon: faFloppyDisk, label: () => m.save(), color: "gray" },
	edit: { icon: faPen, label: () => m.edit(), color: "secondary" },
	update: { icon: faCloudArrowDown, label: () => m.update(), color: "gray" },
	updateAvailable: { icon: faCloudArrowDown, label: () => m.update(), color: "purple" },
	recreate: { icon: faRocket, label: () => m.recreate(), color: "gray" },
	recreateNecessary: { icon: faRocket, label: () => m.recreate(), color: "purple" },
	down: { icon: faStop, label: () => m.down(), color: "gray" },
	delete: { icon: faTrash, label: () => m["delete"](), color: "red" },
	forceDelete: { icon: faTrash, label: () => m.forceDelete(), color: "red" },
	checkUpdates: { icon: faMagnifyingGlass, label: () => m.checkUpdates(), color: "gray" },
	logs: { icon: faFileLines, label: () => m.logs(), color: "gray" },
	terminal: { icon: faTerminal, label: () => m.terminal(), color: "gray" },
};
