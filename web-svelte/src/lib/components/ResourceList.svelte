<script lang="ts">
	import type { Snippet } from "svelte";
	import Card from "./ui/Card.svelte";
	import IconButton from "./ui/IconButton.svelte";
	import TextInput from "./ui/TextInput.svelte";
	import { DropdownMenuRoot, DropdownMenuTrigger, DropdownMenuContent } from "./ui/dropdown-menu";
	import Icon from "./Icon.svelte";
	import * as m from "$lib/paraglide/messages";
	import { faMagnifyingGlass, faXmark, faFilter } from "@fortawesome/free-solid-svg-icons";

	interface Props {
		searchText?: string;
		filterMenu?: Snippet;
		filterActive?: boolean;
		count?: number;
		children: Snippet;
		class?: string;
	}

	let {
		searchText = $bindable(""),
		filterMenu,
		filterActive = false,
		count,
		children,
		class: className,
	}: Props = $props();
</script>

<Card class="flex min-h-0 flex-1 flex-col overflow-hidden dark:shadow-[0_15px_70px_rgba(0,0,0,0.1)] {className ?? ''}">
	{#snippet header()}
		<div class="flex items-center gap-1">
		<TextInput
			bind:value={searchText}
			placeholder={m.search()}
			autocomplete="off"
			class="flex-1"
		>
			{#snippet left()}
				<IconButton
					icon={searchText ? faXmark : faMagnifyingGlass}
					aria-label={searchText ? m.clearSearch() : m.search()}
					size="sm"
					onclick={() => { if (searchText) searchText = ""; }}
				/>
			{/snippet}
		</TextInput>
		{#if filterMenu}
			<DropdownMenuRoot>
				<DropdownMenuTrigger>
					{#snippet child({ props })}
						<button
							{...props}
							aria-label={m.filter()}
							class="inline-flex h-7 w-7 shrink-0 cursor-pointer items-center justify-center rounded-md border border-transparent text-sm transition-colors
								{filterActive ? 'text-blue-500 dark:text-blue-400' : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200'}"
						>
							<Icon icon={faFilter} />
						</button>
					{/snippet}
				</DropdownMenuTrigger>
				<DropdownMenuContent align="end">
					{@render filterMenu()}
				</DropdownMenuContent>
			</DropdownMenuRoot>
		{/if}
	</div>
	{/snippet}

	<div class="flex-1 overflow-y-auto py-[10px] pl-[10px] mr-[10px] mt-[10px] mb-[10px]">
		<div class="pr-[6px]">
			{@render children()}
		</div>
	</div>
</Card>
