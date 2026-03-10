<script lang="ts">
	import { slide } from "svelte/transition";

	export interface Column {
		key: string;
		label: string;
		mono?: boolean;
	}

	interface Props {
		columns: Column[];
		rows: Record<string, string | number>[];
		loading?: boolean;
	}

	let { columns, rows, loading = false }: Props = $props();
</script>

{#if loading}
	<span class="text-(--color-font-dark-muted)">Loading...</span>
{:else}
	<div class="overflow-x-auto" transition:slide={{ duration: 250 }}>
		<table class="w-full border-collapse text-sm">
			<thead>
				<tr>
					{#each columns as col}
						<th class="px-4 py-2 text-xs font-semibold uppercase tracking-wider text-(--color-font-dark-muted)">{col.label}</th>
					{/each}
				</tr>
			</thead>
			<tbody>
				{#each rows as row, i}
					<tr class="{i % 2 === 0 ? 'bg-(--color-body-light) dark:bg-(--color-header-dark)' : ''}">
						{#each columns as col}
							<td class="px-4 py-2 first:rounded-l-lg last:rounded-r-lg">
								{#if col.mono}
									<code class="font-mono text-xs">{row[col.key] ?? ""}</code>
								{:else}
									{row[col.key] ?? ""}
								{/if}
							</td>
						{/each}
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{/if}
