<script module lang="ts">
	import { defineMeta } from "@storybook/addon-svelte-csf";
	import Chip from "./Chip.svelte";

	const { Story } = defineMeta({
		title: "UI/Chip",
		argTypes: {
			label: { control: "text" },
			values: { control: "object" },
		},
		args: {
			label: "IMAGE",
			values: [{ text: "nginx:latest", href: "#" }],
		},
	});
</script>

<Story name="Playground">
	{#snippet template(args)}
		<Chip {...args} />
	{/snippet}
</Story>

<Story name="Image Chip">
	<div class="p-4">
		<Chip label="IMAGE" values={[{ text: "nginx:latest", href: "#" }]} />
	</div>
</Story>

<Story name="Ports Chip">
	<div class="p-4">
		<Chip
			label="PORTS"
			values={[
				{ text: "8080:80", href: "http://localhost:8080" },
				{ text: "8443:443", href: "http://localhost:8443" },
			]}
		/>
	</div>
</Story>

<Story name="Container ID">
	<div class="p-4">
		<Chip label="ID" values={["abc123def456"]} />
	</div>
</Story>

<Story name="Chip Group">
	<div class="flex flex-wrap gap-2 p-4">
		<Chip label="IMAGE" values={[{ text: "nginx:latest", href: "#" }]} />
		<Chip
			label="PORTS"
			values={[
				{ text: "8080:80", href: "http://localhost:8080" },
				{ text: "8443:443", href: "http://localhost:8443" },
			]}
		/>
		<Chip label="IP" values={["172.18.0.3"]} />
		<Chip label="ID" values={["abc123def456"]} />
	</div>
</Story>
