<script module lang="ts">
	import { defineMeta } from "@storybook/addon-svelte-csf";
	import Button from "./Button.svelte";

	const { Story } = defineMeta({
		title: "UI/Button",
		argTypes: {
			text: { control: "text" },
			variant: { control: "select", options: ["ghost", "brand"] },
			size: { control: "select", options: ["sm", "md", "lg"] },
			disabled: { control: "boolean" },
		},
		args: {
			text: "Button",
			variant: "ghost",
			size: "md",
			disabled: false,
		},
	});
</script>

<script lang="ts">
	import { faPlus } from "@fortawesome/free-solid-svg-icons";
</script>

<Story name="Playground">
	{#snippet template(args)}
		<Button {...args} icon={faPlus} />
	{/snippet}
</Story>

<Story name="All Variants">
	<div class="flex flex-wrap items-center gap-2">
		<Button text="ghost" variant="ghost" />
		<Button text="brand" variant="brand" />
	</div>
</Story>

<Story name="All Sizes">
	<div class="flex flex-wrap items-center gap-2">
		<Button text="sm" size="sm" />
		<Button text="md" size="md" />
		<Button text="lg" size="lg" />
	</div>
</Story>

<Story name="With Icons">
	<div class="flex flex-wrap items-center gap-2">
		<Button text="Compose" variant="brand" icon={faPlus} />
		<Button text="Ghost" variant="ghost" icon={faPlus} />
	</div>
</Story>

<Story name="Disabled">
	<div class="flex flex-wrap items-center gap-2">
		<Button text="ghost" variant="ghost" disabled />
		<Button text="brand" variant="brand" disabled />
	</div>
</Story>

<Story name="Size and Variant Matrix">
	<div class="flex flex-col gap-3">
		<div>
			<span class="mr-2 text-sm font-medium text-gray-500">sm</span>
			<span class="inline-flex flex-wrap items-center gap-2">
				<Button text="ghost" variant="ghost" size="sm" />
				<Button text="brand" variant="brand" size="sm" />
			</span>
		</div>
		<div>
			<span class="mr-2 text-sm font-medium text-gray-500">md</span>
			<span class="inline-flex flex-wrap items-center gap-2">
				<Button text="ghost" variant="ghost" size="md" />
				<Button text="brand" variant="brand" size="md" />
			</span>
		</div>
		<div>
			<span class="mr-2 text-sm font-medium text-gray-500">lg</span>
			<span class="inline-flex flex-wrap items-center gap-2">
				<Button text="ghost" variant="ghost" size="lg" />
				<Button text="brand" variant="brand" size="lg" />
			</span>
		</div>
	</div>
</Story>
