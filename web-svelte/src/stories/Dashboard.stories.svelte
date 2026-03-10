<script module lang="ts">
	import { defineMeta } from "@storybook/addon-svelte-csf";
	import Header from "$lib/components/Header.svelte";
	import Sidebar from "$lib/components/Sidebar.svelte";
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
			<Sidebar />
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
