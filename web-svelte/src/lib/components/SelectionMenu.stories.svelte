<script module lang="ts">
	import { defineMeta } from "@storybook/addon-svelte-csf";
	import SelectionMenu from "./SelectionMenu.svelte";
	import About from "$lib/features/settings/About.svelte";

	const { Story } = defineMeta({
		title: "Components/SelectionMenu",
		argTypes: {
			activeKey: {
				control: "select",
				options: ["general", "appearance", "security", "global-env", "about"],
			},
		},
		args: {
			activeKey: "general",
			items: [
				{ key: "general", label: "General" },
				{ key: "appearance", label: "Appearance" },
				{ key: "security", label: "Security" },
				{ key: "global-env", label: "Global Env" },
				{ key: "about", label: "About" },
			],
		},
	});
</script>

<Story name="Default">
	{#snippet template(args)}
		<SelectionMenu {...args}>
			<p class="text-(--color-font-dark-muted)">Select a menu item to view its content.</p>
		</SelectionMenu>
	{/snippet}
</Story>

<Story name="With About Content">
	{#snippet template(args)}
		<SelectionMenu {...args} activeKey="about">
			<About />
		</SelectionMenu>
	{/snippet}
</Story>
