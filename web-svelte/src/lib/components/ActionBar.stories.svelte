<script module lang="ts">
	import { defineMeta } from "@storybook/addon-svelte-csf";
	import ActionBar from "./ActionBar.svelte";

	const { Story } = defineMeta({
		title: "UI/ActionBar",
		argTypes: {
			size: { control: "select", options: ["sm", "lg"] },
			processing: { control: "boolean" },
			actions: { control: "object" },
			overflow: { control: "object" },
		},
		args: {
			size: "lg",
			processing: false,
			actions: ["edit", "restart", "update", "stop"],
			overflow: ["checkUpdates", "down", "delete"],
		},
	});
</script>

<Story name="Playground">
	{#snippet template(args)}
		<ActionBar {...args} />
	{/snippet}
</Story>

<Story name="Stack View Mode (Large)">
	<ActionBar
		actions={["edit", "restart", "update", "stop"]}
		overflow={["checkUpdates", "down", "delete"]}
		onaction={(a) => console.log("action:", a)}
	/>
</Story>

<Story name="Stack View Mode — Update Available">
	<ActionBar
		actions={["edit", "restart", "updateAvailable", "stop"]}
		overflow={["checkUpdates", "down", "delete"]}
		onaction={(a) => console.log("action:", a)}
	/>
</Story>

<Story name="Stack Edit Mode (Large)">
	<ActionBar
		actions={["deploy", "saveDraft"]}
		overflow={["checkUpdates", "down"]}
		onaction={(a) => console.log("action:", a)}
	/>
</Story>

<Story name="Service Card (Small)">
	<ActionBar
		size="sm"
		actions={["logs", "terminal", "restart", "recreateNecessary", "updateAvailable", "stop"]}
		onaction={(a) => console.log("action:", a)}
	/>
</Story>

<Story name="Processing State">
	<ActionBar
		actions={["edit", "restart", "update", "stop"]}
		overflow={["checkUpdates", "down", "delete"]}
		processing
		onaction={(a) => console.log("action:", a)}
	/>
</Story>
