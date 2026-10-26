<script lang="ts">
	import Player from '$lib/components/Player.svelte';
	import { formatExpiry } from '$lib/media';
	import { formatProgressTime } from '$lib/progress';
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

<section class="play-page">
	<p class="kicker">{modeLabel}</p>
	<h1>{data.item.title}</h1>

	{#if data.play && data.playUrl}
		<p class="detail-meta">Enlace válido hasta {formatExpiry(data.play.expires_at)}</p>
		{#if data.startAt > 0 && data.item.progress}
			<p class="continue-note">
				Continúas por {formatProgressTime(data.startAt)}.
				<a href={`/items/${data.item.id}/play?t=0`}>Empezar desde el principio</a>
			</p>
		{/if}
		<div class="player-stage">
			<Player
				src={data.playUrl}
				mode={data.play.mode}
				title={data.item.title}
				itemId={data.item.id}
				startAt={data.startAt}
				options={data.play}
				nextEpisodeUrl={data.nextEpisodeUrl}
			/>
		</div>
	{:else}
		<p class="error" role="alert">{data.playError}</p>
	{/if}
</section>
