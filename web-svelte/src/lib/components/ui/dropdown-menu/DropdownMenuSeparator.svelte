<script lang="ts">
	import { DropdownMenu, type DropdownMenuSeparatorProps } from "bits-ui";

	let {
		class: className = "",
		...rest
	}: DropdownMenuSeparatorProps = $props();
</script>

<DropdownMenu.Separator
	class="my-1 h-px bg-gray-200 dark:bg-white/10 {className}"
	{...rest}
/>
