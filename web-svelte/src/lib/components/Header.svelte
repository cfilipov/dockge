<script lang="ts">
	import Icon from "./Icon.svelte";
	import {
		faLayerGroup,
		faCubes,
		faNetworkWired,
		faBoxArchive,
		faHardDrive,
		faTerminal,
		faAngleDown,
	} from "@fortawesome/free-solid-svg-icons";
	import { m } from "$lib/i18n/messages";

	const navItems = [
		{ label: m.stacks, icon: faLayerGroup, href: "/stacks" },
		{ label: m.containersNav, icon: faCubes, href: "/containers" },
		{ label: m.networksNav, icon: faNetworkWired, href: "/networks" },
		{ label: m.imagesNav, icon: faBoxArchive, href: "/images" },
		{ label: m.volumesNav, icon: faHardDrive, href: "/volumes" },
		{ label: m.console, icon: faTerminal, href: "/console" },
	];

	let activeHref = $state("/stacks");
</script>

<header class="header">
	<a href="/" class="logo">
		<img src="/icon.svg" alt="Dockge" class="logo-img" />
		<span class="logo-text">Dockge</span>
	</a>

	<nav class="nav">
		{#each navItems as item}
			<a
				href={item.href}
				class="nav-pill {activeHref === item.href ? 'active' : ''}"
				onclick={(e: MouseEvent) => {
					e.preventDefault();
					activeHref = item.href;
				}}
			>
				<Icon icon={item.icon} />
				{item.label}
			</a>
		{/each}
	</nav>

	<div class="profile">
		<button class="profile-btn" aria-label="User menu">
			<span class="profile-pic">A</span>
			<Icon icon={faAngleDown} />
		</button>
	</div>
</header>

<style>
	.header {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		padding: 1rem 0;
		margin-bottom: 1rem;
		background-color: white;
		border-bottom: 1px solid #e5e7eb;
	}

	:global(.dark) .header {
		background-color: var(--color-header-dark);
		border-bottom-color: var(--color-header-dark);
	}

	.logo {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		margin-left: 1.5rem;
		margin-right: auto;
		text-decoration: none;
	}

	.logo-img {
		width: 2.5rem;
		height: 2.5rem;
	}

	.logo-text {
		font-size: 1.5rem;
		font-weight: bold;
		color: #111;
	}

	:global(.dark) .logo-text {
		color: #f0f6fc;
	}

	/* Nav pills — default: centered second row */
	.nav {
		order: 1;
		width: 100%;
		display: flex;
		flex-wrap: wrap;
		justify-content: center;
		gap: 0.25rem;
		padding: 0.5rem 1.5rem 0;
		margin: 0;
	}

	/* Wide screens: single row */
	@media (min-width: 1250px) {
		.nav {
			order: 0;
			width: auto;
			flex: 1 1 auto;
			flex-wrap: nowrap;
			padding: 0;
		}

		.profile {
			order: 1;
		}
	}

	.nav-pill {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 0.5rem 1rem;
		border-radius: 50rem;
		font-size: 1rem;
		text-decoration: none;
		color: #6b7280;
		transition: none;
	}

	.nav-pill:hover {
		background-color: #f3f4f6;
	}

	:global(.dark) .nav-pill {
		color: var(--color-font-dark);
	}

	:global(.dark) .nav-pill:hover {
		background-color: var(--color-body-dark-deep);
	}

	.nav-pill.active {
		color: #020b05;
		background: linear-gradient(135deg, #74c2ff 0%, #74c2ff 75%, #86e6a9);
	}

	:global(.dark) .nav-pill.active {
		color: #020b05;
	}

	.profile {
		margin-right: 1.5rem;
		flex-shrink: 0;
	}

	.profile-btn {
		display: flex;
		align-items: center;
		gap: 6px;
		background-color: rgba(200, 200, 200, 0.2);
		border: none;
		padding: 0.5rem 0.8rem;
		border-radius: 50rem;
		cursor: pointer;
		color: inherit;
	}

	.profile-btn:hover {
		background-color: rgba(255, 255, 255, 0.2);
	}

	.profile-pic {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 24px;
		height: 24px;
		margin-right: 5px;
		border-radius: 50rem;
		background-color: var(--color-primary);
		color: white;
		font-weight: bold;
		font-size: 10px;
	}
</style>
