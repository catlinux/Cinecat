<script lang="ts">
	import ThemeToggle from './ThemeToggle.svelte';
	import type { PublicUI } from '$lib/settings';

	let {
		ui,
		activeCategory = null,
		isHome = false,
		isAdmin = false,
		hasSession = false,
		hasVideo = true,
		hasClips = false,
		hasMusic = false,
		versionLabel
	}: {
		ui: PublicUI;
		activeCategory?:
			| 'movies'
			| 'collections'
			| 'series'
			| 'videos'
			| 'music'
			| 'favorites'
			| 'watchlist'
			| null;
		isHome?: boolean;
		isAdmin?: boolean;
		hasSession?: boolean;
		hasVideo?: boolean;
		hasClips?: boolean;
		hasMusic?: boolean;
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
			{#if hasVideo}
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
			{/if}
			{#if hasClips}
				<li>
					<a
						href="/videos"
						aria-current={activeCategory === 'videos' ? 'page' : undefined}>Vídeos</a
					>
				</li>
			{/if}
			{#if hasMusic}
				<li>
					<a
						href="/music"
						aria-current={activeCategory === 'music' ? 'page' : undefined}>Música</a
					>
				</li>
			{/if}
		</ul>

		{#if hasSession}
			<div class="side-lists">
				<a
					href="/favorites"
					aria-current={activeCategory === 'favorites' ? 'page' : undefined}>Favoritos</a
				>
				<a
					href="/watchlist"
					aria-current={activeCategory === 'watchlist' ? 'page' : undefined}
					>Próximas a ver</a
				>
			</div>
		{/if}

		{#if isAdmin || hasSession}
			<div class="side-account">
				{#if isAdmin}
					<a href="/admin">Administración</a>
				{/if}
				{#if hasSession}
					<a href="/account">Mi cuenta</a>
					<form method="POST" action="/login?/logout" class="side-logout-form">
						<button class="side-link-button" type="submit">Cerrar sesión</button>
					</form>
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
		width: 10.5rem;
		height: 100vh;
		overflow-y: auto;
		padding: 1.15rem 0.65rem 1.15rem clamp(0.75rem, 1.5vw, 1.1rem);
		border-right: 1px solid var(--borde-suave);
	}
	.side-brand {
		margin-bottom: 1rem;
	}
	.side-wordmark {
		display: inline-block;
		overflow-wrap: anywhere;
		font-family: 'Fraunces', Georgia, serif;
		font-size: 1.2rem;
		font-weight: 700;
		letter-spacing: 0.02em;
		color: var(--granate);
		text-decoration: none;
	}
	.side-tagline {
		margin: 0.15rem 0 0;
		overflow-wrap: anywhere;
		color: var(--granate);
		font-size: 0.58rem;
		font-weight: 600;
		letter-spacing: 0.14em;
		line-height: 1.35;
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
	.side-account a,
	.side-account button {
		display: flex;
		align-items: center;
		min-height: 40px;
		padding: 0.35rem 0.4rem;
		border-left: 3px solid transparent;
		color: var(--texto-tenue);
		font-size: 0.62rem;
		font-weight: 600;
		letter-spacing: 0.08em;
		text-decoration: none;
		text-transform: uppercase;
	}
	.side-links a:hover,
	.side-account a:hover,
	.side-account button:hover {
		color: var(--granate);
	}
	.side-links a[aria-current='page'] {
		border-left-color: var(--granate);
		background: var(--tinte);
		color: var(--granate);
	}
	.side-lists {
		display: flex;
		flex-direction: column;
		gap: 0.1rem;
		margin-top: 0.65rem;
		padding-top: 0.65rem;
		border-top: 1px solid var(--borde-suave);
	}
	.side-lists a {
		display: flex;
		align-items: center;
		min-height: 40px;
		padding: 0.35rem 0.4rem;
		border-left: 3px solid transparent;
		color: var(--texto-tenue);
		font-size: 0.62rem;
		font-weight: 600;
		letter-spacing: 0.08em;
		text-decoration: none;
		text-transform: uppercase;
	}
	.side-lists a:hover {
		color: var(--granate);
	}
	.side-lists a[aria-current='page'] {
		border-left-color: var(--naranja);
		background: var(--tinte);
		color: var(--naranja);
	}
	.side-account {
		display: flex;
		flex-direction: column;
		gap: 0.1rem;
		margin-top: 0.75rem;
		padding-top: 0.75rem;
		border-top: 1px solid var(--borde-suave);
	}
	.side-logout-form {
		display: flex;
		margin: 0;
	}
	.side-account .side-link-button {
		width: 100%;
		border: 0;
		border-left: 3px solid transparent;
		background: transparent;
		text-align: left;
		cursor: pointer;
	}
	.side-foot {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		margin-top: auto;
		padding-top: 0.75rem;
		border-top: 1px solid var(--borde-suave);
	}
	.side-foot :global(.theme-toggle) {
		justify-content: center;
	}
	.side-foot :global(.theme-toggle button) {
		padding: 0.3rem 0.35rem;
		font-size: 0.58rem;
		letter-spacing: 0.04em;
	}
	.side-version {
		margin: 0;
		overflow-wrap: anywhere;
		color: var(--texto-apagado);
		font-size: 0.6rem;
		line-height: 1.35;
	}

	@media (max-width: 900px) {
		.side-nav {
			position: static;
			width: 100%;
			height: auto;
			overflow: visible;
			padding: 1rem clamp(1rem, 4vw, 3rem) 0.85rem;
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
		.side-wordmark {
			font-size: 1.35rem;
		}
		.side-tagline {
			font-size: 0.66rem;
			letter-spacing: 0.16em;
		}
		.side-links {
			flex-direction: row;
			flex-wrap: wrap;
			gap: 0.15rem 0.55rem;
		}
		.side-links a,
		.side-account a,
		.side-account button {
			padding: 0.45rem 0.1rem;
			border-left: 0;
			border-bottom: 2px solid transparent;
			font-size: 0.68rem;
			letter-spacing: 0.12em;
		}
		.side-foot :global(.theme-toggle button) {
			padding: 0.35rem 0.65rem;
			font-size: 0.66rem;
			letter-spacing: 0.08em;
		}
		.side-links a[aria-current='page'] {
			border-left-color: transparent;
			border-bottom-color: var(--granate);
			background: transparent;
		}
		.side-lists {
			flex-direction: row;
			flex-wrap: wrap;
			gap: 0.15rem 0.55rem;
			margin-top: 0.5rem;
			padding-top: 0.4rem;
		}
		.side-lists a {
			padding: 0.45rem 0.1rem;
			border-left: 0;
			border-bottom: 2px solid transparent;
			font-size: 0.68rem;
			letter-spacing: 0.12em;
		}
		.side-lists a[aria-current='page'] {
			border-left-color: transparent;
			border-bottom-color: var(--naranja);
			background: transparent;
		}
		.side-account {
			flex-direction: row;
			flex-wrap: wrap;
			gap: 0.5rem 1rem;
			margin-top: 0.5rem;
			padding-top: 0.4rem;
		}
		.side-account a,
		.side-account button {
			border-bottom: 0;
		}
		.side-account .side-logout-form {
			width: auto;
		}
		.side-account .side-link-button {
			width: auto;
			border-left: 0;
			border-bottom: 2px solid transparent;
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
