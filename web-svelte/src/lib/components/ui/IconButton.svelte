<script lang="ts">
	import type { IconDefinition } from "@fortawesome/fontawesome-svg-core";
	import Icon from "../Icon.svelte";

	type IconButtonSize = "sm" | "md" | "lg";

	interface Props {
		icon: IconDefinition;
		"aria-label": string;
		size?: IconButtonSize;
		disabled?: boolean;
		onclick?: (e: MouseEvent) => void;
	}

	let {
		icon,
		"aria-label": ariaLabel,
		size = "md",
		disabled = false,
		onclick,
	}: Props = $props();

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
	class="inline-flex items-center justify-center rounded-md cursor-pointer transition-colors disabled:opacity-50 disabled:cursor-not-allowed bg-transparent text-gray-500 border border-transparent hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 {sizeClasses[size]}"
>
	<Icon {icon} />
</button>
