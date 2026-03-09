<script lang="ts">
	import Icon from "./Icon.svelte";
	import {
		faLayerGroup,
		faCubes,
		faNetworkWired,
		faTerminal,
	} from "@fortawesome/free-solid-svg-icons";
	import { m } from "$lib/i18n/messages";

	const navItems = [
		{ label: m.stacks, icon: faLayerGroup, href: "/stacks" },
		{ label: m.containersNav, icon: faCubes, href: "/containers" },
		{ label: m.networksNav, icon: faNetworkWired, href: "/networks" },
		{ label: m.console, icon: faTerminal, href: "/console" },
	];

	let activeHref = $state("/stacks");
</script>

<nav
	class="fixed bottom-0 left-0 z-50 flex w-full border-t border-gray-200 bg-white shadow-[0_-2px_10px_rgba(0,0,0,0.05)] dark:border-border-dark dark:bg-body-dark"
	style="padding-bottom: env(safe-area-inset-bottom)"
>
	{#each navItems as item}
		<a
			href={item.href}
			class="flex flex-1 flex-col items-center gap-0.5 py-2 text-xs transition-colors
				{activeHref === item.href
				? 'text-primary font-bold'
				: 'text-gray-400 dark:text-font-dark-muted'}"
			onclick={(e: MouseEvent) => {
				e.preventDefault();
				activeHref = item.href;
			}}
		>
			<Icon icon={item.icon} class="text-xl" />
			<span>{item.label}</span>
		</a>
	{/each}
</nav>
