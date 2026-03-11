<script module lang="ts">
	import { defineMeta } from "@storybook/addon-svelte-csf";
	import Card from "./Card.svelte";
	import CardInner from "./CardInner.svelte";

	const { Story } = defineMeta({
		title: "UI/CardInner",
	});
</script>

<Story name="Default">
	<CardInner>
		<span class="text-sm">Inner card with subtle background.</span>
	</CardInner>
</Story>

<Story name="Nested Inside Card">
	<Card class="max-w-md p-3">
		<div class="flex flex-col gap-2">
			<CardInner>
				<span class="text-sm font-semibold">First item</span>
			</CardInner>
			<CardInner>
				<span class="text-sm font-semibold">Second item</span>
			</CardInner>
		</div>
	</Card>
</Story>
