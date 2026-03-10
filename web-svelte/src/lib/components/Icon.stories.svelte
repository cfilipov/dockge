<script module lang="ts">
	import { defineMeta } from "@storybook/addon-svelte-csf";
	import Icon from "./Icon.svelte";
	import {
		faLayerGroup,
		faCubes,
		faNetworkWired,
		faBoxArchive,
		faHardDrive,
		faTerminal,
		faPlus,
		faAngleDown,
	} from "@fortawesome/free-solid-svg-icons";

	const { Story } = defineMeta({
		title: "Components/Icon",
		component: Icon,
	});

	const allIcons = [
		{ name: "faLayerGroup", icon: faLayerGroup },
		{ name: "faCubes", icon: faCubes },
		{ name: "faNetworkWired", icon: faNetworkWired },
		{ name: "faBoxArchive", icon: faBoxArchive },
		{ name: "faHardDrive", icon: faHardDrive },
		{ name: "faTerminal", icon: faTerminal },
		{ name: "faPlus", icon: faPlus },
		{ name: "faAngleDown", icon: faAngleDown },
	];
</script>

<Story name="Default" args={{ icon: faLayerGroup }} />

<Story name="All Icons">
	<div class="flex flex-wrap gap-6 text-2xl">
		{#each allIcons as { name, icon }}
			<div class="flex flex-col items-center gap-2">
				<Icon {icon} />
				<span class="text-xs text-gray-500">{name}</span>
			</div>
		{/each}
	</div>
</Story>

<Story name="Custom Class">
	<Icon icon={faLayerGroup} class="text-4xl text-primary" />
</Story>
