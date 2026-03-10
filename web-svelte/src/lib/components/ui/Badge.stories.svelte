<script module lang="ts">
	import { defineMeta } from "@storybook/addon-svelte-csf";
	import Badge from "./Badge.svelte";

	const { Story } = defineMeta({
		title: "UI/Badge",
	});
</script>

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

<Story name="All Sizes">
	<div class="flex flex-wrap items-center gap-2">
		<Badge status="running" size="sm" />
		<Badge status="running" size="md" />
		<Badge status="running" size="lg" />
	</div>
</Story>

<Story name="Size and Status Matrix">
	<div class="flex flex-col gap-3">
		<div>
			<span class="mr-2 text-sm font-medium text-gray-500">sm</span>
			<span class="inline-flex flex-wrap items-center gap-2">
				<Badge status="active" size="sm" />
				<Badge status="running" size="sm" />
				<Badge status="unhealthy" size="sm" />
				<Badge status="exited" size="sm" />
				<Badge status="paused" size="sm" />
				<Badge status="inUse" size="sm" />
				<Badge status="down" size="sm" />
			</span>
		</div>
		<div>
			<span class="mr-2 text-sm font-medium text-gray-500">md</span>
			<span class="inline-flex flex-wrap items-center gap-2">
				<Badge status="active" size="md" />
				<Badge status="running" size="md" />
				<Badge status="unhealthy" size="md" />
				<Badge status="exited" size="md" />
				<Badge status="paused" size="md" />
				<Badge status="inUse" size="md" />
				<Badge status="down" size="md" />
			</span>
		</div>
		<div>
			<span class="mr-2 text-sm font-medium text-gray-500">lg</span>
			<span class="inline-flex flex-wrap items-center gap-2">
				<Badge status="active" size="lg" />
				<Badge status="running" size="lg" />
				<Badge status="unhealthy" size="lg" />
				<Badge status="exited" size="lg" />
				<Badge status="paused" size="lg" />
				<Badge status="inUse" size="lg" />
				<Badge status="down" size="lg" />
			</span>
		</div>
	</div>
</Story>

<Story name="In Context">
	<div class="flex flex-col gap-4">
		<div class="flex items-center gap-2 text-base">
			<span class="text-gray-500">List item (sm, default):</span>
			<Badge status="running" />
			<Badge status="exited" />
			<Badge status="down" />
		</div>
		<div class="flex items-center gap-2 text-base">
			<span class="text-gray-500">Card heading (md):</span>
			<Badge status="running" size="md" />
			<Badge status="unhealthy" size="md" />
			<Badge status="paused" size="md" />
		</div>
		<div class="flex items-center gap-2 text-2xl">
			<span class="text-gray-500">Page title (lg):</span>
			<Badge status="inUse" size="lg" />
			<Badge status="dangling" size="lg" />
		</div>
	</div>
</Story>
