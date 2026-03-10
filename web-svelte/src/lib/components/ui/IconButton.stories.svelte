<script module lang="ts">
	import { defineMeta } from "@storybook/addon-svelte-csf";
	import IconButton from "./IconButton.svelte";

	const { Story } = defineMeta({
		title: "UI/IconButton",
		argTypes: {
			size: { control: "select", options: ["sm", "md", "lg"] },
			disabled: { control: "boolean" },
			"aria-label": { control: "text" },
		},
		args: {
			size: "md",
			disabled: false,
			"aria-label": "Settings",
		},
	});
</script>

<script lang="ts">
	import { faXmark, faFilter, faEllipsisVertical, faGear } from "@fortawesome/free-solid-svg-icons";
</script>

<Story name="Playground">
	{#snippet template(args)}
		<IconButton {...args} icon={faGear} />
	{/snippet}
</Story>

<Story name="Default">
	<div class="flex flex-wrap items-center gap-2">
		<IconButton icon={faGear} aria-label="Settings" />
	</div>
</Story>

<Story name="All Sizes">
	<div class="flex flex-wrap items-center gap-2">
		<IconButton icon={faGear} aria-label="Settings" size="sm" />
		<IconButton icon={faGear} aria-label="Settings" size="md" />
		<IconButton icon={faGear} aria-label="Settings" size="lg" />
	</div>
</Story>

<Story name="Common Actions">
	<div class="flex flex-wrap items-center gap-2">
		<IconButton icon={faXmark} aria-label="Close" />
		<IconButton icon={faFilter} aria-label="Filter" />
		<IconButton icon={faEllipsisVertical} aria-label="More actions" />
		<IconButton icon={faGear} aria-label="Settings" />
	</div>
</Story>

<Story name="Disabled">
	<div class="flex flex-wrap items-center gap-2">
		<IconButton icon={faGear} aria-label="Settings" disabled />
	</div>
</Story>
