<script module lang="ts">
	import { defineMeta } from "@storybook/addon-svelte-csf";
	import ContentPane from "./ContentPane.svelte";

	const { Story } = defineMeta({
		title: "Components/ContentPane",
		component: ContentPane,
	});
</script>

<Story name="Default" />
