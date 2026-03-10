<script module lang="ts">
	import { defineMeta } from "@storybook/addon-svelte-csf";
	import TextArea from "./TextArea.svelte";

	const { Story } = defineMeta({
		title: "UI/TextArea",
		argTypes: {
			value: { control: "text" },
			placeholder: { control: "text" },
			rows: { control: "number" },
			disabled: { control: "boolean" },
		},
		args: {
			value: "",
			placeholder: "docker run ...",
			rows: 3,
			disabled: false,
		},
	});
</script>

<Story name="Playground">
	{#snippet template(args)}
		<TextArea {...args} />
	{/snippet}
</Story>

<Story name="Disabled">
	<TextArea placeholder="docker run ..." disabled />
</Story>
