<script module lang="ts">
	import { defineMeta } from "@storybook/addon-svelte-csf";
	import MobileNav from "./MobileNav.svelte";

	const { Story } = defineMeta({
		title: "Components/MobileNav",
		component: MobileNav,
		parameters: {
			layout: "fullscreen",
			viewport: { defaultViewport: "mobile1" },
		},
	});
</script>

<Story name="Default">
	<div class="relative h-screen">
		<MobileNav />
	</div>
</Story>
