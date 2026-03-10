<script lang="ts">
	import type { IconDefinition } from "@fortawesome/fontawesome-svg-core";
	import Icon from "../Icon.svelte";

	type ButtonVariant = "primary" | "secondary" | "outline" | "ghost" | "danger" | "brand";
	type ButtonSize = "sm" | "md" | "lg";

	interface Props {
		text: string;
		icon?: IconDefinition;
		variant?: ButtonVariant;
		size?: ButtonSize;
		disabled?: boolean;
		type?: "button" | "submit" | "reset";
		onclick?: (e: MouseEvent) => void;
	}

	let {
		text,
		icon,
		variant = "primary",
		size = "md",
		disabled = false,
		type = "button",
		onclick,
	}: Props = $props();

	const variantClasses: Record<ButtonVariant, string> = {
		primary: "bg-(--color-primary) text-white border border-transparent hover:brightness-110",
		secondary: "bg-gray-500 text-white border border-transparent hover:bg-gray-600",
		outline: "bg-transparent text-(--color-primary) border border-(--color-primary) hover:bg-(--color-primary)/10",
		ghost: "bg-transparent text-(--color-primary) border border-transparent hover:bg-(--color-primary)/10",
		danger: "bg-(--color-danger) text-white border border-transparent hover:brightness-110",
		brand: "bg-brand-gradient text-[#020b05] border border-transparent hover:bg-brand-gradient-hover",
	};

	const sizeClasses: Record<ButtonSize, string> = {
		sm: "text-sm px-3 py-1",
		md: "text-base px-4 py-1.5",
		lg: "text-lg px-5 py-2",
	};

	const roundingClass = $derived(variant === "brand" ? "rounded-full" : "rounded-md");
</script>

<button
	{type}
	{disabled}
	{onclick}
	class="inline-flex items-center gap-1.5 font-normal cursor-pointer transition-colors disabled:opacity-50 disabled:cursor-not-allowed {variantClasses[variant]} {sizeClasses[size]} {roundingClass}"
>
	{#if icon}
		<Icon {icon} />
	{/if}
	{text}
</button>
