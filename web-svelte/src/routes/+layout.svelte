<script lang="ts">
	import "../app.css";
	import { theme } from "$lib/theme.svelte";
	import type { Snippet } from "svelte";

	let { children }: { children: Snippet } = $props();
</script>

<div class="h-full" class:dark={theme.isDark}>
	{@render children()}
</div>
