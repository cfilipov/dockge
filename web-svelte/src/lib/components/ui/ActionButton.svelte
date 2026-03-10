<script lang="ts">
	import Icon from "../Icon.svelte";
	import { type ActionType, type ActionColor, actionDefs } from "./action-types";

	type ActionButtonSize = "sm" | "lg";

	interface Props {
		action: ActionType;
		size?: ActionButtonSize;
		disabled?: boolean;
		onclick?: (e: MouseEvent) => void;
	}

	let { action, size = "lg", disabled = false, onclick }: Props = $props();

	const def = $derived(actionDefs[action]);

	const colorClasses: Record<ActionColor, string> = {
		brand: "bg-brand-gradient text-white hover:bg-brand-gradient-hover dark:text-(--color-font-dark-contrast)",
		secondary: "bg-[#6c757d] text-white hover:bg-[#545b62]",
		gray: "bg-[#f5f5f5] text-(--color-font-body) hover:bg-[#ededed] dark:bg-(--color-header-dark) dark:text-(--color-font-dark) dark:hover:bg-[#12161c]",
		purple: "bg-(--color-purple) text-white hover:bg-[#9170e8] dark:bg-(--color-info) dark:text-(--color-font-dark-contrast) dark:hover:bg-[#b491ff]",
		red: "bg-[#f5f5f5] text-(--color-danger) hover:bg-[#ededed] dark:bg-(--color-header-dark) dark:hover:bg-[#12161c]",
	};

	const sizeClasses: Record<ActionButtonSize, string> = {
		sm: "px-0 py-1 text-sm w-[45px] h-[31px]",
		lg: "px-5 py-[7px] text-base",
	};
</script>

<button
	type="button"
	aria-label={size === "sm" ? def.label() : undefined}
	{disabled}
	{onclick}
	class="inline-flex items-center justify-center gap-1.5 cursor-pointer transition-colors ml-px first:ml-0 disabled:opacity-40 disabled:cursor-not-allowed border-0 {colorClasses[def.color]} {sizeClasses[size]}"
>
	<Icon icon={def.icon} />
	{#if size === "lg"}
		{def.label()}
	{/if}
</button>
