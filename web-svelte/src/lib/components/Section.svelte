<script lang="ts">
	import type { Snippet } from "svelte";
	import { slide } from "svelte/transition";
	import SectionHeading from "./ui/SectionHeading.svelte";

	interface Props {
		title: string;
		count?: number;
		collapsible?: boolean;
		expanded?: boolean;
		children: Snippet;
	}

	let {
		title,
		count,
		collapsible = false,
		expanded = $bindable(true),
		children,
	}: Props = $props();

	function toggle() {
		if (collapsible) {
			expanded = !expanded;
		}
	}
</script>

<div>
	<SectionHeading
		{title}
		{count}
		{collapsible}
		{expanded}
		size="sm"
		onclick={toggle}
	/>
	{#if collapsible}
		{#if expanded}
			<div transition:slide={{ duration: 250 }}>
				{@render children()}
			</div>
		{/if}
	{:else}
		{@render children()}
	{/if}
</div>
