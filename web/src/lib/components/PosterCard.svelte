<script lang="ts">
	import type { MediaItem } from '$lib/api';
	import ListToggle from './ListToggle.svelte';
	import {
		episodeCardSubtitle,
		episodeImageURL,
		formatScore,
		itemSubtitle,
		kindLabel
	} from '$lib/media';
	import { posterColors } from '$lib/poster-color';
	import {
		episodeProgressSubtitle,
		formatProgressTime,
		progressPercent
	} from '$lib/progress';

	let {
		item,
		posterUrl = null
	}: {
		item: MediaItem;
		posterUrl?: string | null;
	} = $props();

	const colors = $derived(posterColors(item.title));
	const progress = $derived(item.progress);
	const progressValue = $derived(
		progress ? progressPercent(progress.position_seconds, progress.duration_seconds) : 0
	);
	const progressText = $derived(
		progress ? `Continúas por ${formatProgressTime(progress.position_seconds)}` : ''
	);
	const caption = $derived(
		item.kind === 'episode'
			? episodeProgressSubtitle(item) || itemSubtitle(item)
			: itemSubtitle(item)
	);
	const isEpisode = $derived(item.kind === 'episode');
	const artwork = $derived(isEpisode ? episodeImageURL(item) : posterUrl);
	const episodeSubtitle = $derived(isEpisode ? episodeCardSubtitle(item) : '');
	const ratingText = $derived(
		!isEpisode && item.library_kind !== 'clips' ? formatScore(item.rating) : ''
	);
	const collections = $derived(item.collections ?? []);
	const collectionLabel = $derived(
		collections.length === 1
			? `En la colección ${collections[0].name}`
			: `En ${collections.length} colecciones`
	);
</script>

<div class="list-card-wrap" class:has-rating={Boolean(ratingText)}>
{#if isEpisode}
	<a
		class="episode-card"
		href={`/items/${item.id}`}
		style={`--poster-a: ${colors.from}; --poster-b: ${colors.to};`}
	>
		<div class="episode-frame">
			{#if artwork}
				<img src={artwork} alt="" loading="lazy" />
			{:else}
				<div class="episode-fallback">
					<span>{item.title}</span>
				</div>
			{/if}
			<span class="episode-kind">EPISODIO</span>
			{#if item.watched}
				<span class="poster-watched">Visto</span>
			{/if}
			{#if progress && progressValue > 0}
				<div
					class="poster-progress"
					role="progressbar"
					aria-label={`Progreso de ${item.title}`}
					aria-valuemin="0"
					aria-valuemax="100"
					aria-valuenow={progressValue}
				>
					<span style={`width: ${progressValue}%`}></span>
				</div>
			{/if}
		</div>
		<div class="episode-caption">
			<h2>{item.episode_title || item.title}</h2>
			{#if episodeSubtitle}<p>{episodeSubtitle}</p>{/if}
			{#if progressText}
				<p class="poster-continues">{progressText}</p>
			{/if}
		</div>
	</a>
{:else}
	<a
		class="poster-card"
		href={`/items/${item.id}`}
		style={`--poster-a: ${colors.from}; --poster-b: ${colors.to};`}
	>
		<div
			class="poster-frame"
			class:has-rating={Boolean(ratingText)}
			class:has-collections={item.kind === 'movie' && item.library_kind !== 'clips' && collections.length > 0}
		>
			{#if posterUrl}
				<img class="poster-image" src={posterUrl} alt={`Cartel de ${item.title}`} loading="lazy" />
			{:else}
				<div class="poster-fallback">
					<span class="poster-title">{item.title}</span>
				</div>
			{/if}
			<span class="poster-kind">{kindLabel(item.kind)}</span>
			{#if item.kind === 'movie' && item.library_kind !== 'clips' && collections.length > 0}
				<span
					class="poster-collections"
					role="img"
					title={collectionLabel}
					aria-label={collectionLabel}
				>
					<svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
						<rect x="8" y="3" width="12" height="16" rx="1.5" />
						<rect x="4" y="6" width="12" height="16" rx="1.5" />
						<path d="M6.5 10h7M6.5 13h7M6.5 16h4" />
					</svg>
				</span>
			{/if}
			{#if ratingText}
				<span class="poster-rating" aria-label={`Puntuación ${ratingText} de 10`}>
					{ratingText}
				</span>
			{/if}
			{#if item.watched}
				<span class="poster-watched">Visto</span>
			{/if}
			{#if progress && progressValue > 0}
				<div
					class="poster-progress"
					role="progressbar"
					aria-label={`Progreso de ${item.title}`}
					aria-valuemin="0"
					aria-valuemax="100"
					aria-valuenow={progressValue}
				>
					<span style={`width: ${progressValue}%`}></span>
				</div>
			{/if}
		</div>

		<div class="poster-caption">
			<h2>{item.title}</h2>
			<p>{caption}</p>
			{#if progressText}
				<p class="poster-continues">{progressText}</p>
			{/if}
		</div>
	</a>
{/if}
	{#if item.favorite !== undefined || item.watchlist !== undefined}
		<ListToggle kind="item" id={item.id} list="favorite" active={item.favorite ?? false} size="small" />
	{/if}
</div>
