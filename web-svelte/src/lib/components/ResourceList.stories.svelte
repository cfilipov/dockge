<script module lang="ts">
	import { defineMeta } from "@storybook/addon-svelte-csf";
	import ResourceList from "./ResourceList.svelte";

	const { Story } = defineMeta({
		title: "Components/ResourceList",
		component: ResourceList,
		});
</script>

<Story name="Default">
	<div class="flex h-screen w-80">
		<ResourceList />
	</div>
</Story>
