<script lang="ts">
	import ResourceList from "$lib/components/ResourceList.svelte";
	import ListItem from "$lib/components/ui/ListItem.svelte";
	import StatusLabel from "$lib/components/ui/StatusLabel.svelte";
	import Checkbox from "$lib/components/ui/Checkbox.svelte";
	import type { BadgeStatus } from "$lib/components/ui/Badge.svelte";
	import * as m from "$lib/paraglide/messages";

	type FilterKey = "unhealthy" | "active" | "partially" | "exited" | "down" | "update" | "unmanaged";

	interface DemoStack {
		name: string;
		status: BadgeStatus;
		services: string;
		updateAvailable: boolean;
		recreateNecessary: boolean;
		managed: boolean;
	}

	const stackNames = [
		"test-alpine", "web-app", "monitoring", "blog", "postgres-db", "redis-cache",
		"nginx-proxy", "traefik-lb", "portainer", "grafana-stack", "prometheus",
		"loki-logging", "minio-storage", "vault-secrets", "consul-discovery",
		"keycloak-auth", "gitea-forge", "drone-ci", "harbor-registry", "argocd",
		"jellyfin", "plex-media", "sonarr", "radarr", "prowlarr", "bazarr",
		"jackett", "lidarr", "readarr", "overseerr", "tautulli", "sabnzbd",
		"qbittorrent", "deluge", "transmission", "wireguard-vpn", "pihole-dns",
		"adguard-home", "unbound-dns", "technitium-dns", "homeassistant",
		"node-red", "zigbee2mqtt", "mosquitto", "esphome", "frigate-nvr",
		"nextcloud", "immich-photos", "photoprism", "syncthing", "filebrowser",
		"vaultwarden", "authelia", "crowdsec", "fail2ban", "uptime-kuma",
		"healthchecks", "gatus-monitor", "changedetection", "n8n-automation",
		"huginn", "homepage-dash", "homarr-dash", "dashy", "flame-startpage",
		"bookstack-wiki", "outline-docs", "hedgedoc", "trilium-notes",
		"paperless-ngx", "stirling-pdf", "ghost-blog", "wordpress-site",
		"matomo-analytics", "plausible", "umami-stats", "mailcow", "mailu",
		"roundcube", "matrix-synapse", "element-web", "mattermost",
		"rocket-chat", "mumble-voice", "teamspeak", "minecraft-server",
		"valheim-server", "satisfactory", "factorio-server", "terraria",
		"code-server", "gitpod-ws", "jupyter-lab", "rstudio-server",
		"pgadmin", "phpmyadmin", "adminer", "mongo-express", "redis-commander",
		"elasticsearch", "kibana", "logstash", "fluentd", "telegraf",
		"influxdb", "chronograf", "kapacitor", "victoriametrics", "thanos",
		"jaeger-tracing", "zipkin", "tempo-traces", "mimir-metrics",
		"semaphore-ansible", "awx-tower", "rundeck", "salt-master",
		"netbox-dcim", "librenms", "cacti-monitor", "zabbix-server",
	];

	// Spread of statuses: active, partially, unhealthy, exited, down
	const statuses: BadgeStatus[] = ["active", "active", "active", "partially", "unhealthy", "exited", "exited", "down"];
	const stacks: DemoStack[] = stackNames.map((name, i) => ({
		name,
		status: statuses[i % statuses.length],
		services: `${(i % 3) + 1} service${(i % 3) > 0 ? "s" : ""}`,
		updateAvailable: name === "web-app" || name === "grafana-stack" || name === "nextcloud" || name === "vaultwarden",
		recreateNecessary: name === "monitoring" || name === "traefik-lb",
		managed: name !== "portainer" && name !== "pihole-dns" && name !== "homeassistant",
	}));

	let activeItem = $state("test-alpine");
	let searchText = $state("");

	let filters = $state<Record<FilterKey, boolean>>({
		unhealthy: false,
		active: false,
		partially: false,
		exited: false,
		down: false,
		update: false,
		unmanaged: false,
	});

	const filterLabels: Record<FilterKey, () => string> = {
		unhealthy:  m.badgeUnhealthy,
		active:     m.badgeActive,
		partially:  m.badgePartially,
		exited:     m.badgeExited,
		down:       m.badgeDown,
		update:     m.tooltipIconUpdate,
		unmanaged:  () => "unmanaged",
	};

	const filterKeys: FilterKey[] = ["unhealthy", "active", "partially", "exited", "down", "update", "unmanaged"];

	let anyFilterActive = $derived(filterKeys.some((k) => filters[k]));

	function clearFilters() {
		for (const k of filterKeys) filters[k] = false;
	}

	function matchesFilters(stack: DemoStack): boolean {
		if (!anyFilterActive) return true;
		if (filters.unhealthy && stack.status === "unhealthy") return true;
		if (filters.active && stack.status === "active") return true;
		if (filters.partially && stack.status === "partially") return true;
		if (filters.exited && stack.status === "exited") return true;
		if (filters.down && stack.status === "down") return true;
		if (filters.update && stack.updateAvailable) return true;
		if (filters.unmanaged && !stack.managed) return true;
		return false;
	}

	let filteredStacks = $derived(
		stacks.filter((s) => {
			if (searchText && !s.name.toLowerCase().includes(searchText.toLowerCase())) return false;
			return matchesFilters(s);
		})
	);
</script>

<ResourceList bind:searchText filterActive={anyFilterActive} count={filteredStacks.length}>
	{#snippet filterMenu()}
		<div class="p-2 min-w-52">
			<button
				class="mb-2 flex w-full cursor-pointer items-center gap-1 rounded px-1 py-0.5 text-xs transition-colors
					{anyFilterActive
						? 'text-(--color-primary) hover:bg-gray-100 dark:hover:bg-white/10'
						: 'text-gray-400 cursor-default'}"
				disabled={!anyFilterActive}
				onclick={clearFilters}
			>
				&#x2715; clear filter
			</button>
			<div class="flex flex-col gap-1.5">
				{#each filterKeys as key}
					<Checkbox bind:checked={filters[key]} label={filterLabels[key]()} class="px-1 py-0.5 text-sm" />
				{/each}
			</div>
		</div>
	{/snippet}
	{#each filteredStacks as stack}
		<ListItem
			href="/stacks/{stack.name}"
			active={activeItem === stack.name}
			onclick={(e: MouseEvent) => {
				e.preventDefault();
				activeItem = stack.name;
			}}
		>
			<StatusLabel status={stack.status} name={stack.name} size="sm" recreateNecessary={stack.recreateNecessary} updateAvailable={stack.updateAvailable} />
		</ListItem>
	{/each}
</ResourceList>
