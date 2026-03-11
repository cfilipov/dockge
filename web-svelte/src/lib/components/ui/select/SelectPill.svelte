<script lang="ts">
	import type { Snippet } from "svelte";

	interface Props {
		class?: string;
		children?: Snippet;
	}

	let {
		class: className = "",
		children,
	}: Props = $props();
</script>

<div
	class="border border-gray-300 bg-white
		rounded-xl
		has-[:focus]:border-(--color-primary) has-[:focus]:ring-1 has-[:focus]:ring-(--color-primary)
		has-[:disabled]:opacity-50 has-[:disabled]:cursor-not-allowed
		dark:border-(--color-border-dark) dark:bg-(--color-body-dark-deep) dark:text-(--color-font-dark)
		{className}"
>
	{#if children}{@render children()}{/if}
</div>
