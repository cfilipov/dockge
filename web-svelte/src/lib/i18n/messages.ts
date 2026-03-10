const en = {
	stacks: "Stacks",
	containersNav: "Containers",
	networksNav: "Networks",
	imagesNav: "Images",
	volumesNav: "Volumes",
	console: "Console",
	badgeActive: "active",
	badgeRunning: "running",
	badgeUnhealthy: "unhealthy",
	badgeExited: "exited",
	badgePartially: "active⁻",
	badgePaused: "paused",
	badgeCreated: "created",
	badgeDead: "dead",
	badgeDown: "down",
	badgeInUse: "in use",
	badgeUnused: "unused",
	badgeDangling: "dangling",
} as const;

export type MessageKey = keyof typeof en;
export const m = en;
