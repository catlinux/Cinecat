<script lang="ts">
	import type { MusicAlbum, MusicTrack } from '$lib/api';
	import {
		formatTrackDuration,
		trackPlayableReason,
		toQueueTrack
	} from '$lib/music';
	import {
		musicPlayer,
		appendToMusicQueue,
		playNextInMusicQueue,
		playTrack
	} from '$lib/music-player.svelte';

	let {
		tracks,
		album = null,
		showAlbum = false,
		currentTrackId = undefined
	}: {
		tracks: MusicTrack[];
		album?: MusicAlbum | null;
		showAlbum?: boolean;
		currentTrackId?: number | null;
	} = $props();

	const playable = $derived(tracks.filter((track) => track.playable));
	const queue = $derived(
		playable.map((track) => toQueueTrack(track, album))
	);
	const currentID = $derived(
		currentTrackId !== undefined
			? currentTrackId
			: musicPlayer.index >= 0
				? musicPlayer.tracks[musicPlayer.index]?.id ?? null
				: null
	);
	const discs = $derived(
		[...new Set(tracks.map((track) => track.disc_no || 1))].sort((left, right) => left - right)
	);
	const showDiscs = $derived(Boolean(album) && discs.length > 1);

	function queueTrack(track: MusicTrack) {
		return toQueueTrack(track, album);
	}

	function play(track: MusicTrack) {
		if (!track.playable) {
			return;
		}
		playTrack(queueTrack(track), queue);
	}

	function playNext(track: MusicTrack) {
		if (!track.playable) {
			return;
		}
		playNextInMusicQueue(queueTrack(track));
	}

	function addToQueue(track: MusicTrack) {
		if (!track.playable) {
			return;
		}
		appendToMusicQueue(queueTrack(track));
	}

	function trackNumber(track: MusicTrack, index: number): string {
		return track.track_no ? String(track.track_no) : String(index + 1);
	}

	function isCurrent(track: MusicTrack): boolean {
		return currentID === track.id;
	}
</script>

<ol class="track-list">
	{#each tracks as track, index (track.id)}
		{#if showDiscs && (index === 0 || (tracks[index - 1]?.disc_no || 1) !== (track.disc_no || 1))}
			<li class="track-disc">Disco {track.disc_no || 1}</li>
		{/if}
		<li
			class="track-row"
			class:track-row--current={isCurrent(track)}
			class:track-row--disabled={!track.playable}
			aria-current={isCurrent(track) ? 'true' : undefined}
		>
			<span class="track-number" aria-hidden="true">{trackNumber(track, index)}</span>
			<div class="track-copy">
				<strong>{track.title}</strong>
				{#if showAlbum}
					<span class="track-context">
						{track.artist}{#if track.album} · {track.album}{/if}
					</span>
				{:else if album}
					<span class="track-context">{track.artist || album.artist}</span>
				{/if}
				{#if isCurrent(track)}
					<span class="track-current">Sonando ahora</span>
				{/if}
				{#if !track.playable}
					<span class="track-unplayable">{trackPlayableReason(track)}</span>
				{/if}
			</div>
			<span class="track-duration">{formatTrackDuration(track.duration_seconds)}</span>
			<div class="track-actions">
				<button
					type="button"
					class="track-action track-action--play"
					disabled={!track.playable}
					aria-label={`Reproducir ${track.title}`}
					title={track.playable ? 'Reproducir' : trackPlayableReason(track)}
					onclick={() => play(track)}
				>
					▶
				</button>
				<button
					type="button"
					class="track-action"
					disabled={!track.playable}
					aria-label={`Reproducir a continuación ${track.title}`}
					title={track.playable ? 'Reproducir a continuación' : trackPlayableReason(track)}
					onclick={() => playNext(track)}
				>
					A continuación
				</button>
				<button
					type="button"
					class="track-action"
					disabled={!track.playable}
					aria-label={`Añadir ${track.title} a la cola`}
					title={track.playable ? 'Añadir a la cola' : trackPlayableReason(track)}
					onclick={() => addToQueue(track)}
				>
					+ Cola
				</button>
			</div>
		</li>
	{/each}
</ol>

{#if tracks.length === 0}
	<p class="track-empty">No hay pistas que mostrar.</p>
{/if}

<style>
	.track-list {
		display: grid;
		gap: 0.15rem;
		margin: 1.2rem 0 0;
		padding: 0;
		list-style: none;
	}
	.track-row {
		display: grid;
		grid-template-columns: 2rem minmax(0, 1fr) auto auto;
		gap: 0.7rem;
		align-items: center;
		min-height: 3rem;
		padding: 0.45rem 0.35rem;
		border-bottom: 1px dashed var(--borde-suave);
	}
	.track-row--current {
		border-left: 3px solid var(--naranja);
		background: var(--tinte);
	}
	.track-row--disabled {
		opacity: 0.62;
	}
	.track-disc {
		margin-top: 1rem;
		padding: 0.2rem 0;
		color: var(--naranja);
		font-size: 0.68rem;
		font-weight: 700;
		letter-spacing: 0.12em;
		text-transform: uppercase;
	}
	.track-number {
		color: var(--texto-apagado);
		font-size: 0.75rem;
		text-align: right;
	}
	.track-copy {
		display: grid;
		gap: 0.1rem;
		min-width: 0;
	}
	.track-copy strong {
		overflow-wrap: anywhere;
		font-family: 'Fraunces', Georgia, serif;
		font-size: 0.98rem;
		font-weight: 600;
		line-height: 1.2;
		color: var(--granate);
	}
	.track-context {
		overflow-wrap: anywhere;
		color: var(--texto-apagado);
		font-size: 0.72rem;
		line-height: 1.35;
	}
	.track-current {
		color: var(--naranja);
		font-size: 0.62rem;
		font-weight: 700;
		letter-spacing: 0.1em;
		text-transform: uppercase;
	}
	.track-unplayable {
		color: var(--naranja);
		font-size: 0.68rem;
	}
	.track-duration {
		color: var(--texto-apagado);
		font-size: 0.76rem;
		font-variant-numeric: tabular-nums;
	}
	.track-actions {
		display: flex;
		flex-wrap: wrap;
		justify-content: flex-end;
		gap: 0.3rem;
	}
	.track-action {
		min-height: 40px;
		padding: 0.35rem 0.55rem;
		border: 1px solid var(--borde-suave);
		border-radius: 3px;
		background: var(--superficie);
		color: var(--texto-tenue);
		font-family: Inter, system-ui, sans-serif;
		font-size: 0.62rem;
		font-weight: 600;
		letter-spacing: 0.04em;
		cursor: pointer;
	}
	.track-action:hover:not(:disabled) {
		border-color: var(--dorado);
		color: var(--granate);
	}
	.track-action:focus-visible {
		outline: 2px solid var(--naranja);
		outline-offset: 2px;
	}
	.track-action:disabled {
		cursor: not-allowed;
		opacity: 0.5;
	}
	.track-action--play {
		min-width: 40px;
		color: var(--naranja);
	}
	.track-empty {
		margin: 1rem 0 0;
		color: var(--texto-apagado);
	}
	@media (max-width: 760px) {
		.track-row {
			grid-template-columns: 1.5rem minmax(0, 1fr) auto;
		}
		.track-duration {
			display: none;
		}
		.track-actions {
			grid-column: 2 / -1;
			justify-content: flex-start;
		}
	}
</style>
