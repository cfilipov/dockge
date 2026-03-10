<script lang="ts">
	import Badge from "./Badge.svelte";
	import type { BadgeStatus } from "./Badge.svelte";

	type StatusLabelSize = "sm" | "md" | "lg";

	interface Props {
		status: BadgeStatus;
		name: string;
		size?: StatusLabelSize;
		href?: string;
	}

	let {
		status,
		name,
		size = "sm",
		href,
	}: Props = $props();

	const sizeClasses: Record<StatusLabelSize, string> = {
		sm: "text-base",
		md: "text-lg font-medium",
		lg: "text-2xl font-semibold",
	};

	const tagMap: Record<StatusLabelSize, string> = {
		sm: "span",
		md: "h5",
		lg: "h1",
	};

	const tag = $derived(tagMap[size]);
</script>

<svelte:element this={tag} class="inline-flex min-w-0 items-center gap-2 {sizeClasses[size]}">
	{#if href}
		<a {href} class="contents no-underline text-inherit">
			<Badge {status} {size} />
			<span class="truncate">{name}</span>
		</a>
	{:else}
		<Badge {status} {size} />
		<span class="truncate">{name}</span>
	{/if}
</svelte:element>
