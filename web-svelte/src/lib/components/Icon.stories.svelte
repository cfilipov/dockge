<script module lang="ts">
	import { defineMeta } from "@storybook/addon-svelte-csf";
	import Icon from "./Icon.svelte";
	import IconGallery from "../../stories/IconGallery.svelte";
	import { faLayerGroup } from "@fortawesome/free-solid-svg-icons";

	const { Story } = defineMeta({
		title: "Components/Icon",
	});
</script>

<Story name="Default">
	<Icon icon={faLayerGroup} class="text-2xl" />
</Story>

<Story name="All Icons">
	<IconGallery />
</Story>

<Story name="Custom Class">
	<Icon icon={faLayerGroup} class="text-4xl text-primary" />
</Story>
