<script lang="ts">
	import Header from "$lib/components/Header.svelte";
	import ComposeButton from "$lib/components/ComposeButton.svelte";
	import StacksList from "$lib/features/stacks/StacksList.svelte";
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
			<aside class="hidden flex-shrink-0 flex-col overflow-hidden px-3 pt-0 pb-4 md:flex md:w-1/3 xl:w-1/4">
				<div class="mb-4 flex items-center">
					<ComposeButton />
				</div>
				<StacksList />
			</aside>
		{/if}
		<ContentPane />
	</main>

	{#if theme.isMobile}
		<MobileNav />
	{/if}
</div>
