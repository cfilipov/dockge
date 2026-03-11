<script lang="ts">
	import { Select, type SelectItemProps } from "bits-ui";
	import { faCheck } from "@fortawesome/free-solid-svg-icons";
	import Icon from "../../Icon.svelte";
	import type { Snippet } from "svelte";

	interface Props extends Omit<SelectItemProps, "children" | "child"> {
		children?: Snippet;
	}

	let {
		class: className = "",
		children,
		...rest
	}: Props = $props();
</script>

<Select.Item
	class="flex items-center gap-2 rounded px-3 py-1.5 text-sm cursor-pointer transition-colors
		hover:bg-gray-100 data-highlighted:bg-gray-100
		dark:hover:bg-white/10 dark:data-highlighted:bg-white/10
		text-(--color-font-body) dark:text-(--color-font-dark)
		{className}"
	{...rest}
>
	{#snippet child({ props, selected })}
		<div {...props} class="{props.class} flex items-center gap-2">
			<span class="w-4 shrink-0 text-center">
				{#if selected}
					<Icon icon={faCheck} class="text-[0.8em]" />
				{/if}
			</span>
			{#if children}
				{@render children()}
			{/if}
		</div>
	{/snippet}
</Select.Item>
