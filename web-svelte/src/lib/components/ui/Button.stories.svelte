<script module lang="ts">
	import { defineMeta } from "@storybook/addon-svelte-csf";
	import Button from "./Button.svelte";

	const { Story } = defineMeta({
		title: "UI/Button",
	});
</script>

<script lang="ts">
	import { faPlus, faPlay, faStop, faTrash, faDownload } from "@fortawesome/free-solid-svg-icons";
</script>

<Story name="All Variants">
	<div class="flex flex-wrap items-center gap-2">
		<Button text="primary" variant="primary" />
		<Button text="secondary" variant="secondary" />
		<Button text="outline" variant="outline" />
		<Button text="ghost" variant="ghost" />
		<Button text="danger" variant="danger" />
		<Button text="brand" variant="brand" />
	</div>
</Story>

<Story name="All Sizes">
	<div class="flex flex-wrap items-center gap-2">
		<Button text="sm" variant="primary" size="sm" />
		<Button text="md" variant="primary" size="md" />
		<Button text="lg" variant="primary" size="lg" />
	</div>
</Story>

<Story name="With Icons">
	<div class="flex flex-wrap items-center gap-2">
		<Button text="Compose" variant="brand" icon={faPlus} />
		<Button text="Start" variant="primary" icon={faPlay} />
		<Button text="Stop" variant="secondary" icon={faStop} />
		<Button text="Delete" variant="danger" icon={faTrash} />
		<Button text="Download" variant="outline" icon={faDownload} />
	</div>
</Story>

<Story name="Disabled">
	<div class="flex flex-wrap items-center gap-2">
		<Button text="primary" variant="primary" disabled />
		<Button text="secondary" variant="secondary" disabled />
		<Button text="outline" variant="outline" disabled />
		<Button text="ghost" variant="ghost" disabled />
		<Button text="danger" variant="danger" disabled />
		<Button text="brand" variant="brand" disabled />
	</div>
</Story>

<Story name="Size and Variant Matrix">
	<div class="flex flex-col gap-3">
		<div>
			<span class="mr-2 text-sm font-medium text-gray-500">sm</span>
			<span class="inline-flex flex-wrap items-center gap-2">
				<Button text="primary" variant="primary" size="sm" />
				<Button text="secondary" variant="secondary" size="sm" />
				<Button text="outline" variant="outline" size="sm" />
				<Button text="ghost" variant="ghost" size="sm" />
				<Button text="danger" variant="danger" size="sm" />
				<Button text="brand" variant="brand" size="sm" />
			</span>
		</div>
		<div>
			<span class="mr-2 text-sm font-medium text-gray-500">md</span>
			<span class="inline-flex flex-wrap items-center gap-2">
				<Button text="primary" variant="primary" size="md" />
				<Button text="secondary" variant="secondary" size="md" />
				<Button text="outline" variant="outline" size="md" />
				<Button text="ghost" variant="ghost" size="md" />
				<Button text="danger" variant="danger" size="md" />
				<Button text="brand" variant="brand" size="md" />
			</span>
		</div>
		<div>
			<span class="mr-2 text-sm font-medium text-gray-500">lg</span>
			<span class="inline-flex flex-wrap items-center gap-2">
				<Button text="primary" variant="primary" size="lg" />
				<Button text="secondary" variant="secondary" size="lg" />
				<Button text="outline" variant="outline" size="lg" />
				<Button text="ghost" variant="ghost" size="lg" />
				<Button text="danger" variant="danger" size="lg" />
				<Button text="brand" variant="brand" size="lg" />
			</span>
		</div>
	</div>
</Story>
