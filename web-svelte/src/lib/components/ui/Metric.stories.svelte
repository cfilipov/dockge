<script module lang="ts">
	import { defineMeta } from "@storybook/addon-svelte-csf";
	import Metric from "./Metric.svelte";

	const { Story } = defineMeta({
		title: "UI/Metric",
		argTypes: {
			label: { control: "text" },
			value1: { control: "text" },
			unit1: { control: "text" },
			tag1: { control: "text" },
			value2: { control: "text" },
			unit2: { control: "text" },
			tag2: { control: "text" },
			loading: { control: "boolean" },
		},
		args: {
			label: "CPU",
			value1: "1.60",
			unit1: "%",
			tag1: "usage",
			loading: false,
		},
	});
</script>

<Story name="Playground">
	{#snippet template(args)}
		<Metric {...args} />
	{/snippet}
</Story>

<Story name="CPU">
	<div class="p-4 w-52">
		<Metric label="CPU" value1="1.60" unit1="%" tag1="usage" />
	</div>
</Story>

<Story name="Memory">
	<div class="p-4 w-52">
		<Metric label="Memory" value1="178.0" unit1="MiB" tag1="8.69% used" value2="2.0" unit2="GiB" tag2="avail." />
	</div>
</Story>

<Story name="Block I/O">
	<div class="p-4 w-52">
		<Metric label="Block I/O" value1="3.2" unit1="MiB" tag1="read" value2="327.2" unit2="KiB" tag2="write" />
	</div>
</Story>

<Story name="Network I/O">
	<div class="p-4 w-52">
		<Metric label="Network I/O" value1="50.0" unit1="KiB" tag1="rx" value2="33.2" unit2="KiB" tag2="tx" />
	</div>
</Story>

<Story name="Loading">
	<div class="p-4 w-52">
		<Metric label="CPU" value1="--" unit1="%" tag1="usage" loading />
	</div>
</Story>

<Story name="All Metrics Row">
	<div class="p-4">
		<div class="shadow-box dark:bg-[#0d1117] dark:shadow-none p-5 rounded-[10px] max-w-[920px]">
			<div class="grid grid-cols-4 gap-4">
				<Metric label="CPU" value1="1.60" unit1="%" tag1="usage" />
				<Metric label="Memory" value1="178.0" unit1="MiB" tag1="8.69% used" value2="2.0" unit2="GiB" tag2="avail." />
				<Metric label="Block I/O" value1="3.2" unit1="MiB" tag1="read" value2="327.2" unit2="KiB" tag2="write" />
				<Metric label="Network I/O" value1="50.0" unit1="KiB" tag1="rx" value2="33.2" unit2="KiB" tag2="tx" />
			</div>
		</div>
	</div>
</Story>
