<script module lang="ts">
	import { defineMeta } from "@storybook/addon-svelte-csf";
	import Checkbox from "./Checkbox.svelte";

	const { Story } = defineMeta({
		title: "UI/Checkbox",
		argTypes: {
			checked: { control: "boolean" },
			disabled: { control: "boolean" },
			label: { control: "text" },
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
		<Checkbox {...args} />
	{/snippet}
</Story>

<Story name="Default">
	<Checkbox />
</Story>

<Story name="Checked">
	<Checkbox checked={true} />
</Story>

<Story name="With Label">
	<Checkbox label="Check for updates automatically" />
</Story>

<Story name="Disabled">
	<div class="flex flex-col gap-3">
		<Checkbox label="Disabled unchecked" disabled />
		<Checkbox label="Disabled checked" checked={true} disabled />
	</div>
</Story>
