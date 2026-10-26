<script lang="ts">
	import type { MediaItem } from '$lib/api';
	import { itemSubtitle, kindLabel } from '$lib/media';
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
		progress ? episodeProgressSubtitle(item) || itemSubtitle(item) : itemSubtitle(item)
	);
</script>

<a
	class="poster-card"
	href={`/items/${item.id}`}
	style={`--poster-a: ${colors.from}; --poster-b: ${colors.to};`}
>
	<div class="poster-frame">
		{#if posterUrl}
			<img class="poster-image" src={posterUrl} alt={`Cartel de ${item.title}`} loading="lazy" />
		{:else}
			<div class="poster-fallback">
				<span class="poster-title">{item.title}</span>
			</div>
		{/if}
		<span class="poster-kind">{kindLabel(item.kind)}</span>
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
