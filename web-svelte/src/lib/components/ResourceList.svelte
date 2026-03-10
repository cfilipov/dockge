<script lang="ts">
	import IconButton from "./ui/IconButton.svelte";
	import TextInput from "./ui/TextInput.svelte";
	import ListItem from "./ui/ListItem.svelte";
	import StatusLabel from "./ui/StatusLabel.svelte";
	import type { BadgeStatus } from "./ui/Badge.svelte";
	import * as m from "$lib/paraglide/messages";
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
		updateAvailable: name === "web-app",
		recreateNecessary: name === "monitoring",
	}));

	let activeStack = $state("test-alpine");
	let searchText = $state("");
</script>

<div class="shadow-box dark:bg-(--color-body-dark) dark:shadow-[0_15px_70px_rgba(0,0,0,0.1)] flex min-h-0 flex-1 flex-col overflow-hidden">
	<!-- Header with search + filter -->
	<div class="flex items-center border-b border-gray-200 rounded-t-[10px] p-[10px] dark:border-transparent dark:bg-(--color-header-dark)">
		<IconButton
			icon={searchText ? faXmark : faMagnifyingGlass}
			aria-label={searchText ? m.clearSearch() : m.search()}
			size="md"
			onclick={() => { if (searchText) searchText = ""; }}
		/>
		<TextInput
			bind:value={searchText}
			placeholder={m.search()}
			autocomplete="off"
			class="max-w-60 flex-1"
		/>
		<IconButton
			icon={faFilter}
			aria-label={m.filter()}
			size="md"
		/>
	</div>

	<!-- Stack list -->
	<div class="flex-1 overflow-y-auto py-[10px] pl-[10px] mr-[10px] mt-[10px] mb-[10px]">
		<div class="pr-[6px]">
		{#each stacks as stack}
			<ListItem
				href="/stacks/{stack.name}"
				active={activeStack === stack.name}
				onclick={(e: MouseEvent) => {
					e.preventDefault();
					activeStack = stack.name;
				}}
			>
				<StatusLabel status={stack.status} name={stack.name} size="sm" recreateNecessary={stack.recreateNecessary} updateAvailable={stack.updateAvailable} />
			</ListItem>
		{/each}
		</div>
	</div>
</div>
