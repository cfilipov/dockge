<script module lang="ts">
	import { defineMeta } from "@storybook/addon-svelte-csf";
	import About from "./About.svelte";

	const { Story } = defineMeta({
		title: "Components/About",
		argTypes: {
			version: { control: "text" },
			frontendVersion: { control: "text" },
		},
		args: {
			version: "1.5.0",
			frontendVersion: "1.5.0",
		},
	});
</script>

<Story name="Default">
	{#snippet template(args)}
		<About {...args} />
	{/snippet}
</Story>
