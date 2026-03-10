<script lang="ts">
	import OverviewItemComponent from "./ui/OverviewItem.svelte";
	import type { OverviewItem } from "./ui/OverviewItem.svelte";

	interface Props {
		items: OverviewItem[];
		loading?: boolean;
		ariaLabel?: string;
	}

	let { items, loading = false, ariaLabel }: Props = $props();
</script>

<div
	class="shadow-box dark:bg-(--color-body-dark) dark:shadow-none p-3 mb-3 rounded-[10px]"
	role="region"
	aria-label={ariaLabel}
>
	{#if loading}
		<span class="text-(--color-font-dark-muted)">Loading...</span>
	{:else}
		<div class="flex flex-col gap-2">
			{#each items as item}
				<OverviewItemComponent {item} />
			{/each}
		</div>
	{/if}
</div>
