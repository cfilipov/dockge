<script lang="ts">
	import type { Snippet } from "svelte";
	import ListItem from "./ui/ListItem.svelte";

	interface Item {
		key: string;
		label: string;
	}

	interface Props {
		items: Item[];
		activeKey?: string;
		onSelect?: (key: string) => void;
		children: Snippet;
	}

	let {
		items,
		activeKey,
		onSelect,
		children,
	}: Props = $props();

	let activeLabel = $derived(items.find((i) => i.key === activeKey)?.label ?? "");
</script>

<div class="shadow-box dark:bg-(--color-body-dark) dark:shadow-none p-5">
	<div class="flex">
		<nav class="w-60 shrink-0 px-6 py-3">
			{#each items as item (item.key)}
				<ListItem
					active={item.key === activeKey}
					onclick={() => onSelect?.(item.key)}
				>
					{item.label}
				</ListItem>
			{/each}
		</nav>

		<div class="flex-1 min-w-0">
			{#if activeLabel}
				<div class="rounded-tr-[10px] -mt-5 -mr-5 px-4 py-3 mb-4 dark:bg-(--color-header-dark)">
					<h4 class="mb-0 text-2xl font-medium">{activeLabel}</h4>
				</div>
			{/if}
			<div class="px-3">
				{@render children()}
			</div>
		</div>
	</div>
</div>
