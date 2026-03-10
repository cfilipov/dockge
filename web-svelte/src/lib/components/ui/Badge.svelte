<script lang="ts">
	import * as m from "$lib/paraglide/messages";

	export type BadgeStatus =
		| "active" | "running" | "unhealthy" | "exited"
		| "partially" | "paused" | "created" | "dead" | "down"
		| "inUse" | "unused" | "dangling";

	interface Props {
		status: BadgeStatus;
	}

	let { status }: Props = $props();

	const statusConfig: Record<BadgeStatus, { label: () => string; color: string }> = {
		active:    { label: m.badgeActive,    color: "bg-(--color-primary) text-white dark:bg-brand-gradient dark:text-(--color-font-dark-contrast)" },
		running:   { label: m.badgeRunning,   color: "bg-(--color-primary) text-white dark:bg-brand-gradient dark:text-(--color-font-dark-contrast)" },
		unhealthy: { label: m.badgeUnhealthy, color: "bg-(--color-danger) text-white" },
		exited:    { label: m.badgeExited,    color: "bg-(--color-warning) text-(--color-font-dark-body)" },
		partially: { label: m.badgePartially, color: "bg-(--color-purple) text-white dark:bg-(--color-info) dark:text-(--color-font-dark-contrast)" },
		paused:    { label: m.badgePaused,    color: "bg-(--color-purple) text-white dark:bg-(--color-info) dark:text-(--color-font-dark-contrast)" },
		created:   { label: m.badgeCreated,   color: "bg-gray-800 text-white" },
		dead:      { label: m.badgeDead,      color: "bg-gray-800 text-white" },
		down:      { label: m.badgeDown,      color: "bg-gray-500 text-white" },
		inUse:     { label: m.badgeInUse,     color: "bg-green-500 text-white" },
		unused:    { label: m.badgeUnused,    color: "bg-(--color-warning) text-(--color-font-dark-body)" },
		dangling:  { label: m.badgeDangling,  color: "bg-gray-500 text-white" },
	};

	const config = $derived(statusConfig[status]);
</script>

<!-- Uses em units so the badge scales with the parent's font size (matches Bootstrap .badge) -->
<span class="inline-block whitespace-nowrap rounded-full text-center align-baseline font-bold leading-none text-[0.75em] py-[0.35em] px-[0.65em] {config.color}">{config.label()}</span>
