<script module lang="ts">
	import { defineMeta } from "@storybook/addon-svelte-csf";
	import DataTable from "./DataTable.svelte";
	import type { Column } from "./DataTable.svelte";

	const { Story } = defineMeta({
		title: "UI/DataTable",
		argTypes: {
			columns: { control: "object" },
			rows: { control: "object" },
			loading: { control: "boolean" },
		},
		args: {
			columns: [
				{ key: "name", label: "Name" },
				{ key: "value", label: "Value" },
			] satisfies Column[],
			rows: [
				{ name: "Alpha", value: "100" },
				{ name: "Beta", value: "200" },
				{ name: "Gamma", value: "300" },
			],
			loading: false,
		},
	});
</script>

<Story name="Playground">
	{#snippet template(args)}
		<div class="shadow-box dark:bg-(--color-body-dark) dark:shadow-none p-4 max-w-2xl">
			<DataTable {...args} />
		</div>
	{/snippet}
</Story>

<Story name="Process List">
	<div class="shadow-box dark:bg-(--color-body-dark) dark:shadow-none p-4 max-w-2xl">
		<DataTable
			columns={[
				{ key: "pid", label: "PID", mono: true },
				{ key: "user", label: "User" },
				{ key: "command", label: "Command", mono: true },
			]}
			rows={[
				{ pid: "1", user: "root", command: "/sbin/init" },
				{ pid: "42", user: "root", command: "nginx: master process nginx -g daemon off;" },
				{ pid: "78", user: "www-data", command: "nginx: worker process" },
				{ pid: "112", user: "www-data", command: "nginx: worker process" },
			]}
		/>
	</div>
</Story>

<Story name="Image Layers">
	<div class="shadow-box dark:bg-(--color-body-dark) dark:shadow-none p-4 max-w-2xl">
		<DataTable
			columns={[
				{ key: "id", label: "ID", mono: true },
				{ key: "size", label: "Size" },
				{ key: "command", label: "Command", mono: true },
			]}
			rows={[
				{ id: "sha256:a1b2c3", size: "77.8 MB", command: "/bin/sh -c #(nop) ADD file:..." },
				{ id: "sha256:d4e5f6", size: "0 B", command: "/bin/sh -c #(nop) CMD [\"bash\"]" },
				{ id: "sha256:789abc", size: "23.1 MB", command: "/bin/sh -c apt-get update && apt-get install -y..." },
				{ id: "sha256:def012", size: "512 B", command: "/bin/sh -c #(nop) EXPOSE 80" },
				{ id: "sha256:345678", size: "0 B", command: "/bin/sh -c #(nop) ENTRYPOINT [\"docker-entry..." },
			]}
		/>
	</div>
</Story>

<Story name="Loading" args={{ loading: true }}>
	{#snippet template(args)}
		<div class="shadow-box dark:bg-(--color-body-dark) dark:shadow-none p-4 max-w-2xl">
			<DataTable
				columns={[
					{ key: "pid", label: "PID", mono: true },
					{ key: "user", label: "User" },
					{ key: "command", label: "Command", mono: true },
				]}
				rows={[
					{ pid: "1", user: "root", command: "/sbin/init" },
					{ pid: "42", user: "root", command: "nginx: master process nginx -g daemon off;" },
					{ pid: "78", user: "www-data", command: "nginx: worker process" },
					{ pid: "112", user: "www-data", command: "nginx: worker process" },
				]}
				loading={args.loading}
			/>
		</div>
	{/snippet}
</Story>
