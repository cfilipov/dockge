<script lang="ts">
	import type { IconDefinition } from "@fortawesome/fontawesome-svg-core";
	import Icon from "../Icon.svelte";

	type IconButtonVariant = "primary" | "secondary" | "outline" | "ghost" | "danger";
	type IconButtonSize = "sm" | "md" | "lg";

	interface Props {
		icon: IconDefinition;
		"aria-label": string;
		variant?: IconButtonVariant;
		size?: IconButtonSize;
		disabled?: boolean;
		onclick?: (e: MouseEvent) => void;
	}

	let {
		icon,
		"aria-label": ariaLabel,
		variant = "ghost",
		size = "md",
		disabled = false,
		onclick,
	}: Props = $props();

	const variantClasses: Record<IconButtonVariant, string> = {
		primary: "bg-(--color-primary) text-white border border-transparent hover:brightness-110",
		secondary: "bg-gray-500 text-white border border-transparent hover:bg-gray-600",
		outline: "bg-transparent text-(--color-primary) border border-(--color-primary) hover:bg-(--color-primary)/10",
		ghost: "bg-transparent text-gray-500 border border-transparent hover:bg-gray-200 dark:text-gray-400 dark:hover:bg-gray-700",
		danger: "bg-(--color-danger) text-white border border-transparent hover:brightness-110",
	};

	const sizeClasses: Record<IconButtonSize, string> = {
		sm: "text-sm w-7 h-7",
		md: "text-base w-9 h-9",
		lg: "text-lg w-11 h-11",
	};
</script>

<button
	type="button"
	aria-label={ariaLabel}
	{disabled}
	{onclick}
	class="inline-flex items-center justify-center rounded-md cursor-pointer transition-colors disabled:opacity-50 disabled:cursor-not-allowed {variantClasses[variant]} {sizeClasses[size]}"
>
	<Icon {icon} />
</button>
