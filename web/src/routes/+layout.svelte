<script lang="ts">
	import '../app.css';
	import type { Snippet } from 'svelte';
	import type { LayoutData } from './$types';
	import SideNav from '$lib/components/SideNav.svelte';
	import ThemeToggle from '$lib/components/ThemeToggle.svelte';
	import { apiBaseUrl } from '$lib/config';
	import { DEFAULT_PUBLIC_UI } from '$lib/settings';
	import { versionMismatch } from '$lib/version';

	let { data, children }: { data: LayoutData; children: Snippet } = $props();

	const ui = $derived({ ...DEFAULT_PUBLIC_UI, ...(data.ui ?? {}) });
	const sideNav = $derived(ui.nav_layout === 'side');
	const versionLabel = $derived(
		`${data.serverVersion ?? data.webVersion}${data.serverCommit ? ` (${data.serverCommit})` : ''}`
	);

	$effect(() => {
		if (typeof document !== 'undefined') {
			document.documentElement.dataset.accent = ui.accent;
		}
	});
</script>

<svelte:head>
	<title>{ui.server_name}</title>
</svelte:head>

<div class="shell" class:shell-side={sideNav}>
	{#if sideNav}
		<SideNav
			{ui}
			activeCategory={data.activeCategory}
			isHome={data.isHome ?? false}
			isAdmin={data.isAdmin}
			hasSession={data.hasSession}
			{versionLabel}
		/>
	{/if}

	<div class="shell-content">
		{#if !sideNav}
			<header class="marquee">
				<div class="marquee-brand">
					{#if ui.tagline}
						<p class="eyebrow">{ui.tagline}</p>
					{/if}
					<a class="wordmark" href="/">{ui.server_name}</a>
				</div>
				<div class="marquee-tools">
					<nav class="category-nav" aria-label="Principal">
						<a
							href="/"
							class:category-active={data.isHome}
							aria-current={data.isHome ? 'page' : undefined}>Inicio</a
						>
						<a
							href="/movies"
							class:category-active={data.activeCategory === 'movies'}
							aria-current={data.activeCategory === 'movies' ? 'page' : undefined}>Películas</a
						>
						{#if ui.show_collections}
							<a
								href="/collections"
								class:category-active={data.activeCategory === 'collections'}
								aria-current={data.activeCategory === 'collections' ? 'page' : undefined}
								>Colecciones</a
							>
						{/if}
						{#if ui.show_series}
							<a
								href="/series"
								class:category-active={data.activeCategory === 'series'}
								aria-current={data.activeCategory === 'series' ? 'page' : undefined}>Series</a
							>
						{/if}
					</nav>
					{#if data.isAdmin}
						<a class="account-link" href="/admin">Administración</a>
					{/if}
					{#if data.hasSession}
						<a class="account-link" href="/account">Mi cuenta</a>
					{/if}
					<ThemeToggle defaultPreference={ui.theme_default} />
				</div>
			</header>
		{/if}

		<main>
			{@render children()}
		</main>

		<footer>
			<span>{ui.server_name} {versionLabel} · GPL-3.0</span>
			<span class="server">Servidor: {apiBaseUrl || 'sin configurar'}</span>
		</footer>
		{#if versionMismatch(data.serverVersion, data.webVersion)}
			<p class="version-warning" role="status">
				La web y el servidor tienen versiones distintas: recarga o actualiza.
			</p>
		{/if}
	</div>
</div>

<style>
	.shell-side {
		display: grid;
		grid-template-columns: 14rem minmax(0, 1fr);
		align-items: start;
		gap: clamp(1rem, 3vw, 2.5rem);
		max-width: 84rem;
		padding-top: 0;
		padding-bottom: 0;
	}
	.shell-content {
		display: flex;
		flex-direction: column;
		min-width: 0;
		min-height: 100vh;
	}
	.shell-side > .shell-content {
		padding: 1.5rem 0 2rem;
	}
	.version-warning {
		margin: 0.5rem 0 0;
		font-size: 0.78rem;
		line-height: 1.5;
		color: var(--naranja);
	}

	@media (max-width: 900px) {
		.shell-side {
			display: block;
			max-width: 72rem;
			padding: 1.25rem clamp(1rem, 4vw, 3rem) 2rem;
		}
		.shell-side > .shell-content {
			min-height: 0;
			padding: 0;
		}
	}
</style>
