<script module lang="ts">
	import { defineMeta } from "@storybook/addon-svelte-csf";
	import OverviewItem from "./OverviewItem.svelte";

	const { Story } = defineMeta({
		title: "UI/OverviewItem",
		argTypes: {
			item: { control: "object" },
		},
		args: {
			item: { type: "text", label: "IMAGE", text: "nginx:latest" },
		},
	});
</script>

<Story name="Playground">
	{#snippet template(args)}
		<div class="p-4 max-w-sm">
			<OverviewItem item={args.item} />
		</div>
	{/snippet}
</Story>

<Story name="Text">
	<div class="p-4 max-w-sm">
		<OverviewItem item={{ type: "text", label: "NETWORK", text: "bridge" }} />
	</div>
</Story>

<Story name="Text with Link">
	<div class="p-4 max-w-sm">
		<OverviewItem item={{ type: "text", label: "IMAGE", text: "nginx:latest", href: "#" }} />
	</div>
</Story>

<Story name="Code">
	<div class="p-4 max-w-sm">
		<OverviewItem item={{ type: "code", label: "SUBNET", text: "172.18.0.0/16" }} />
	</div>
</Story>

<Story name="Code Truncated">
	<div class="p-4 max-w-sm">
		<OverviewItem item={{ type: "code", label: "ID", text: "sha256:9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08", truncate: true }} />
	</div>
</Story>

<Story name="Code with Link">
	<div class="p-4 max-w-sm">
		<OverviewItem item={{ type: "code", label: "CONTAINER ID", text: "a1b2c3d4e5f6", href: "#" }} />
	</div>
</Story>

<Story name="Status">
	<div class="p-4 max-w-sm">
		<OverviewItem item={{ type: "status", label: "STATUS", status: "running", name: "nginx-proxy" }} />
	</div>
</Story>

<Story name="Status Exited">
	<div class="p-4 max-w-sm">
		<OverviewItem item={{ type: "status", label: "STATUS", status: "exited", name: "old-container" }} />
	</div>
</Story>

<Story name="Mapping">
	<div class="p-4 max-w-sm">
		<OverviewItem item={{ type: "mapping", label: "PORTS", pairs: [{ from: "8080", to: "80/tcp" }, { from: "8443", to: "443/tcp" }] }} />
	</div>
</Story>

