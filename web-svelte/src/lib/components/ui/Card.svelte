<script lang="ts">
	import type { Snippet } from "svelte";

	interface Props {
		header?: Snippet;
		children: Snippet;
		class?: string;
		role?: string;
		"aria-label"?: string;
	}

	let {
		header,
		children,
		class: className = "",
		role: roleAttr,
		"aria-label": ariaLabel,
	}: Props = $props();
</script>

<div
	class="rounded-[10px] bg-white shadow-[0_1px_3px_rgba(0,0,0,0.08)] dark:bg-(--color-body-dark) dark:shadow-none {className}"
	role={roleAttr}
	aria-label={ariaLabel}
>
	{#if header}
		<div class="rounded-t-[10px] border-b border-gray-200 p-[10px] dark:border-transparent dark:bg-(--color-header-dark)">
			{@render header()}
		</div>
	{/if}
	{@render children()}
</div>
