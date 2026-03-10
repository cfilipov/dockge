<script module lang="ts">
	import { defineMeta } from "@storybook/addon-svelte-csf";
	import Header from "$lib/components/Header.svelte";
	import ComposeButton from "$lib/components/ComposeButton.svelte";
	import ResourceList from "$lib/components/ResourceList.svelte";
	import ContentPane from "$lib/components/ContentPane.svelte";
	import MobileNav from "$lib/components/MobileNav.svelte";

	const { Story } = defineMeta({
		title: "Compositions/Dashboard",
		parameters: {
			layout: "fullscreen",
		},
	});
</script>

<Story name="Desktop">
	<div class="flex h-screen flex-col">
		<Header />
		<div class="flex flex-1 overflow-hidden">
			<aside class="hidden flex-shrink-0 flex-col overflow-hidden px-3 py-0 md:flex md:w-1/3 xl:w-1/4">
				<div class="mb-3 flex items-center">
					<ComposeButton />
				</div>
				<ResourceList />
			</aside>
			<ContentPane />
		</div>
	</div>
</Story>

<Story name="Mobile" parameters={{ viewport: { defaultViewport: "mobile1" } }}>
	<div class="flex h-screen flex-col">
		<ContentPane />
		<MobileNav />
	</div>
</Story>
