<script lang="ts">
	import { DropdownMenu, type DropdownMenuItemProps } from "bits-ui";

	interface Props extends DropdownMenuItemProps {
		variant?: "default" | "danger";
	}

	let {
		variant = "default",
		class: className = "",
		children,
		...rest
	}: Props = $props();

	const variantClasses: Record<string, string> = {
		default: "text-(--color-font-body) dark:text-(--color-font-dark)",
		danger: "text-(--color-danger)",
	};
</script>

<DropdownMenu.Item
	class="flex items-center gap-2 rounded px-3 py-1.5 text-sm cursor-pointer transition-colors
		hover:bg-gray-100 data-highlighted:bg-gray-100
		dark:hover:bg-white/10 dark:data-highlighted:bg-white/10
		{variantClasses[variant]}
		{className}"
	{...rest}
>
	{@render children?.()}
</DropdownMenu.Item>
