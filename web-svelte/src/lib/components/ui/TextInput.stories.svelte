<script module lang="ts">
	import { defineMeta } from "@storybook/addon-svelte-csf";
	import TextInput from "./TextInput.svelte";

	const { Story } = defineMeta({
		title: "UI/TextInput",
		argTypes: {
			value: { control: "text" },
			placeholder: { control: "text" },
			type: { control: "select", options: ["text", "password", "email", "search", "url"] },
			disabled: { control: "boolean" },
		},
		args: {
			value: "",
			placeholder: "Search stacks...",
			type: "text",
			disabled: false,
		},
	});
</script>

<script lang="ts">
	import { faEye, faEyeSlash, faMagnifyingGlass } from "@fortawesome/free-solid-svg-icons";
	import Icon from "../Icon.svelte";
	import Button from "./Button.svelte";
	import Chip from "./Chip.svelte";
	import IconButton from "./IconButton.svelte";

	let passwordVisible = $state(false);
</script>

<Story name="Playground">
	{#snippet template(args)}
		<TextInput {...args} />
	{/snippet}
</Story>

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

<Story name="With Right Chip">
	<TextInput placeholder="Image Update Checking" value="https://registry.example.com">
		{#snippet right()}
			<Chip label="Enabled" values={[]} />
		{/snippet}
	</TextInput>
</Story>

<Story name="With Right Button">
	<TextInput placeholder="Primary Hostname" value="dockge.example.com">
		{#snippet right()}
			<Button text="Auto Get" variant="brand" size="sm" />
		{/snippet}
	</TextInput>
</Story>

<Story name="Password with Toggle">
	<TextInput
		type={passwordVisible ? "text" : "password"}
		placeholder="Enter password"
		value="supersecret"
	>
		{#snippet right()}
			<IconButton
				icon={passwordVisible ? faEyeSlash : faEye}
				aria-label={passwordVisible ? "Hide password" : "Show password"}
				size="sm"
				onclick={() => (passwordVisible = !passwordVisible)}
			/>
		{/snippet}
	</TextInput>
</Story>

<Story name="With Left Icon">
	<TextInput placeholder="Search stacks...">
		{#snippet left()}
			<Icon icon={faMagnifyingGlass} class="text-gray-400" />
		{/snippet}
	</TextInput>
</Story>

<Story name="Combined">
	<TextInput placeholder="Search stacks...">
		{#snippet left()}
			<Icon icon={faMagnifyingGlass} class="text-gray-400" />
		{/snippet}
		{#snippet right()}
			<Button text="Search" variant="brand" size="sm" />
		{/snippet}
	</TextInput>
</Story>
