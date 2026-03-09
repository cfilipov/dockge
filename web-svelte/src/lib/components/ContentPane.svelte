<script lang="ts">
</script>

<div class="flex-1 overflow-y-auto px-3 pb-3">
	<div class="shadow-box p-6">
		<h1 class="mb-4 text-2xl font-bold">Welcome to Dockge</h1>
		<p class="text-muted">
			Select a stack from the sidebar to view its details.
		</p>
	</div>
</div>

<style>
	.shadow-box {
		border-radius: 10px;
		background-color: white;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
	}

	:global(.dark) .shadow-box {
		background-color: var(--color-body-dark);
		box-shadow: 0 15px 70px rgba(0, 0, 0, 0.1);
	}

	.text-muted {
		color: #6b7280;
	}

	:global(.dark) .text-muted {
		color: var(--color-font-dark-muted);
	}
</style>
