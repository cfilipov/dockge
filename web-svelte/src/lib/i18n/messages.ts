const en = {
	stacks: "Stacks",
	containersNav: "Containers",
	networksNav: "Networks",
	imagesNav: "Images",
	volumesNav: "Volumes",
	console: "Console",
} as const;

export type MessageKey = keyof typeof en;
export const m = en;
