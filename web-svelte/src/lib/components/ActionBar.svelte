<script lang="ts">
	import { DropdownMenu } from "bits-ui";
	import { faChevronDown } from "@fortawesome/free-solid-svg-icons";
	import Icon from "./Icon.svelte";
	import * as m from "$lib/paraglide/messages";
	import ActionButton from "./ui/ActionButton.svelte";
	import { type ActionType, actionDefs } from "./ui/action-types";

	type ActionBarSize = "sm" | "lg";

	interface Props {
		actions: ActionType[];
		overflow?: ActionType[];
		size?: ActionBarSize;
		processing?: boolean;
		onaction?: (action: ActionType) => void;
	}

	let {
		actions,
		overflow = [],
		size = "lg",
		processing = false,
		onaction,
	}: Props = $props();

	const triggerSizeClasses: Record<ActionBarSize, string> = {
		sm: "px-0 py-1 text-sm w-[45px] h-[31px]",
		lg: "px-5 py-[7px] text-base",
	};
</script>

<div
	class="inline-flex items-stretch rounded-full overflow-hidden bg-[#c0c0c0] shadow-[0_1px_3px_rgba(0,0,0,0.08)] dark:bg-(--color-body-dark) dark:shadow-none"
	role="group"
>
	{#each actions as action}
		<ActionButton
			{action}
			{size}
			disabled={processing}
			onclick={() => onaction?.(action)}
		/>
	{/each}
	{#if overflow.length > 0}
		<DropdownMenu.Root>
			<DropdownMenu.Trigger>
				{#snippet child({ props })}
					<button
						{...props}
						type="button"
						aria-label={m.moreActions()}
						disabled={processing}
						class="inline-flex items-center justify-center cursor-pointer transition-colors ml-px border-0 bg-[#f5f5f5] text-(--color-font-body) hover:bg-[#ededed] dark:bg-(--color-header-dark) dark:text-(--color-font-dark) dark:hover:bg-[#12161c] disabled:opacity-40 disabled:cursor-not-allowed {triggerSizeClasses[size]}"
					>
						<Icon icon={faChevronDown} class="text-[0.7em]" />
					</button>
				{/snippet}
			</DropdownMenu.Trigger>
			<DropdownMenu.Content
				class="z-50 min-w-44 rounded-lg border border-[#c0c0c0] bg-white p-1 shadow-lg origin-top dark:border-(--color-border-dark) dark:bg-(--color-body-dark) data-[state=open]:animate-dropdown-in data-[state=closed]:animate-dropdown-out"
			>
				{#each overflow as action}
					{@const def = actionDefs[action]}
					<DropdownMenu.Item
						class="flex items-center gap-2 rounded px-3 py-1.5 text-sm cursor-pointer transition-colors hover:bg-gray-100 data-highlighted:bg-gray-100 dark:hover:bg-white/10 dark:data-highlighted:bg-white/10 {action === 'delete' || action === 'forceDelete' ? 'text-(--color-danger)' : 'text-(--color-font-body) dark:text-(--color-font-dark)'}"
						onclick={() => onaction?.(action)}
					>
						<Icon icon={def.icon} />
						{def.label()}
					</DropdownMenu.Item>
				{/each}
			</DropdownMenu.Content>
		</DropdownMenu.Root>
	{/if}
</div>
