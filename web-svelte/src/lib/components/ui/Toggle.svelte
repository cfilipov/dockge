<script lang="ts">
	import { Switch } from "bits-ui";

	interface Props {
		checked?: boolean;
		disabled?: boolean;
		label?: string;
		onCheckedChange?: (checked: boolean) => void;
		class?: string;
	}

	let {
		checked = $bindable(false),
		disabled = false,
		label,
		onCheckedChange,
		class: className = "",
	}: Props = $props();
</script>

{#snippet toggle()}
	<Switch.Root
		bind:checked
		{disabled}
		{onCheckedChange}
		class="inline-flex h-5 w-9 shrink-0 cursor-pointer items-center rounded-full transition-colors
			bg-gray-300 data-[state=checked]:bg-(--color-primary)
			dark:bg-gray-600 dark:data-[state=checked]:bg-(--color-primary)
			disabled:opacity-50 disabled:cursor-not-allowed
			{className}"
	>
		<Switch.Thumb
			class="pointer-events-none block h-4 w-4 rounded-full bg-white shadow-sm transition-transform
				translate-x-0.5 data-[state=checked]:translate-x-4"
		/>
	</Switch.Root>
{/snippet}

{#if label}
	<label
		class="inline-flex items-center gap-2 text-base text-(--color-font-body) dark:text-(--color-font-dark)
			{disabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'}"
	>
		{@render toggle()}
		{label}
	</label>
{:else}
	{@render toggle()}
{/if}
