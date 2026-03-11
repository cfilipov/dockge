<script lang="ts">
	import { faChevronDown } from "@fortawesome/free-solid-svg-icons";
	import Icon from "./Icon.svelte";
	import * as m from "$lib/paraglide/messages";
	import ActionButton from "./ui/ActionButton.svelte";
	import { type ActionType, actionDefs } from "./ui/action-types";
	import {
		DropdownMenuRoot,
		DropdownMenuTrigger,
		DropdownMenuContent,
		DropdownMenuItem,
	} from "./ui/dropdown-menu";

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
		<DropdownMenuRoot>
			<DropdownMenuTrigger>
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
			</DropdownMenuTrigger>
			<DropdownMenuContent>
				{#each overflow as action}
					{@const def = actionDefs[action]}
					<DropdownMenuItem
						variant={action === 'delete' || action === 'forceDelete' ? 'danger' : 'default'}
						onclick={() => onaction?.(action)}
					>
						<Icon icon={def.icon} />
						{def.label()}
					</DropdownMenuItem>
				{/each}
			</DropdownMenuContent>
		</DropdownMenuRoot>
	{/if}
</div>
