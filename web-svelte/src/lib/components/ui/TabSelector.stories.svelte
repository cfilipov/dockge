<script module lang="ts">
	import { defineMeta } from "@storybook/addon-svelte-csf";
	import {
		faList,
		faCode,
		faFileLines,
		faTerminal,
	} from "@fortawesome/free-solid-svg-icons";
	import TabSelector from "./TabSelector.svelte";
	import type { Tab } from "./TabSelector.svelte";

	const viewModeTabs: Tab[] = [
		{ value: "parsed", icon: faList, label: "Parsed" },
		{ value: "raw", icon: faCode, label: "Raw" },
		{ value: "logs", icon: faFileLines, label: "Logs" },
		{ value: "shell", icon: faTerminal, label: "Shell" },
	];

	const twoTabs: Tab[] = [
		{ value: "parsed", icon: faList, label: "Parsed" },
		{ value: "raw", icon: faCode, label: "Raw" },
	];

	const { Story } = defineMeta({
		title: "UI/TabSelector",
		argTypes: {
			value: {
				control: "select",
				options: ["parsed", "raw", "logs", "shell"],
			},
		},
		args: {
			value: "parsed",
			tabs: viewModeTabs,
		},
	});
</script>

<Story name="Playground">
	{#snippet template(args)}
		<TabSelector {...args} />
	{/snippet}
</Story>

<Story name="Two Tabs">
	<TabSelector tabs={twoTabs} value="parsed" />
</Story>
