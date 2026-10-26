<script lang="ts">
	import { enhance } from '$app/forms';
	import type { MediaItem } from '$lib/api';
	import { formatRuntime, truncateSynopsis } from '$lib/media';
	import { formatEpisodeHeading } from '$lib/series';
	import { formatProgressTime, progressPercent } from '$lib/progress';
	import { posterColors } from '$lib/poster-color';

	let { item }: { item: MediaItem } = $props();

	const colors = $derived(posterColors(item.title));
	const progress = $derived(item.progress);
	const progressValue = $derived(
		progress ? progressPercent(progress.position_seconds, progress.duration_seconds) : 0
	);
	const progressLabel = $derived(
		progress ? `Continúas por ${formatProgressTime(progress.position_seconds)}` : ''
	);
	const duration = $derived(
		formatRuntime(item.runtime_minutes) ||
			(progress ? formatProgressTime(progress.duration_seconds) : '')
	);
	const shortOverview = $derived(item.overview ? truncateSynopsis(item.overview, 160) : '');
	const hasLongOverview = $derived(
		Boolean(item.overview) && shortOverview !== item.overview?.replace(/\s+/g, ' ').trim()
	);
</script>

<article class="episode-row" class:episode-watched={item.watched}>
	<a class="episode-still" href={`/items/${item.id}`} aria-label={`Ficha de ${item.title}`}>
		{#if item.still_url}
			<img src={item.still_url} alt={`Fotograma de ${item.title}`} loading="lazy" />
		{:else}
			<div
				class="episode-fallback"
				style={`--poster-a: ${colors.from}; --poster-b: ${colors.to};`}
				aria-hidden="true"
			>
				<span>{item.episode !== null ? `E${item.episode}` : 'Episodio'}</span>
			</div>
		{/if}
	</a>

	<div class="episode-body">
		<div class="episode-head">
			<a class="episode-title" href={`/items/${item.id}`}>{formatEpisodeHeading(item)}</a>
			{#if item.watched}
				<span class="episode-watched-tag">Visto</span>
			{/if}
		</div>

		{#if duration}
			<p class="episode-meta">{duration}</p>
		{/if}

		{#if item.overview}
			<p class="episode-overview">{shortOverview}</p>
			{#if hasLongOverview}
				<details class="episode-synopsis">
					<summary>Leer sinopsis completa</summary>
					<p>{item.overview}</p>
				</details>
			{/if}
		{/if}

		{#if progress && progressValue > 0}
			<div
				class="episode-progress"
				role="progressbar"
				aria-label={`Progreso de ${item.title}`}
				aria-valuemin="0"
				aria-valuemax="100"
				aria-valuenow={progressValue}
			>
				<span style={`width: ${progressValue}%`}></span>
			</div>
			{#if progressLabel}
				<p class="episode-continues">{progressLabel}</p>
			{/if}
		{/if}

		<div class="episode-actions">
			<a class="button" href={`/items/${item.id}/play`}>
				{progress ? 'Continuar' : 'Reproducir'}
			</a>
			<form method="POST" action="?/watchedEpisode" use:enhance>
				<input type="hidden" name="itemId" value={item.id} />
				<input type="hidden" name="watched" value={item.watched ? '0' : '1'} />
				<button class="button button-secondary" type="submit" aria-pressed={item.watched}>
					{item.watched ? 'Desmarcar visto' : 'Marcar como visto'}
				</button>
			</form>
		</div>
	</div>
</article>
