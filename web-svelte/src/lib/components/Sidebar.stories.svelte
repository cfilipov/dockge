<script module lang="ts">
	import { defineMeta } from "@storybook/addon-svelte-csf";
	import Sidebar from "./Sidebar.svelte";

	const { Story } = defineMeta({
		title: "Components/Sidebar",
		component: Sidebar,
		parameters: {
			layout: "fullscreen",
		},
	});
</script>

<Story name="Default">
	<div class="flex h-screen">
		<Sidebar />
	</div>
</Story>
