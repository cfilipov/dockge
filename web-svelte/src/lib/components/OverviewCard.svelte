<script lang="ts">
	import { slide } from "svelte/transition";
	import Card from "./ui/Card.svelte";
	import OverviewItemComponent from "./ui/OverviewItem.svelte";
	import type { OverviewItem } from "./ui/OverviewItem.svelte";

	interface Props {
		items: OverviewItem[];
		loading?: boolean;
		ariaLabel?: string;
	}

	let { items, loading = false, ariaLabel }: Props = $props();
</script>

<Card class="mb-3" role="region" aria-label={ariaLabel}>
	<div class="p-3">
		{#if loading}
			<span class="text-(--color-font-dark-muted)">Loading...</span>
		{:else}
			<div class="flex flex-col gap-2" transition:slide={{ duration: 250 }}>
				{#each items as item}
					<OverviewItemComponent {item} />
				{/each}
			</div>
		{/if}
	</div>
</Card>
