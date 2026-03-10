<script lang="ts">
	import Badge from "./Badge.svelte";
	import type { BadgeStatus } from "./Badge.svelte";
	import Icon from "../Icon.svelte";
	import { faRocket, faArrowUp } from "@fortawesome/free-solid-svg-icons";
	import * as m from "$lib/paraglide/messages";

	type StatusLabelSize = "sm" | "md" | "lg";

	interface Props {
		status: BadgeStatus;
		name: string;
		size?: StatusLabelSize;
		href?: string;
		recreateNecessary?: boolean;
		updateAvailable?: boolean;
	}

	let {
		status,
		name,
		size = "sm",
		href,
		recreateNecessary = false,
		updateAvailable = false,
	}: Props = $props();

	// sm = sidebar list item (body text size)
	// md = card heading (Bootstrap h5 = 1.25rem)
	// lg = page title (Bootstrap h1 = 2rem)
	const sizeClasses: Record<StatusLabelSize, string> = {
		sm: "text-base",
		md: "text-[1.25rem] font-medium",
		lg: "text-[2rem] font-semibold",
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
			<Badge {status} />
			<span class="truncate">{name}</span>
		</a>
	{:else}
		<Badge {status} />
		<span class="truncate">{name}</span>
	{/if}
	{#if recreateNecessary}
		<Icon icon={faRocket} class="text-(--color-info) shrink-0" title={m.tooltipIconRecreate()} />
	{/if}
	{#if updateAvailable}
		<Icon icon={faArrowUp} class="text-(--color-info) shrink-0" title={m.tooltipIconUpdate()} />
	{/if}
</svelte:element>
