<script module lang="ts">
	import { defineMeta } from "@storybook/addon-svelte-csf";
	import Section from "./Section.svelte";
	import DataTable from "./ui/DataTable.svelte";
	import OverviewCard from "./OverviewCard.svelte";

	const { Story } = defineMeta({
		title: "Components/Section",
	});
</script>

<Story
	name="Playground"
	argTypes={{
		title: { control: "text" },
		count: { control: "number" },
		collapsible: { control: "boolean" },
		expanded: { control: "boolean" },
	}}
	args={{ title: "Section Title", collapsible: false, expanded: true }}
>
	{#snippet template(args)}
		<div class="max-w-2xl">
			<Section title={args.title} count={args.count} collapsible={args.collapsible} expanded={args.expanded}>
				<div class="shadow-box dark:bg-(--color-body-dark) dark:shadow-none p-4 rounded-[10px]">
					<p class="text-(--color-font-dark-muted)">Section content goes here.</p>
				</div>
			</Section>
		</div>
	{/snippet}
</Story>

<Story name="Plain Section">
	<div class="max-w-2xl">
		<Section title="Container Overview">
			<OverviewCard
				items={[
					{ type: "text", label: "Image", text: "nginx:latest" },
					{ type: "text", label: "Status", text: "Running" },
					{ type: "text", label: "Ports", text: "80:80, 443:443" },
				]}
			/>
		</Section>
	</div>
</Story>

<Story name="Collapsible Section">
	<div class="max-w-2xl">
		<Section title="Processes" count={4} collapsible expanded>
			<div class="shadow-box dark:bg-(--color-body-dark) dark:shadow-none p-4 rounded-[10px]">
				<DataTable
					columns={[
						{ key: "pid", label: "PID", mono: true },
						{ key: "user", label: "User" },
						{ key: "command", label: "Command", mono: true },
					]}
					rows={[
						{ pid: "1", user: "root", command: "/sbin/init" },
						{ pid: "42", user: "root", command: "nginx: master process" },
						{ pid: "78", user: "www-data", command: "nginx: worker process" },
						{ pid: "112", user: "www-data", command: "nginx: worker process" },
					]}
				/>
			</div>
		</Section>
	</div>
</Story>

<Story name="Sidebar Label">
	<div class="max-w-xs">
		<Section title="Containers">
			<div class="flex flex-col gap-1">
				{#each ["web-app", "monitoring", "blog", "test-alpine"] as name}
					<div class="shadow-box dark:bg-(--color-body-dark) dark:shadow-none px-3 py-2 rounded-lg text-sm">
						{name}
					</div>
				{/each}
			</div>
		</Section>
	</div>
</Story>
