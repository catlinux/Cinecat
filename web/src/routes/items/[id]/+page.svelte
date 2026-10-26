<script lang="ts">
	import { itemSubtitle, kindLabel } from '$lib/media';
	import { posterColors } from '$lib/poster-color';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const item = $derived(data.item);
	const colors = $derived(posterColors(item.title));
</script>

<svelte:head>
	<title>{item.title} · Cinecat</title>
</svelte:head>

<a class="back" href="/">← Volver a la cartelera</a>

<section class="detail">
	<div
		class="detail-poster"
		class:detail-poster--image={Boolean(item.poster_url)}
		style={`--poster-a: ${colors.from}; --poster-b: ${colors.to};`}
		aria-hidden="true"
	>
		<span class="poster-kind">{kindLabel(item.kind)}</span>
		{#if item.poster_url}
			<img
				class="detail-poster-img"
				src={item.poster_url}
				alt={`Cartel de ${item.title}`}
				loading="lazy"
			/>
		{:else}
			<span class="detail-poster-title">{item.title}</span>
		{/if}
	</div>

	<div class="detail-info">
		<p class="kicker">{kindLabel(item.kind)}</p>
		<h1>{item.title}</h1>
		<p class="detail-meta">{itemSubtitle(item)}</p>

		{#if item.kind === 'episode' && item.season !== null && item.episode !== null}
			<p class="detail-extra">Temporada {item.season}, episodio {item.episode}</p>
		{/if}

		{#if item.overview}
			<p class="detail-overview">{item.overview}</p>
		{/if}

		<a class="button" href={`/items/${item.id}/play`}>Reproducir</a>
	</div>
</section>
