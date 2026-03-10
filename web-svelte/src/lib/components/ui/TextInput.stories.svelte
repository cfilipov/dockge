<script module lang="ts">
	import { defineMeta } from "@storybook/addon-svelte-csf";
	import TextInput from "./TextInput.svelte";

	const { Story } = defineMeta({
		title: "UI/TextInput",
	});
</script>

<Story name="Default">
	<TextInput placeholder="Search stacks..." />
</Story>

<Story name="With Value">
	<TextInput value="Hello, world!" />
</Story>

<Story name="Disabled">
	<div class="flex flex-col gap-3">
		<TextInput placeholder="Disabled empty" disabled />
		<TextInput value="Disabled with value" disabled />
	</div>
</Story>

<Story name="Input Types">
	<div class="flex flex-col gap-3">
		<TextInput type="text" placeholder="Text" />
		<TextInput type="password" placeholder="Password" />
		<TextInput type="email" placeholder="Email" />
		<TextInput type="search" placeholder="Search" />
		<TextInput type="url" placeholder="URL" />
	</div>
</Story>
