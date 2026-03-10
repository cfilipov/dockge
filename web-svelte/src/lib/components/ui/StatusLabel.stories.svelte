<script module lang="ts">
	import { defineMeta } from "@storybook/addon-svelte-csf";
	import StatusLabel from "./StatusLabel.svelte";

	const { Story } = defineMeta({
		title: "UI/StatusLabel",
		argTypes: {
			status: {
				control: "select",
				options: ["active", "running", "unhealthy", "exited", "partially", "paused", "created", "dead", "down", "inUse", "unused", "dangling"],
			},
			name: { control: "text" },
			size: { control: "select", options: ["sm", "md", "lg"] },
			href: { control: "text" },
			recreateNecessary: { control: "boolean" },
			updateAvailable: { control: "boolean" },
		},
		args: {
			status: "running",
			name: "my-stack",
			size: "sm",
			recreateNecessary: false,
			updateAvailable: false,
		},
	});
</script>

<Story name="Playground">
	{#snippet template(args)}
		<StatusLabel {...args} />
	{/snippet}
</Story>

<Story name="All Sizes">
	<div class="flex flex-col gap-4">
		<StatusLabel status="running" name="my-stack" size="sm" />
		<StatusLabel status="running" name="my-stack" size="md" />
		<StatusLabel status="running" name="my-stack" size="lg" />
	</div>
</Story>

<Story name="All Statuses">
	<div class="flex flex-col gap-2">
		<StatusLabel status="running" name="web-app" />
		<StatusLabel status="active" name="proxy-server" />
		<StatusLabel status="exited" name="old-service" />
		<StatusLabel status="unhealthy" name="failing-app" />
		<StatusLabel status="down" name="stopped-stack" />
		<StatusLabel status="partially" name="mixed-stack" />
		<StatusLabel status="paused" name="paused-stack" />
		<StatusLabel status="created" name="new-container" />
		<StatusLabel status="dead" name="dead-container" />
		<StatusLabel status="inUse" name="shared-network" />
		<StatusLabel status="unused" name="orphan-volume" />
		<StatusLabel status="dangling" name="old-image" />
	</div>
</Story>

<Story name="As Link">
	<div class="flex flex-col gap-4">
		<StatusLabel status="running" name="linked-stack" size="sm" href="#" />
		<StatusLabel status="exited" name="linked-stack" size="md" href="#" />
		<StatusLabel status="down" name="linked-stack" size="lg" href="#" />
	</div>
</Story>

<Story name="Long Name Truncation">
	<div class="w-60">
		<StatusLabel status="running" name="my-extremely-long-stack-name-that-should-be-truncated-in-the-ui" />
	</div>
</Story>

<Story name="Notification Icons">
	<div class="flex flex-col gap-4">
		<div class="flex flex-col gap-2">
			<p class="text-sm text-gray-500">Recreate necessary (sm)</p>
			<StatusLabel status="running" name="web-app" recreateNecessary />
		</div>
		<div class="flex flex-col gap-2">
			<p class="text-sm text-gray-500">Update available (sm)</p>
			<StatusLabel status="running" name="monitoring" updateAvailable />
		</div>
		<div class="flex flex-col gap-2">
			<p class="text-sm text-gray-500">Both icons (sm)</p>
			<StatusLabel status="running" name="full-stack" recreateNecessary updateAvailable />
		</div>
		<div class="flex flex-col gap-2">
			<p class="text-sm text-gray-500">Both icons (md)</p>
			<StatusLabel status="running" name="full-stack" size="md" recreateNecessary updateAvailable />
		</div>
		<div class="flex flex-col gap-2">
			<p class="text-sm text-gray-500">Both icons (lg)</p>
			<StatusLabel status="running" name="full-stack" size="lg" recreateNecessary updateAvailable />
		</div>
		<div class="flex flex-col gap-2">
			<p class="text-sm text-gray-500">Long name with icons (truncation test)</p>
			<div class="w-60">
				<StatusLabel status="running" name="my-very-long-stack-name-that-truncates" recreateNecessary updateAvailable />
			</div>
		</div>
	</div>
</Story>
