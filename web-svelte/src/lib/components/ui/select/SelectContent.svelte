<script lang="ts">
	import { Select, type SelectContentStaticProps } from "bits-ui";

	let {
		class: className = "",
		children,
		...rest
	}: SelectContentStaticProps = $props();
</script>

<Select.ContentStatic forceMount {...rest}>
	{#snippet child({ props, open })}
		<div
			class="grid transition-[grid-template-rows] duration-200
				{open ? 'grid-rows-[1fr]' : 'grid-rows-[0fr]'}
				{className}"
		>
			<div {...props} class="overflow-hidden min-h-0">
				<div class="mx-3 border-t border-gray-200 dark:border-(--color-border-dark)"></div>
				<div class="p-1">
					{@render children?.()}
				</div>
			</div>
		</div>
	{/snippet}
</Select.ContentStatic>
