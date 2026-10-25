<script lang="ts">
	import '../app.css';
	import { apiBaseUrl } from '$lib/config';

	let { children } = $props();
</script>

<svelte:head>
	<title>Cinecat</title>
</svelte:head>

<div class="shell">
	<header class="marquee">
		<p class="eyebrow">Cine de barrio · Biblioteca personal</p>
		<a class="wordmark" href="/">Cinecat</a>
	</header>

	<main>
		{@render children()}
	</main>

	<footer>
		<span>Cinecat · GPL-3.0</span>
		<span class="server">Servidor: {apiBaseUrl || 'sin configurar'}</span>
	</footer>
</div>
