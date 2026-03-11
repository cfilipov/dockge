<script lang="ts">
	import { Select, type SelectContentProps } from "bits-ui";

	let {
		class: className = "",
		children,
		...rest
	}: SelectContentProps = $props();
</script>

<Select.Content
	class="z-50 min-w-44 rounded-lg border border-[#c0c0c0] bg-white p-1 shadow-lg origin-top
		dark:border-(--color-border-dark) dark:bg-(--color-body-dark)
		data-[state=open]:animate-dropdown-in data-[state=closed]:animate-dropdown-out
		{className}"
	{...rest}
>
	{@render children?.()}
</Select.Content>
