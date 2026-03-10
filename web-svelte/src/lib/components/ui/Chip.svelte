<script lang="ts">
	type ChipValue = string | { text: string; href: string };

	interface Props {
		label: string;
		values: ChipValue[];
	}

	let { label, values }: Props = $props();
</script>

<span class="inline-flex items-center gap-[0.5rem] rounded-full px-3.5 h-[32px] bg-black/6 dark:bg-(--color-header-dark)">
	<span class="text-xs font-semibold uppercase whitespace-nowrap leading-none text-(--color-font-dark-muted)">{label}</span>
	<span class="inline-flex items-center text-sm">
		{#each values as value, i}
			{#if i > 0}<span class="text-(--color-font-dark-muted) mr-1">,</span>{/if}
			{#if typeof value === "string"}
				<code class="font-mono leading-none text-(--color-primary) bg-transparent p-0">{value}</code>
			{:else}
				<a href={value.href} class="inline-flex items-center no-underline text-inherit hover:[&_code]:underline">
					<code class="font-mono leading-none text-(--color-primary) bg-transparent p-0">{value.text}</code>
				</a>
			{/if}
		{/each}
	</span>
</span>
