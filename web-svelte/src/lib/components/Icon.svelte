<script lang="ts">
	import type { IconDefinition } from "@fortawesome/fontawesome-svg-core";

	interface Props {
		icon: IconDefinition;
		class?: string;
	}

	let { icon, class: className = "" }: Props = $props();

	const viewBox = $derived(`0 0 ${icon.icon[0]} ${icon.icon[1]}`);
	const pathData = $derived(icon.icon[4] as string);
	const aspectWidth = $derived(`${icon.icon[0] / icon.icon[1]}em`);
</script>

<svg
	xmlns="http://www.w3.org/2000/svg"
	viewBox={viewBox}
	fill="currentColor"
	class="inline-block h-[1em] {className}"
	style:width={aspectWidth}
	aria-hidden="true"
>
	<path d={pathData} />
</svg>
