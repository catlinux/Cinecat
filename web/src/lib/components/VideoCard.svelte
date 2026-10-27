<script lang="ts">
	import type { MediaItem } from '$lib/api';
	import { posterColors } from '$lib/poster-color';
	import { formatProgressTime, progressPercent } from '$lib/progress';

	let { item }: { item: MediaItem } = $props();

	const colors = $derived(posterColors(item.title));
	const progress = $derived(item.progress);
	const progressValue = $derived(
		progress ? progressPercent(progress.position_seconds, progress.duration_seconds) : 0
	);
	const progressText = $derived(
		progress ? `Continúas por ${formatProgressTime(progress.position_seconds)}` : ''
	);
</script>

<a
	class="video-card"
	href={`/items/${item.id}`}
	style={`--video-a: ${colors.from}; --video-b: ${colors.to};`}
>
	<div class="video-frame">
		<div class="video-fallback">
			<span class="video-play" aria-hidden="true">▶</span>
			<span class="video-title">{item.title}</span>
		</div>
		<span class="video-kind">Vídeo</span>
		{#if item.watched}
			<span class="video-watched">Visto</span>
		{/if}
		{#if progress && progressValue > 0}
			<div
				class="video-progress"
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
	<div class="video-caption">
		<h3>{item.title}</h3>
		{#if progressText}
			<p>{progressText}</p>
		{/if}
	</div>
</a>

<style>
	.video-card {
		display: flex;
		flex-direction: column;
		gap: 0.55rem;
		min-width: 0;
		color: inherit;
		text-decoration: none;
	}
	.video-frame {
		position: relative;
		overflow: hidden;
		aspect-ratio: 16 / 9;
		border: 1px solid var(--borde-suave);
		border-radius: 4px;
		background:
			linear-gradient(145deg, var(--video-a), var(--video-b)),
			repeating-linear-gradient(90deg, rgb(255 255 255 / 5%) 0 1px, transparent 1px 18px);
		box-shadow: 0 1rem 1.8rem -1.4rem var(--sombra);
		transition:
			transform 160ms ease,
			box-shadow 160ms ease;
	}
	.video-card:hover .video-frame,
	.video-card:focus-visible .video-frame {
		transform: translateY(-3px);
		box-shadow: 0 1.4rem 2rem -1rem var(--sombra);
	}
	.video-fallback {
		position: absolute;
		inset: 0;
		display: flex;
		flex-direction: column;
		justify-content: flex-end;
		gap: 0.35rem;
		padding: 0.9rem;
		color: var(--crema-clara);
	}
	.video-play {
		font-size: 1.25rem;
		color: var(--dorado);
	}
	.video-title {
		display: -webkit-box;
		overflow: hidden;
		font-family: 'Fraunces', Georgia, serif;
		font-size: clamp(1rem, 1.3vw, 1.3rem);
		font-weight: 600;
		line-height: 1.1;
		line-clamp: 2;
		-webkit-box-orient: vertical;
		-webkit-line-clamp: 2;
	}
	.video-kind,
	.video-watched {
		position: absolute;
		top: 0.55rem;
		padding: 0.2rem 0.4rem;
		border-radius: 2px;
		background: rgb(28 13 15 / 72%);
		color: var(--crema-clara);
		font-size: 0.58rem;
		font-weight: 700;
		letter-spacing: 0.11em;
		text-transform: uppercase;
	}
	.video-kind {
		left: 0.55rem;
	}
	.video-watched {
		right: 0.55rem;
		color: var(--dorado);
	}
	.video-progress {
		position: absolute;
		right: 0;
		bottom: 0;
		left: 0;
		height: 4px;
		background: rgb(28 13 15 / 48%);
	}
	.video-progress span {
		display: block;
		height: 100%;
		background: var(--dorado);
	}
	.video-caption h3 {
		margin: 0;
		overflow-wrap: anywhere;
		font-family: 'Fraunces', Georgia, serif;
		font-size: 1rem;
		line-height: 1.2;
		color: var(--granate);
	}
	.video-caption p {
		margin: 0.2rem 0 0;
		color: var(--texto-apagado);
		font-size: 0.74rem;
	}
</style>
