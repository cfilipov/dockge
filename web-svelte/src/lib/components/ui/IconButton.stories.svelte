<script module lang="ts">
	import { defineMeta } from "@storybook/addon-svelte-csf";
	import IconButton from "./IconButton.svelte";

	const { Story } = defineMeta({
		title: "UI/IconButton",
	});
</script>

<script lang="ts">
	import { faXmark, faFilter, faEllipsisVertical, faGear, faTrash } from "@fortawesome/free-solid-svg-icons";
</script>

<Story name="All Variants">
	<div class="flex flex-wrap items-center gap-2">
		<IconButton icon={faGear} aria-label="Settings" variant="primary" />
		<IconButton icon={faGear} aria-label="Settings" variant="secondary" />
		<IconButton icon={faGear} aria-label="Settings" variant="outline" />
		<IconButton icon={faGear} aria-label="Settings" variant="ghost" />
		<IconButton icon={faGear} aria-label="Settings" variant="danger" />
	</div>
</Story>

<Story name="All Sizes">
	<div class="flex flex-wrap items-center gap-2">
		<IconButton icon={faGear} aria-label="Settings" size="sm" />
		<IconButton icon={faGear} aria-label="Settings" size="md" />
		<IconButton icon={faGear} aria-label="Settings" size="lg" />
	</div>
</Story>

<Story name="Common Actions">
	<div class="flex flex-wrap items-center gap-2">
		<IconButton icon={faXmark} aria-label="Close" variant="ghost" />
		<IconButton icon={faFilter} aria-label="Filter" variant="ghost" />
		<IconButton icon={faEllipsisVertical} aria-label="More actions" variant="ghost" />
		<IconButton icon={faGear} aria-label="Settings" variant="outline" />
		<IconButton icon={faTrash} aria-label="Delete" variant="danger" />
	</div>
</Story>

<Story name="Disabled">
	<div class="flex flex-wrap items-center gap-2">
		<IconButton icon={faGear} aria-label="Settings" variant="primary" disabled />
		<IconButton icon={faGear} aria-label="Settings" variant="secondary" disabled />
		<IconButton icon={faGear} aria-label="Settings" variant="outline" disabled />
		<IconButton icon={faGear} aria-label="Settings" variant="ghost" disabled />
		<IconButton icon={faGear} aria-label="Settings" variant="danger" disabled />
	</div>
</Story>
