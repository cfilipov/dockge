<script lang="ts">
	import type { Snippet } from "svelte";

	interface Props {
		href?: string;
		active?: boolean;
		onclick?: (e: MouseEvent) => void;
		children: Snippet;
	}

	let {
		href,
		active = false,
		onclick,
		children,
	}: Props = $props();

	const baseClasses = "flex items-center min-h-[46px] rounded-[10px] w-full px-2 my-[3px] overflow-hidden min-w-0 no-underline text-inherit transition-none";
	const activeClasses = "bg-(--color-highlight-active) border-l-4 border-l-(--color-primary) rounded-tl-none rounded-bl-none dark:bg-(--color-header-dark)";
	const inactiveClasses = "hover:bg-(--color-body-light) dark:hover:bg-(--color-header-dark)";
</script>

{#if href}
	<a
		{href}
		class="{baseClasses} {active ? activeClasses : inactiveClasses}"
		{onclick}
	>
		{@render children()}
	</a>
{:else}
	<div
		role="button"
		tabindex="0"
		class="{baseClasses} {active ? activeClasses : inactiveClasses}"
		{onclick}
		onkeydown={(e: KeyboardEvent) => { if (e.key === "Enter" || e.key === " ") onclick?.(e as unknown as MouseEvent); }}
	>
		{@render children()}
	</div>
{/if}
