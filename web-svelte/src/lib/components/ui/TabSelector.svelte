<script lang="ts">
	import { Tabs } from "bits-ui";
	import type { IconDefinition } from "@fortawesome/fontawesome-svg-core";
	import Icon from "../Icon.svelte";

	export interface Tab {
		value: string;
		icon: IconDefinition;
		label: string;
	}

	interface Props {
		tabs: Tab[];
		value?: string;
	}

	let { tabs, value = $bindable(tabs[0]?.value) }: Props = $props();

	const TAB_WIDTH = 45;
	const activeIndex = $derived(Math.max(0, tabs.findIndex((t) => t.value === value)));
</script>

<Tabs.Root bind:value>
	<Tabs.List
		class="relative inline-flex items-stretch rounded-full bg-white shadow-[0_1px_3px_rgba(0,0,0,0.08)] dark:bg-(--color-header-dark) dark:shadow-none"
		style="padding: 3px;"
	>
		<!-- Sliding indicator -->
		<div
			class="absolute top-[3px] bottom-[3px] w-[45px] rounded-full bg-(--color-primary) shadow-sm dark:bg-brand-gradient"
			style="left: 3px; transform: translateX({activeIndex * TAB_WIDTH}px); transition: transform 200ms cubic-bezier(0.25, 0.1, 0.25, 1);"
		></div>

		{#each tabs as tab}
			<Tabs.Trigger value={tab.value}>
				{#snippet child({ props })}
					<button
						{...props}
						type="button"
						aria-label={tab.label}
						title={tab.label}
						class="relative z-10 w-[45px] h-[31px] rounded-full inline-flex items-center justify-center cursor-pointer border-0 {props['data-state'] === 'active'
							? 'text-white dark:text-(--color-font-dark-contrast)'
							: 'text-(--color-font-body) dark:text-(--color-font-dark)'}"
						style="transition: color 200ms ease;"
					>
						<Icon icon={tab.icon} />
					</button>
				{/snippet}
			</Tabs.Trigger>
		{/each}
	</Tabs.List>
</Tabs.Root>
