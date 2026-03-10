<script lang="ts">
	import * as m from "$lib/paraglide/messages";

	export type BadgeStatus =
		| "active" | "running" | "unhealthy" | "exited"
		| "partially" | "paused" | "created" | "dead" | "down"
		| "inUse" | "unused" | "dangling";

	type BadgeSize = "sm" | "md" | "lg";

	interface Props {
		status: BadgeStatus;
		size?: BadgeSize;
	}

	let { status, size = "sm" }: Props = $props();

	const statusConfig: Record<BadgeStatus, { label: () => string; color: string }> = {
		active:    { label: m.badgeActive,    color: "bg-(--color-primary) text-white dark:bg-brand-gradient dark:text-[#020b05]" },
		running:   { label: m.badgeRunning,   color: "bg-(--color-primary) text-white dark:bg-brand-gradient dark:text-[#020b05]" },
		unhealthy: { label: m.badgeUnhealthy, color: "bg-(--color-danger) text-white" },
		exited:    { label: m.badgeExited,    color: "bg-(--color-warning) text-[#212529]" },
		partially: { label: m.badgePartially, color: "bg-[#a78bfa] text-white dark:bg-(--color-info) dark:text-[#020b05]" },
		paused:    { label: m.badgePaused,    color: "bg-[#a78bfa] text-white dark:bg-(--color-info) dark:text-[#020b05]" },
		created:   { label: m.badgeCreated,   color: "bg-gray-800 text-white" },
		dead:      { label: m.badgeDead,      color: "bg-gray-800 text-white" },
		down:      { label: m.badgeDown,      color: "bg-gray-500 text-white" },
		inUse:     { label: m.badgeInUse,     color: "bg-green-500 text-white" },
		unused:    { label: m.badgeUnused,    color: "bg-(--color-warning) text-[#212529]" },
		dangling:  { label: m.badgeDangling,  color: "bg-gray-500 text-white" },
	};

	// sm = Bootstrap .badge default (font-size .75em, padding .35em .65em)
	// md/lg scale up for card headings and page titles
	const sizeClasses: Record<BadgeSize, string> = {
		sm: "text-[0.75em] py-[0.35em] px-[0.65em]",
		md: "text-[0.85em] py-[0.4em] px-[0.75em]",
		lg: "text-[0.95em] py-[0.45em] px-[0.85em]",
	};

	const config = $derived(statusConfig[status]);
</script>

<span class="inline-block whitespace-nowrap rounded-full text-center align-baseline font-bold leading-none {config.color} {sizeClasses[size]}">{config.label()}</span>
