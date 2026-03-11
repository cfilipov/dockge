<script module lang="ts">
	import { defineMeta } from "@storybook/addon-svelte-csf";
	import ResourceList from "./ResourceList.svelte";
	import ListItem from "./ui/ListItem.svelte";

	const { Story } = defineMeta({
		title: "Components/ResourceList",
		parameters: {
			layout: "fullscreen",
		},
		argTypes: {
			itemCount: { control: { type: "range", min: 0, max: 20, step: 1 } },
			showFilterIcon: { control: "boolean" },
			filterActive: { control: "boolean" },
		},
		args: {
			itemCount: 6,
			showFilterIcon: false,
			filterActive: false,
		},
	});

	const allItems = [
		"alpha", "bravo", "charlie", "delta", "echo", "foxtrot",
		"golf", "hotel", "india", "juliet", "kilo", "lima",
		"mike", "november", "oscar", "papa", "quebec", "romeo",
		"sierra", "tango",
	];
</script>

<Story name="Playground">
	{#snippet template(args)}
		{@const items = allItems.slice(0, args.itemCount)}
		<div class="flex h-full min-h-0 flex-col p-3" style="width: 25vw; min-width: 320px;">
			{#if args.showFilterIcon}
				<ResourceList filterActive={args.filterActive} count={items.length}>
					{#snippet filterMenu()}
						<span>Filter menu</span>
					{/snippet}
					{#each items as item}
						<ListItem href="/{item}">{item}</ListItem>
					{/each}
				</ResourceList>
			{:else}
				<ResourceList count={items.length}>
					{#each items as item}
						<ListItem href="/{item}">{item}</ListItem>
					{/each}
				</ResourceList>
			{/if}
		</div>
	{/snippet}
</Story>

<Story name="Empty">
	<div class="flex h-full min-h-0 flex-col p-3" style="width: 25vw; min-width: 320px;">
		<ResourceList count={0}>
			<p class="p-4 text-center text-sm text-gray-400">No items</p>
		</ResourceList>
	</div>
</Story>

<Story name="With Filter Icon">
	<div class="flex h-full min-h-0 flex-col p-3" style="width: 25vw; min-width: 320px;">
		<ResourceList>
			{#snippet filterMenu()}
				<span>Filter content</span>
			{/snippet}
			{#each allItems.slice(0, 6) as item}
				<ListItem href="/{item}">{item}</ListItem>
			{/each}
		</ResourceList>
	</div>
</Story>
