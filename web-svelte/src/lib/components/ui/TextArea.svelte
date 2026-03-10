<script lang="ts">
	interface Props {
		value?: string;
		placeholder?: string;
		rows?: number;
		disabled?: boolean;
		class?: string;
	}

	let {
		value = $bindable(""),
		placeholder,
		rows = 3,
		disabled = false,
		class: className = "",
	}: Props = $props();
</script>

<textarea
	bind:value
	{placeholder}
	{rows}
	{disabled}
	class="shadow-box min-w-0 w-full rounded-2xl border-none px-4 py-3 outline-none resize-y
		text-[15px] placeholder:text-gray-400
		disabled:opacity-50 disabled:cursor-not-allowed
		font-mono dark:bg-(--color-body-dark) dark:text-(--color-font-dark) dark:shadow-lg
		{className}"
></textarea>
