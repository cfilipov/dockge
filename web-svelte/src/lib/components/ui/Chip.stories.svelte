<script module lang="ts">
	import { defineMeta } from "@storybook/addon-svelte-csf";
	import Chip from "./Chip.svelte";

	const { Story } = defineMeta({
		title: "UI/Chip",
	});
</script>

<Story name="Image Chip">
	<Chip label="IMAGE" values={[{ text: "nginx:latest", href: "#" }]} />
</Story>

<Story name="Ports Chip">
	<Chip
		label="PORTS"
		values={[
			{ text: "8080:80", href: "http://localhost:8080" },
			{ text: "8443:443", href: "http://localhost:8443" },
		]}
	/>
</Story>

<Story name="Container ID">
	<Chip label="ID" values={["abc123def456"]} />
</Story>

<Story name="Chip Group">
	<div class="flex flex-wrap gap-2">
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
