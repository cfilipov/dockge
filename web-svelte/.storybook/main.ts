import type { StorybookConfig } from "@storybook/sveltekit";

const config: StorybookConfig = {
	stories: ["../src/**/*.stories.@(svelte|ts)"],
	addons: [
		"@storybook/addon-themes",
		"@storybook/addon-svelte-csf",
	],
	framework: "@storybook/sveltekit",
	staticDirs: ["../static"],
	core: {
		allowedHosts: true,
	},
};

export default config;
