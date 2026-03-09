<script lang="ts">
	import Header from "$lib/components/Header.svelte";
	import Sidebar from "$lib/components/Sidebar.svelte";
	import ContentPane from "$lib/components/ContentPane.svelte";
	import MobileNav from "$lib/components/MobileNav.svelte";
	import { theme } from "$lib/theme.svelte";
</script>

<div class="flex h-full flex-col overflow-hidden">
	{#if !theme.isMobile}
		<Header />
	{/if}

	<main class="mx-auto flex min-h-0 flex-1" style="width: 98%">
		{#if !theme.isMobile}
			<Sidebar />
		{/if}
		<ContentPane />
	</main>

	{#if theme.isMobile}
		<MobileNav />
	{/if}
</div>
