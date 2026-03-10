<script module lang="ts">
	import { defineMeta } from "@storybook/addon-svelte-csf";
	import SectionHeading from "./SectionHeading.svelte";

	const { Story } = defineMeta({
		title: "UI/SectionHeading",
	});
</script>

<Story
	name="Playground"
	argTypes={{
		title: { control: "text" },
		count: { control: "number" },
		collapsible: { control: "boolean" },
		expanded: { control: "boolean" },
		size: { control: "select", options: ["sm", "lg"] },
	}}
	args={{ title: "Section Heading", collapsible: true, expanded: true, size: "sm" }}
>
	{#snippet template(args)}
		<SectionHeading {...args} />
	{/snippet}
</Story>

<Story name="All Sizes">
	<div class="flex flex-col gap-4">
		<SectionHeading title="Subsection (sm)" size="sm" />
		<SectionHeading title="Containers (lg)" size="lg" />
	</div>
</Story>

<Story name="With Count">
	<div class="flex flex-col gap-4">
		<SectionHeading title="Processes" count={4} size="sm" />
		<SectionHeading title="Networks" count={2} size="lg" />
	</div>
</Story>

<Story name="Collapsible States">
	<div class="flex flex-col gap-4">
		<SectionHeading title="Expanded" collapsible expanded={true} size="sm" />
		<SectionHeading title="Collapsed" collapsible expanded={false} size="sm" />
		<SectionHeading title="Expanded with Count" collapsible expanded={true} count={4} size="sm" />
		<SectionHeading title="Collapsed with Count" collapsible expanded={false} count={3} size="sm" />
	</div>
</Story>
