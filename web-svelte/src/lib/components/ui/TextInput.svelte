<script lang="ts">
	import type { Snippet } from "svelte";

	interface Props {
		value?: string;
		placeholder?: string;
		type?: "text" | "password" | "email" | "search" | "url";
		disabled?: boolean;
		autocomplete?: AutoFill | undefined;
		class?: string;
		left?: Snippet;
		right?: Snippet;
	}

	let {
		value = $bindable(""),
		placeholder,
		type = "text",
		disabled = false,
		autocomplete,
		class: className = "",
		left,
		right,
	}: Props = $props();
</script>

<div
	class="flex min-h-10 items-center gap-2 rounded-full border border-gray-300 bg-white py-1 pl-1.5 pr-1.5
		has-[:focus]:border-(--color-primary) has-[:focus]:ring-1 has-[:focus]:ring-(--color-primary)
		has-[:disabled]:opacity-50 has-[:disabled]:cursor-not-allowed
		dark:border-(--color-border-dark) dark:bg-(--color-body-dark-deep) dark:text-(--color-font-dark)
		{className}"
>
	{#if left}{@render left()}{/if}
	<input
		{type}
		bind:value
		{placeholder}
		{disabled}
		{autocomplete}
		class="min-w-0 flex-1 bg-transparent outline-none placeholder:text-gray-400"
	/>
	{#if right}{@render right()}{/if}
</div>
