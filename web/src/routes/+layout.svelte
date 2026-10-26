<script lang="ts">
	import '../app.css';
	import type { Snippet } from 'svelte';
	import type { LayoutData } from './$types';
	import ThemeToggle from '$lib/components/ThemeToggle.svelte';
	import { apiBaseUrl } from '$lib/config';
	import { versionMismatch } from '$lib/version';

	let { data, children }: { data: LayoutData; children: Snippet } = $props();

	const versionLabel = $derived(
		`${data.serverVersion ?? data.webVersion}${data.serverCommit ? ` (${data.serverCommit})` : ''}`
	);
</script>

<svelte:head>
	<title>Cinecat</title>
</svelte:head>

<div class="shell">
	<header class="marquee">
		<div class="marquee-brand">
			<p class="eyebrow">Cine de barrio · Biblioteca personal</p>
			<a class="wordmark" href="/">Cinecat</a>
		</div>
		<div class="marquee-tools">
			<nav class="category-nav" aria-label="Categorías">
				<a
					href="/movies"
					class:category-active={data.activeCategory === 'movies'}
					aria-current={data.activeCategory === 'movies' ? 'page' : undefined}>Películas</a
				>
				<a
					href="/collections"
					class:category-active={data.activeCategory === 'collections'}
					aria-current={data.activeCategory === 'collections' ? 'page' : undefined}>Colecciones</a
				>
				<span class="category-soon" aria-disabled="true">
					Series <span class="category-soon-tag">Próximamente</span>
				</span>
			</nav>
			{#if data.isAdmin}
				<a class="account-link" href="/admin/libraries">Bibliotecas</a>
			{/if}
			{#if data.hasSession}
				<a class="account-link" href="/account">Mi cuenta</a>
			{/if}
			<ThemeToggle />
		</div>
	</header>

	<main>
		{@render children()}
	</main>

	<footer>
		<span>Cinecat {versionLabel} · GPL-3.0</span>
		<span class="server">Servidor: {apiBaseUrl || 'sin configurar'}</span>
	</footer>
	{#if versionMismatch(data.serverVersion, data.webVersion)}
		<p class="version-warning" role="status">
			La web y el servidor tienen versiones distintas: recarga o actualiza.
		</p>
	{/if}
</div>

<style>
	.version-warning {
		margin: 0.5rem 0 0;
		font-size: 0.78rem;
		line-height: 1.5;
		color: var(--naranja);
	}
</style>
