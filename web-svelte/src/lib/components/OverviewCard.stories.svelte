<script module lang="ts">
	import { defineMeta } from "@storybook/addon-svelte-csf";
	import OverviewCard from "./OverviewCard.svelte";
	import type { OverviewItem } from "./ui/OverviewItem.svelte";

	const containerItems: OverviewItem[] = [
		{ type: "status", label: "STATUS", status: "running", name: "nginx-proxy" },
		{ type: "code", label: "CONTAINER ID", text: "a1b2c3d4e5f6", truncate: true },
		{ type: "text", label: "IMAGE", text: "nginx:latest" },
		{ type: "mapping", label: "PORTS", pairs: [
			{ from: "8080", to: "80/tcp" },
			{ from: "8443", to: "443/tcp" },
		]},
{ type: "text", label: "NETWORK", text: "bridge" },
	];

	const networkItems: OverviewItem[] = [
		{ type: "text", label: "NAME", text: "my-network" },
		{ type: "text", label: "DRIVER", text: "bridge" },
		{ type: "text", label: "SCOPE", text: "local" },
		{ type: "code", label: "SUBNET", text: "172.18.0.0/16" },
		{ type: "code", label: "GATEWAY", text: "172.18.0.1" },
		{ type: "code", label: "ID", text: "sha256:9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08", truncate: true },
	];

	const { Story } = defineMeta({
		title: "Components/OverviewCard",
		argTypes: {
			loading: { control: "boolean" },
			ariaLabel: { control: "text" },
		},
		args: {
			loading: false,
			ariaLabel: "Container overview",
		},
	});
</script>

<Story name="Playground">
	{#snippet template(args)}
		<div class="p-4 max-w-md">
			<OverviewCard items={containerItems} loading={args.loading} ariaLabel={args.ariaLabel} />
		</div>
	{/snippet}
</Story>

<Story name="Network Overview">
	{#snippet template(args)}
		<div class="p-4 max-w-md">
			<OverviewCard items={networkItems} loading={args.loading} ariaLabel="Network: my-network" />
		</div>
	{/snippet}
</Story>

<Story name="Loading" args={{ loading: true }}>
	{#snippet template(args)}
		<div class="p-4 max-w-md">
			<OverviewCard items={containerItems} loading={args.loading} ariaLabel="Container overview" />
		</div>
	{/snippet}
</Story>
