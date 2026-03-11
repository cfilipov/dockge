<script lang="ts">
	import { Select, type SelectTriggerProps } from "bits-ui";
	import { faChevronDown } from "@fortawesome/free-solid-svg-icons";
	import Icon from "../../Icon.svelte";

	interface Props extends SelectTriggerProps {
		placeholder?: string;
	}

	let {
		placeholder = "Select...",
		class: className = "",
		children,
		...rest
	}: Props = $props();
</script>

<Select.Trigger {...rest}>
	{#snippet child({ props })}
		<button
			{...props}
			class="flex min-h-10 w-full items-center gap-2 rounded-full border border-gray-300 bg-white py-1 pl-4 pr-3
				focus:border-(--color-primary) focus:ring-1 focus:ring-(--color-primary) focus:outline-none
				disabled:opacity-50 disabled:cursor-not-allowed
				dark:border-(--color-border-dark) dark:bg-(--color-body-dark-deep) dark:text-(--color-font-dark)
				{className}"
		>
			<span class="flex-1 text-left text-sm">
				{#if children}
					{@render children()}
				{:else}
					<span class="text-gray-400">{placeholder}</span>
				{/if}
			</span>
			<Icon
				icon={faChevronDown}
				class="text-[0.7em] text-gray-400"
			/>
		</button>
	{/snippet}
</Select.Trigger>
