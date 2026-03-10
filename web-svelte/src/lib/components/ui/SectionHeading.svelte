<script lang="ts">
	import { faChevronDown } from "@fortawesome/free-solid-svg-icons";
	import Icon from "../Icon.svelte";

	interface Props {
		title: string;
		count?: number;
		collapsible?: boolean;
		expanded?: boolean;
		size?: "sm" | "lg";
		onclick?: () => void;
	}

	let {
		title,
		count,
		collapsible = false,
		expanded = true,
		size = "sm",
		onclick,
	}: Props = $props();
</script>

{#snippet headingContent()}
	{#if collapsible}
		<span class="inline-block text-[0.7em] mr-1.5 transition-transform duration-200" style:transform={expanded ? undefined : 'rotate(-90deg)'}>
			<Icon icon={faChevronDown} />
		</span>
	{/if}
	{title}
	{#if count != null}
		<span class="text-(--color-font-dark-muted)"> ({count})</span>
	{/if}
{/snippet}

{#if size === "lg"}
	{#if collapsible}
		<h1 class="mb-0 font-medium">
			<button type="button" class="cursor-pointer select-none hover:opacity-80 bg-transparent border-none p-0 font-inherit text-inherit text-left" onclick={onclick}>
				{@render headingContent()}
			</button>
		</h1>
	{:else}
		<h1 class="mb-0 font-medium">
			{@render headingContent()}
		</h1>
	{/if}
{:else}
	{#if collapsible}
		<h4 class="mb-3 text-2xl font-medium">
			<button type="button" class="cursor-pointer select-none hover:opacity-80 bg-transparent border-none p-0 font-inherit text-inherit text-left" onclick={onclick}>
				{@render headingContent()}
			</button>
		</h4>
	{:else}
		<h4 class="mb-3 text-2xl font-medium">
			{@render headingContent()}
		</h4>
	{/if}
{/if}
