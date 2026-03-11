<script module lang="ts">
	import { defineMeta } from "@storybook/addon-svelte-csf";
	import Card from "./Card.svelte";

	const { Story } = defineMeta({
		title: "UI/Card",
	});
</script>

<Story name="Default">
	<Card>
		<div class="p-3">
			<p class="mb-0">Simple card with default padding.</p>
		</div>
	</Card>
</Story>

<Story name="With Header">
	<Card>
		{#snippet header()}
			<span class="font-semibold">Card Header</span>
		{/snippet}
		<div class="p-3">
			<p class="mb-0">Card body content beneath the header.</p>
		</div>
	</Card>
</Story>

<Story name="With Content">
	<Card class="max-w-md">
		<div class="flex flex-col gap-2 p-3">
			<h4 class="text-lg font-semibold">Stack Details</h4>
			<p class="text-sm text-gray-500">A card with richer content inside.</p>
			<div class="flex gap-2">
				<span class="rounded bg-green-100 px-2 py-0.5 text-xs text-green-700">running</span>
				<span class="text-xs text-gray-400">3 services</span>
			</div>
		</div>
	</Card>
</Story>
