<script lang="ts">
	import ThemeToggle from './ThemeToggle.svelte';
	import type { PublicUI } from '$lib/settings';

	let {
		ui,
		activeCategory = null,
		isHome = false,
		isAdmin = false,
		hasSession = false,
		versionLabel
	}: {
		ui: PublicUI;
		activeCategory?: 'movies' | 'collections' | 'series' | null;
		isHome?: boolean;
		isAdmin?: boolean;
		hasSession?: boolean;
		versionLabel: string;
	} = $props();
</script>

<aside class="side-nav">
	<div class="side-brand">
		<a class="side-wordmark" href="/">{ui.server_name}</a>
		{#if ui.tagline}
			<p class="side-tagline">{ui.tagline}</p>
		{/if}
	</div>

	<nav aria-label="Principal">
		<ul class="side-links">
			<li>
				<a href="/" aria-current={isHome ? 'page' : undefined}>Inicio</a>
			</li>
			<li>
				<a
					href="/movies"
					aria-current={activeCategory === 'movies' ? 'page' : undefined}>Películas</a
				>
			</li>
			{#if ui.show_collections}
				<li>
					<a
						href="/collections"
						aria-current={activeCategory === 'collections' ? 'page' : undefined}
						>Colecciones</a
					>
				</li>
			{/if}
			{#if ui.show_series}
				<li>
					<a
						href="/series"
						aria-current={activeCategory === 'series' ? 'page' : undefined}>Series</a
					>
				</li>
			{/if}
		</ul>

		{#if isAdmin || hasSession}
			<div class="side-account">
				{#if isAdmin}
					<a href="/admin">Administración</a>
				{/if}
				{#if hasSession}
					<a href="/account">Mi cuenta</a>
				{/if}
			</div>
		{/if}
	</nav>

	<div class="side-foot">
		<ThemeToggle defaultPreference={ui.theme_default} />
		<p class="side-version">{ui.server_name} {versionLabel} · GPL-3.0</p>
	</div>
</aside>

<style>
	.side-nav {
		position: sticky;
		top: 0;
		align-self: start;
		display: flex;
		flex-direction: column;
		width: 14rem;
		height: 100vh;
		overflow-y: auto;
		padding: 1.5rem 1rem 1.5rem 0;
		border-right: 1px solid var(--borde-suave);
	}
	.side-brand {
		margin-bottom: 1.5rem;
	}
	.side-wordmark {
		display: inline-block;
		overflow-wrap: anywhere;
		font-family: 'Fraunces', Georgia, serif;
		font-size: 1.6rem;
		font-weight: 700;
		letter-spacing: 0.02em;
		color: var(--granate);
		text-decoration: none;
	}
	.side-tagline {
		margin: 0.15rem 0 0;
		overflow-wrap: anywhere;
		color: var(--granate);
		font-size: 0.7rem;
		font-weight: 600;
		letter-spacing: 0.18em;
		line-height: 1.4;
		text-transform: uppercase;
	}
	.side-nav nav {
		flex: 1;
	}
	.side-links {
		display: flex;
		flex-direction: column;
		gap: 0.15rem;
		margin: 0;
		padding: 0;
		list-style: none;
	}
	.side-links a,
	.side-account a {
		display: flex;
		align-items: center;
		min-height: 40px;
		padding: 0.45rem 0.65rem;
		border-left: 3px solid transparent;
		color: var(--texto-tenue);
		font-size: 0.7rem;
		font-weight: 600;
		letter-spacing: 0.12em;
		text-decoration: none;
		text-transform: uppercase;
	}
	.side-links a:hover,
	.side-account a:hover {
		color: var(--granate);
	}
	.side-links a[aria-current='page'] {
		border-left-color: var(--granate);
		background: var(--tinte);
		color: var(--granate);
	}
	.side-account {
		display: flex;
		flex-direction: column;
		gap: 0.15rem;
		margin-top: 1rem;
		padding-top: 1rem;
		border-top: 1px solid var(--borde-suave);
	}
	.side-foot {
		display: flex;
		flex-direction: column;
		gap: 0.7rem;
		margin-top: auto;
		padding-top: 1rem;
		border-top: 1px solid var(--borde-suave);
	}
	.side-version {
		margin: 0;
		overflow-wrap: anywhere;
		color: var(--texto-apagado);
		font-size: 0.68rem;
		line-height: 1.4;
	}

	@media (max-width: 900px) {
		.side-nav {
			position: static;
			width: 100%;
			height: auto;
			overflow: visible;
			padding: 0 0 0.85rem;
			border-right: 0;
			border-bottom: 2px solid var(--granate);
		}
		.side-brand {
			display: flex;
			flex-wrap: wrap;
			align-items: baseline;
			gap: 0.35rem 1rem;
			margin-bottom: 0.6rem;
		}
		.side-links {
			flex-direction: row;
			flex-wrap: wrap;
			gap: 0.15rem 0.55rem;
		}
		.side-links a,
		.side-account a {
			padding: 0.45rem 0.1rem;
			border-left: 0;
			border-bottom: 2px solid transparent;
		}
		.side-links a[aria-current='page'] {
			border-left-color: transparent;
			border-bottom-color: var(--granate);
			background: transparent;
		}
		.side-account {
			flex-direction: row;
			flex-wrap: wrap;
			gap: 0.5rem 1rem;
			margin-top: 0.5rem;
			padding-top: 0.4rem;
		}
		.side-account a {
			border-bottom: 0;
		}
		.side-foot {
			flex-direction: row;
			flex-wrap: wrap;
			align-items: center;
			justify-content: space-between;
			margin-top: 0.7rem;
			padding-top: 0.6rem;
		}
	}
</style>
