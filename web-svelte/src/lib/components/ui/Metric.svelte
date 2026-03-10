<script lang="ts">
	export type MetricColor = "primary" | "info" | "danger" | "warning";

	interface Props {
		label: string;
		value1: string;
		unit1?: string;
		tag1?: string;
		value2?: string;
		unit2?: string;
		tag2?: string;
		color?: MetricColor;
		loading?: boolean;
	}

	let { label, value1, unit1, tag1, value2, unit2, tag2, color, loading = false }: Props = $props();
	let dual = $derived(value2 !== undefined);

	const colorMap: Record<MetricColor, string> = {
		primary: "var(--color-primary)",
		info: "var(--color-info)",
		danger: "var(--color-danger)",
		warning: "var(--color-warning)",
	};

	let valueColor = $derived(color ? colorMap[color] : "var(--color-primary)");
</script>

{#snippet valuePair(value: string, unit?: string, tag?: string)}
	<div class="text-center">
		<span class="inline-flex flex-col items-end">
			<span class="text-[30px] leading-[1.2] font-bold" class:opacity-30={loading} style:color={valueColor}>
				{value}{#if unit}<span class="text-[0.75em] opacity-40">&nbsp;{unit}</span>{/if}
			</span>
			{#if tag}
				<span class="text-[11px] text-(--color-font-dark-muted) leading-none -mt-[4px]">{tag}</span>
			{/if}
		</span>
	</div>
{/snippet}

<div class="rounded-[10px] bg-(--color-body-light) dark:bg-(--color-header-dark) px-2 py-3 flex flex-col text-center">
	<div class="text-[0.95rem] font-semibold mb-1">{label}</div>
	{#if dual}
		{@render valuePair(value1, unit1, tag1)}
		{@render valuePair(value2!, unit2, tag2)}
	{:else}
		<div class="flex-1 flex items-center justify-center -mt-[9px]">
			{@render valuePair(value1, unit1, tag1)}
		</div>
	{/if}
</div>
