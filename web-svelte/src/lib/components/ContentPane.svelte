<script lang="ts">
	import * as m from "$lib/paraglide/messages";
</script>

<div class="flex-1 overflow-y-auto px-3 pb-3">
	<div class="shadow-box dark:bg-(--color-body-dark) dark:shadow-[0_15px_70px_rgba(0,0,0,0.1)] p-6">
		<h1 class="mb-4 text-2xl font-bold">{m.welcomeTitle()}</h1>
		<p class="text-gray-500 dark:text-(--color-font-dark-muted)">
			{m.welcomeSubtitle()}
		</p>
	</div>

	<!-- Temporary filler content for scroll testing — remove when real content exists -->
	{#each Array(20) as _, i}
		<div class="shadow-box dark:bg-(--color-body-dark) dark:shadow-[0_15px_70px_rgba(0,0,0,0.1)] mt-4 p-6">
			<h2 class="mb-2 text-lg font-semibold">Placeholder block {i + 1}</h2>
			<p class="text-gray-500 dark:text-(--color-font-dark-muted)">
				This is filler content to test scrolling behavior in the content pane.
			</p>
		</div>
	{/each}
</div>
