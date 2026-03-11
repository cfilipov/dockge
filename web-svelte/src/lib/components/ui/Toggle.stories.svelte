<script module lang="ts">
	import { defineMeta } from "@storybook/addon-svelte-csf";
	import Toggle from "./Toggle.svelte";

	const { Story } = defineMeta({
		title: "UI/Toggle",
		argTypes: {
			checked: { control: "boolean" },
			disabled: { control: "boolean" },
		},
		args: {
			checked: false,
			disabled: false,
			label: "",
		},
	});
</script>

<Story name="Playground">
	{#snippet template(args)}
		<Toggle {...args} />
	{/snippet}
</Story>

<Story name="Default">
	<Toggle />
</Story>

<Story name="Checked">
	<Toggle checked={true} />
</Story>

<Story name="Disabled">
	<div class="flex gap-4">
		<Toggle disabled />
		<Toggle checked={true} disabled />
	</div>
</Story>

<Story name="With Label">
	<Toggle label="Enable notifications" />
</Story>

<Story name="With Label Disabled">
	<div class="flex flex-col gap-3">
		<Toggle label="Disabled off" disabled />
		<Toggle label="Disabled on" checked={true} disabled />
	</div>
</Story>
