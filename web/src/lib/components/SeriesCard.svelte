<script lang="ts">
	import type { Series } from '$lib/api';
	import { formatSeriesCounts, formatSeriesProgress } from '$lib/series';
	import { posterColors } from '$lib/poster-color';

	let { series }: { series: Series } = $props();

	const colors = $derived(posterColors(series.title));
	const progress = $derived(
		series.episode_count > 0
			? formatSeriesProgress(series.watched_count, series.episode_count)
			: ''
	);
</script>

<a
	class="series-card"
	href={`/series/${series.id}`}
	style={`--poster-a: ${colors.from}; --poster-b: ${colors.to};`}
>
	<div class="series-frame">
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
