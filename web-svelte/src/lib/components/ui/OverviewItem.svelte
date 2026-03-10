<script lang="ts">
	import StatusLabel from "./StatusLabel.svelte";
	import type { BadgeStatus } from "./Badge.svelte";

	export type OverviewItemText = { type: "text"; label: string; text: string; href?: string };
	export type OverviewItemCode = { type: "code"; label: string; text: string; truncate?: boolean; href?: string };
	export type OverviewItemStatus = { type: "status"; label: string; status: BadgeStatus; name: string; href?: string };
	export type OverviewItemMapping = { type: "mapping"; label: string; pairs: Array<{ from: string; to: string }> };

	export type OverviewItem = OverviewItemText | OverviewItemCode | OverviewItemStatus | OverviewItemMapping;

	interface Props {
		item: OverviewItem;
	}

	let { item }: Props = $props();
</script>

<div class="flex flex-col rounded-[10px] bg-(--color-body-light) dark:bg-(--color-header-dark) px-4 py-3">
	<span class="text-[0.8em] font-semibold uppercase tracking-wide text-(--color-font-dark-muted) mb-0.5">
		{item.label}
	</span>

	{#if item.type === "text"}
		{#if item.href}
			<a href={item.href} class="font-semibold no-underline text-(--color-primary) hover:brightness-110 break-all">
				{item.text}
			</a>
		{:else}
			<span class="break-all text-(--color-primary)">{item.text}</span>
		{/if}
	{:else if item.type === "code"}
		{#if item.href}
			<a href={item.href} class="font-semibold no-underline text-(--color-primary) hover:brightness-110 break-all">
				<code class="font-mono text-[0.85em] bg-transparent p-0" class:block={item.truncate} class:overflow-hidden={item.truncate} class:text-ellipsis={item.truncate} class:whitespace-nowrap={item.truncate} title={item.truncate ? item.text : undefined}>
					{item.text}
				</code>
			</a>
		{:else}
			<code class="font-mono text-[0.85em] bg-transparent p-0 break-all text-(--color-primary)" class:block={item.truncate} class:overflow-hidden={item.truncate} class:text-ellipsis={item.truncate} class:whitespace-nowrap={item.truncate} title={item.truncate ? item.text : undefined}>
				{item.text}
			</code>
		{/if}
	{:else if item.type === "status"}
		<StatusLabel status={item.status} name={item.name} size="sm" href={item.href} />
	{:else if item.type === "mapping"}
		<div class="flex flex-col gap-0.5">
			{#each item.pairs as pair}
				<span class="break-all text-(--color-primary)">
					<code class="font-mono text-[0.85em] bg-transparent p-0">{pair.from}</code>
					<span class="mx-1">→</span>
					<code class="font-mono text-[0.85em] bg-transparent p-0">{pair.to}</code>
				</span>
			{/each}
		</div>
	{/if}
</div>
