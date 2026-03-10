<script module lang="ts">
	import { defineMeta } from "@storybook/addon-svelte-csf";
	import Badge from "./Badge.svelte";

	const { Story } = defineMeta({
		title: "UI/Badge",
		argTypes: {
			status: {
				control: "select",
				options: ["active", "running", "unhealthy", "exited", "partially", "paused", "created", "dead", "down", "inUse", "unused", "dangling"],
			},
		},
		args: {
			status: "running",
		},
	});
</script>

<Story name="Playground">
	{#snippet template(args)}
		<Badge {...args} />
	{/snippet}
</Story>

<Story name="Stack Status">
	<div class="flex flex-wrap items-center gap-2">
		<Badge status="active" />
		<Badge status="partially" />
		<Badge status="exited" />
		<Badge status="unhealthy" />
		<Badge status="down" />
	</div>
</Story>

<Story name="Container Status">
	<div class="flex flex-wrap items-center gap-2">
		<Badge status="running" />
		<Badge status="unhealthy" />
		<Badge status="exited" />
		<Badge status="paused" />
		<Badge status="created" />
		<Badge status="dead" />
	</div>
</Story>

<Story name="Resource Usage">
	<div class="flex flex-wrap items-center gap-2">
		<Badge status="inUse" />
		<Badge status="unused" />
		<Badge status="dangling" />
	</div>
</Story>

<Story name="Scales With Parent Font Size">
	<div class="flex flex-col gap-4">
		<div class="flex items-center gap-2 text-base">
			<span class="text-gray-500">Body text (1rem):</span>
			<Badge status="running" />
			<Badge status="exited" />
			<Badge status="down" />
		</div>
		<div class="flex items-center gap-2 text-[1.25rem]">
			<span class="text-gray-500">Heading h5 (1.25rem):</span>
			<Badge status="running" />
			<Badge status="unhealthy" />
			<Badge status="paused" />
		</div>
		<div class="flex items-center gap-2 text-[2rem]">
			<span class="text-gray-500">Heading h1 (2rem):</span>
			<Badge status="inUse" />
			<Badge status="dangling" />
		</div>
	</div>
</Story>
