<script lang="ts">
	interface Props {
		value?: string;
		placeholder?: string;
		type?: "text" | "password" | "email" | "search" | "url";
		disabled?: boolean;
		autocomplete?: AutoFill | undefined;
		class?: string;
	}

	let {
		value = $bindable(""),
		placeholder,
		type = "text",
		disabled = false,
		autocomplete,
		class: className = "",
	}: Props = $props();
</script>

<input
	{type}
	bind:value
	{placeholder}
	{disabled}
	{autocomplete}
	class="min-w-0 rounded-full border border-gray-300 bg-white px-3 py-1.5 outline-none
		focus:border-(--color-primary) focus:ring-1 focus:ring-(--color-primary)
		placeholder:text-gray-400
		disabled:opacity-50 disabled:cursor-not-allowed
		dark:border-(--color-border-dark) dark:bg-(--color-body-dark-deep) dark:text-(--color-font-dark)
		{className}"
/>
