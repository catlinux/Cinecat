<script lang="ts">
	import type { Series } from '$lib/api';
	import ListToggle from './ListToggle.svelte';
	import { formatScore } from '$lib/media';
	import { formatSeriesCounts, formatSeriesProgress } from '$lib/series';
	import { posterColors } from '$lib/poster-color';

	let { series }: { series: Series } = $props();

	const colors = $derived(posterColors(series.title));
	const progress = $derived(
		series.episode_count > 0
			? formatSeriesProgress(series.watched_count, series.episode_count)
			: ''
	);
	const ratingText = $derived(formatScore(series.rating ?? null));
</script>

<div class="list-card-wrap" class:has-rating={Boolean(ratingText)}>
<a
	class="series-card"
	href={`/series/${series.id}`}
	style={`--poster-a: ${colors.from}; --poster-b: ${colors.to};`}
>
	<div class="series-frame" class:has-rating={Boolean(ratingText)}>
		{#if series.poster_url}
			<img
				src={series.poster_url}
				alt={`Cartel de la serie ${series.title}`}
				loading="lazy"
			/>
		{:else}
			<div class="series-fallback">
				<span>{series.title}</span>
			</div>
		{/if}
		{#if ratingText}
			<span class="series-rating" aria-label={`Puntuación ${ratingText} de 10`}>
				{ratingText}
			</span>
		{/if}
		{#if progress}
			<span class="series-progress" class:series-progress--complete={progress === 'Vista'}>
				{progress}
			</span>
		{/if}
	</div>
	<div class="series-caption">
		<h2>{series.title}</h2>
		{#if series.year !== null}
			<p class="series-year">{series.year}</p>
		{/if}
		<p>{formatSeriesCounts(series.season_count, series.episode_count)}</p>
	</div>
</a>
{#if series.favorite !== undefined || series.watchlist !== undefined}
	<ListToggle kind="series" id={series.id} list="favorite" active={series.favorite ?? false} size="small" />
{/if}
</div>
