<script module lang="ts">
	import { defineMeta } from "@storybook/addon-svelte-csf";
	import ComposeButton from "./ComposeButton.svelte";

	const { Story } = defineMeta({
		title: "Components/ComposeButton",
		component: ComposeButton,
	});
</script>

<Story name="Default">
	<ComposeButton />
</Story>
