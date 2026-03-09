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
		<button class="compose-btn">

			<Icon icon={faPlus} class="mr-1" />
			Compose
		</button>
	</div>

	<div class="shadow-box flex-1 overflow-y-auto">
		<div class="stack-list">
		{#each stacks as stack}
			<a
				href="/stacks/{stack.name}"
				class="item {activeStack === stack.name ? 'active' : ''}"
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

<style>
	.compose-btn {
		display: inline-flex;
		align-items: center;
		gap: 0;
		padding: 6px 20px;
		border-radius: 50rem;
		border: 1px solid transparent;
		font-size: 16px;
		font-weight: 400;
		color: #020b05;
		background: linear-gradient(135deg, #74c2ff 0%, #74c2ff 75%, #86e6a9);
		cursor: pointer;
	}

	.compose-btn:hover {
		background: linear-gradient(135deg, #74c2ff 0%, #74c2ff 50%, #86e6a9);
	}

	.shadow-box {
		padding: 10px;
		border-radius: 10px;
		background-color: white;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
	}

	:global(.dark) .shadow-box {
		background-color: var(--color-body-dark);
		box-shadow: 0 15px 70px rgba(0, 0, 0, 0.1);
	}

	.stack-list {
		padding-right: 6px;
	}

	.item {
		display: flex;
		align-items: center;
		height: 46px;
		text-decoration: none;
		border-radius: 10px;
		width: 100%;
		padding: 0 8px;
		margin: 3px 0;
		color: inherit;
		transition: none;
	}

	.item:hover {
		background-color: #f0f2f5;
	}

	:global(.dark) .item:hover {
		background-color: var(--color-header-dark);
	}

	.item.active {
		background-color: #e8f4ff;
		border-left: 4px solid var(--color-primary);
		border-top-left-radius: 0;
		border-bottom-left-radius: 0;
	}

	:global(.dark) .item.active {
		background-color: var(--color-header-dark);
	}
</style>
