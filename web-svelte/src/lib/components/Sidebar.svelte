<script lang="ts">
	import Icon from "./Icon.svelte";
	import { faPlus } from "@fortawesome/free-solid-svg-icons";
	import { m } from "$lib/i18n/messages";

	const stacks = [
		{ name: "test-alpine", status: "running", services: "1 service" },
		{ name: "web-app", status: "running", services: "nginx, redis" },
		{ name: "monitoring", status: "exited", services: "grafana" },
		{ name: "blog", status: "inactive", services: "wordpress, mysql" },
	];

	let activeStack = $state("test-alpine");

	function statusColor(status: string): string {
		if (status === "running") return "bg-green-500";
		if (status === "exited") return "bg-warning";
		return "bg-gray-400";
	}
</script>

<aside class="hidden flex-shrink-0 flex-col overflow-hidden px-3 py-0 md:flex md:w-1/3 xl:w-1/4">
	<div class="mb-3 flex items-center">
		<button class="inline-flex items-center gap-0 py-[6px] px-[20px] rounded-full border border-transparent text-base font-normal text-[#020b05] bg-brand-gradient cursor-pointer hover:bg-brand-gradient-hover">
			<Icon icon={faPlus} class="mr-1" />
			Compose
		</button>
	</div>

	<div class="shadow-box dark:bg-(--color-body-dark) dark:shadow-[0_15px_70px_rgba(0,0,0,0.1)] flex-1 overflow-y-auto p-[10px]">
		<div class="pr-[6px]">
		{#each stacks as stack}
			<a
				href="/stacks/{stack.name}"
				class="flex items-center h-[46px] no-underline rounded-[10px] w-full px-2 my-[3px] text-inherit transition-none
					{activeStack === stack.name
					? 'bg-[#e8f4ff] border-l-4 border-l-(--color-primary) rounded-tl-none rounded-bl-none dark:bg-(--color-header-dark)'
					: 'hover:bg-(--color-body-light) dark:hover:bg-(--color-header-dark)'}"
				onclick={(e: MouseEvent) => {
					e.preventDefault();
					activeStack = stack.name;
				}}
			>
				<span
					class="mr-2 inline-block h-2.5 w-2.5 flex-shrink-0 rounded-full {statusColor(stack.status)}"
				></span>
				<span class="truncate">{stack.name}</span>
			</a>
		{/each}
		</div>
	</div>
</aside>
