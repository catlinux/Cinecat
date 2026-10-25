<script lang="ts">
	import '../app.css';
	import type { Snippet } from 'svelte';
	import type { LayoutData } from './$types';
	import ThemeToggle from '$lib/components/ThemeToggle.svelte';
	import { apiBaseUrl } from '$lib/config';

	let { data, children }: { data: LayoutData; children: Snippet } = $props();
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
		<span>Cinecat · GPL-3.0</span>
		<span class="server">Servidor: {apiBaseUrl || 'sin configurar'}</span>
	</footer>
</div>
