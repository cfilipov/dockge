<script lang="ts">
	interface Props {
		checked?: boolean;
		disabled?: boolean;
		label?: string;
		class?: string;
	}

	let {
		checked = $bindable(false),
		disabled = false,
		label,
		class: className = "",
	}: Props = $props();
</script>

<label
	class="inline-flex items-center gap-2 cursor-pointer
		{disabled ? 'opacity-50 cursor-not-allowed' : ''}
		{className}"
>
	<input
		type="checkbox"
		bind:checked
		{disabled}
		class="h-4 w-4 rounded accent-(--color-primary)"
	/>
	{#if label}<span class="text-(--color-font-body) dark:text-(--color-font-dark)">{label}</span>{/if}
</label>
