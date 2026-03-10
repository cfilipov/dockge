<script lang="ts">
	import Icon from "$lib/components/Icon.svelte";
	import {
		faLayerGroup,
		faCubes,
		faNetworkWired,
		faBoxArchive,
		faHardDrive,
		faTerminal,
		faPlus,
		faAngleDown,
		faMagnifyingGlass,
		faFilter,
		faXmark,
	} from "@fortawesome/free-solid-svg-icons";

	const allIcons = [
		{ name: "faLayerGroup", icon: faLayerGroup },
		{ name: "faCubes", icon: faCubes },
		{ name: "faNetworkWired", icon: faNetworkWired },
		{ name: "faBoxArchive", icon: faBoxArchive },
		{ name: "faHardDrive", icon: faHardDrive },
		{ name: "faTerminal", icon: faTerminal },
		{ name: "faPlus", icon: faPlus },
		{ name: "faAngleDown", icon: faAngleDown },
		{ name: "faMagnifyingGlass", icon: faMagnifyingGlass },
		{ name: "faFilter", icon: faFilter },
		{ name: "faXmark", icon: faXmark },
	];
</script>

<div class="flex flex-wrap gap-6 text-2xl">
	{#each allIcons as { name, icon }}
		<div class="flex flex-col items-center gap-2">
			<Icon {icon} />
			<span class="text-xs text-gray-500">{name}</span>
		</div>
	{/each}
</div>
