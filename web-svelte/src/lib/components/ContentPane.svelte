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
</div>
