<script module lang="ts">
	import { defineMeta } from "@storybook/addon-svelte-csf";
	import ResourceList from "./ResourceList.svelte";

	const { Story } = defineMeta({
		title: "Components/ResourceList",
		parameters: {
			layout: "fullscreen",
		},
	});
</script>

<Story name="Default">
	<div class="flex h-full min-h-0 flex-col p-3" style="width: 25vw; min-width: 320px;">
		<ResourceList />
	</div>
</Story>
