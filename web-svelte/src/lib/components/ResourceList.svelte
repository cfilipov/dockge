<script lang="ts">
	import Icon from "./Icon.svelte";
	import Badge from "./ui/Badge.svelte";
	import type { BadgeStatus } from "./ui/Badge.svelte";
	import { faMagnifyingGlass, faXmark, faFilter } from "@fortawesome/free-solid-svg-icons";

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
	const statuses: BadgeStatus[] = ["running", "running", "running", "exited", "down"];
	const stacks = stackNames.map((name, i) => ({
		name,
		status: statuses[i % statuses.length],
		services: `${(i % 3) + 1} service${(i % 3) > 0 ? "s" : ""}`,
	}));

	let activeStack = $state("test-alpine");
	let searchText = $state("");
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
				<Badge status={stack.status} />
				<span class="ml-2 truncate">{stack.name}</span>
			</a>
		{/each}
		</div>
	</div>
</div>
