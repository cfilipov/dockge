<script lang="ts">
	import type { Snippet } from "svelte";

	interface Props {
		children: Snippet;
		class?: string;
	}

	let { children, class: className = "" }: Props = $props();
</script>

<div class="rounded-[10px] p-3 bg-(--color-body-light) dark:bg-(--color-header-dark) {className}">
	{@render children()}
</div>
