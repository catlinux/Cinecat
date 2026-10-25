<script lang="ts">
	import { formatExpiry } from '$lib/media';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const modeLabel = $derived(
		data.play?.mode === 'remux' ? 'Remux HLS' : 'Reproducción directa'
	);
</script>

<svelte:head>
	<title>Reproducir {data.item.title} · Cinecat</title>
</svelte:head>

<a class="back" href={`/items/${data.item.id}`}>← Volver a la ficha</a>

<section class="play-placeholder">
	<p class="kicker">Sesión preparada</p>
	<h1>{data.item.title}</h1>

	{#if data.play}
		<p class="detail-meta">{modeLabel} · Enlace válido hasta {formatExpiry(data.play.expires_at)}</p>
		<p class="detail-note">
			El reproductor se añadirá en el siguiente paso. Mientras tanto, este es el enlace firmado y de
			vida corta que ha devuelto el servidor:
		</p>
		<code class="play-url">{data.playUrl}</code>
		<a class="button" href={data.playUrl ?? '#'} target="_blank" rel="noreferrer">
			Abrir URL firmada
		</a>
	{:else}
		<p class="error" role="alert">{data.playError}</p>
	{/if}
</section>
