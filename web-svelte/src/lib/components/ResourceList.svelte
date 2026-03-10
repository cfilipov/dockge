<script lang="ts">
	import Icon from "./Icon.svelte";
	import { faMagnifyingGlass, faXmark, faFilter } from "@fortawesome/free-solid-svg-icons";

	const stacks = [
		{ name: "test-alpine", status: "running", services: "1 service" },
		{ name: "web-app", status: "running", services: "nginx, redis" },
		{ name: "monitoring", status: "exited", services: "grafana" },
		{ name: "blog", status: "inactive", services: "wordpress, mysql" },
	];

	let activeStack = $state("test-alpine");
	let searchText = $state("");

	function statusColor(status: string): string {
		if (status === "running") return "bg-green-500";
		if (status === "exited") return "bg-warning";
		return "bg-gray-400";
	}
</script>

<div class="shadow-box dark:bg-(--color-body-dark) dark:shadow-[0_15px_70px_rgba(0,0,0,0.1)] flex min-h-0 flex-1 flex-col overflow-hidden">
	<!-- Header with search + filter -->
	<div class="flex items-center border-b border-gray-200 rounded-t-[10px] px-[5px] py-2 dark:border-transparent dark:bg-(--color-header-dark)">
		<button
			class="shrink-0 border-none bg-transparent px-2.5 py-2.5 text-gray-400 cursor-default"
			class:cursor-pointer={searchText !== ""}
			aria-label={searchText ? "Clear search" : "Search"}
			onclick={() => { if (searchText) searchText = ""; }}
		>
			{#if searchText}
				<Icon icon={faXmark} />
			{:else}
				<Icon icon={faMagnifyingGlass} />
			{/if}
		</button>
		<input
			type="text"
			bind:value={searchText}
			placeholder="Search"
			autocomplete="off"
			class="min-w-0 max-w-60 flex-1 rounded-full border border-gray-300 bg-white px-3 py-1.5 outline-none focus:border-(--color-primary) focus:ring-1 focus:ring-(--color-primary) placeholder:text-gray-400 dark:border-(--color-border-dark) dark:bg-(--color-body-dark-deep) dark:text-(--color-font-dark)"
		/>
		<button
			class="shrink-0 border border-transparent bg-transparent px-2.5 py-2.5 text-[var(--color-font-dark-muted)] cursor-pointer rounded hover:text-[var(--color-font-dark)]"
			aria-label="Filter"
		>
			<Icon icon={faFilter} />
		</button>
	</div>

	<!-- Stack list -->
	<div class="flex-1 overflow-y-auto p-[10px]">
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
</div>
