<script lang="ts">
	import Checkbox from "./ui/Checkbox.svelte";

	interface Props {
		version?: string;
		frontendVersion?: string;
		checkUpdateUrl?: string;
		showUpdateIfAvailable?: boolean;
		alsoCheckBetaRelease?: boolean;
	}

	let {
		version = "1.5.0",
		frontendVersion = "1.5.0",
		checkUpdateUrl = "https://github.com/louislam/dockge/releases",
		showUpdateIfAvailable = $bindable(true),
		alsoCheckBetaRelease = $bindable(false),
	}: Props = $props();
</script>

<div class="flex flex-col items-center gap-4 py-6">
	<img src="/icon.svg" alt="Dockge logo" style="width: 160px; height: 160px;" />

	<div class="flex flex-col items-center gap-1">
		<h2 class="text-2xl font-bold text-(--color-font-body) dark:text-(--color-font-dark)">Dockge</h2>
		<p class="text-base text-(--color-font-body) dark:text-(--color-font-dark)">Version: {version}</p>
		<p class="text-base text-gray-400">Frontend Version: {frontendVersion}</p>
	</div>

	<a
		href={checkUpdateUrl}
		target="_blank"
		rel="noopener noreferrer"
		class="text-base text-(--color-font-body) underline dark:text-(--color-font-dark)"
	>
		Check Update On GitHub
	</a>

	<div class="flex flex-col gap-2">
		<Checkbox bind:checked={showUpdateIfAvailable} label="Show update if available" />
		<Checkbox bind:checked={alsoCheckBetaRelease} label="Also check beta release" />
	</div>
</div>
