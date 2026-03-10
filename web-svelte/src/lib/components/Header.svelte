<script lang="ts">
	import Icon from "./Icon.svelte";
	import {
		faLayerGroup,
		faCubes,
		faNetworkWired,
		faBoxArchive,
		faHardDrive,
		faTerminal,
		faAngleDown,
	} from "@fortawesome/free-solid-svg-icons";
	import * as m from "$lib/paraglide/messages";

	const navItems = [
		{ label: () => m.stacks(), icon: faLayerGroup, href: "/stacks" },
		{ label: () => m.containersNav(), icon: faCubes, href: "/containers" },
		{ label: () => m.networksNav(), icon: faNetworkWired, href: "/networks" },
		{ label: () => m.imagesNav(), icon: faBoxArchive, href: "/images" },
		{ label: () => m.volumesNav(), icon: faHardDrive, href: "/volumes" },
		{ label: () => m.console(), icon: faTerminal, href: "/console" },
	];

	let activeHref = $state("/stacks");
</script>

<header class="flex flex-wrap items-center py-4 mb-4 bg-white border-b border-gray-200 dark:bg-(--color-header-dark) dark:border-(--color-header-dark)">
	<a href="/" class="flex items-center gap-2 ml-6 mr-auto no-underline">
		<img src="/icon.svg" alt="Dockge" class="w-10 h-10" />
		<span class="text-2xl font-bold text-[#111] dark:text-[#f0f6fc]">{m.dockge()}</span>
	</a>

	<nav class="order-1 w-full flex flex-wrap justify-center gap-1 px-6 pt-2 pb-0 m-0 xl:order-0 xl:w-auto xl:flex-auto xl:flex-nowrap xl:p-0">
		{#each navItems as item}
			<a
				href={item.href}
				class="flex items-center gap-[8px] px-4 py-2 rounded-full text-base no-underline transition-none
					{activeHref === item.href
					? 'text-[#020b05] bg-brand-gradient'
					: 'text-gray-500 hover:bg-gray-100 dark:text-(--color-font-dark) dark:hover:bg-(--color-body-dark-deep)'}"
				onclick={(e: MouseEvent) => {
					e.preventDefault();
					activeHref = item.href;
				}}
			>
				<Icon icon={item.icon} />
				{item.label()}
			</a>
		{/each}
	</nav>

	<div class="mr-6 shrink-0 xl:order-1">
		<button class="flex items-center gap-[6px] bg-[rgba(200,200,200,0.2)] border-none py-2 px-[0.8rem] rounded-full cursor-pointer text-inherit hover:bg-white/20" aria-label={m.userMenu()}>
			<span class="flex items-center justify-center size-6 mr-[5px] rounded-full bg-(--color-primary) text-white font-bold text-[10px]">A</span>
			<Icon icon={faAngleDown} />
		</button>
	</div>
</header>
