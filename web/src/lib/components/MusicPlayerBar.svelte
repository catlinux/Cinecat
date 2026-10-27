<script lang="ts">
	import { onMount } from 'svelte';
	import { formatTrackDuration, musicCoverURL } from '$lib/music';
	import {
		currentMusicTrack,
		cycleMusicRepeat,
		musicPlayer,
		nextMusicTrack,
		pauseMusic,
		pauseMusicForVideo,
		previousMusicTrack,
		removeFromMusicQueue,
		restoreMusicPlayer,
		seekMusic,
		setMusicAudioElement,
		setMusicVolume,
		toggleMusicMute,
		toggleMusicPlay,
		toggleMusicShuffle,
		clearMusicQueue
	} from '$lib/music-player.svelte';

	let audioElement: HTMLAudioElement | null = $state(null);
	const current = $derived(currentMusicTrack());
	const progressMax = $derived(musicPlayer.duration > 0 ? musicPlayer.duration : 0);
	const progressText = $derived(
		`${formatTrackDuration(musicPlayer.currentTime)} de ${formatTrackDuration(progressMax)}`
	);
	const repeatLabel = $derived(
		musicPlayer.repeat === 'one'
			? 'Repetir una'
			: musicPlayer.repeat === 'all'
				? 'Repetir todas'
				: 'Repetir'
	);
	const albumHref = $derived(current && current.albumId > 0 ? `/music/albums/${current.albumId}` : null);
	const artistHref = $derived(
		current
			? current.artistId
				? `/music/artists/${current.artistId}`
				: `/music?q=${encodeURIComponent(current.artist)}&vista=artistas`
			: null
	);

	$effect(() => {
		if (typeof navigator === 'undefined' || !('mediaSession' in navigator)) {
			return;
		}
		const track = current;
		if (!track) {
			navigator.mediaSession.metadata = null;
			return;
		}
		if (typeof MediaMetadata === 'undefined') {
			return;
		}
		navigator.mediaSession.metadata = new MediaMetadata({
			title: track.title,
			artist: track.artist,
			album: track.album,
			artwork: track.albumId > 0
				? [{ src: musicCoverURL(track.albumId), sizes: '512x512', type: 'image/jpeg' }]
				: []
		});
	});

	onMount(() => {
		if (audioElement) {
			setMusicAudioElement(audioElement);
		}
		restoreMusicPlayer();
		installMediaSession();
		const onVideoPlay = (event: Event) => {
			pauseMusicForVideo(event.target);
		};
		document.addEventListener('play', onVideoPlay, true);
		return () => {
			document.removeEventListener('play', onVideoPlay, true);
			setMusicAudioElement(null);
		};
	});

	function installMediaSession() {
		if (typeof navigator === 'undefined' || !('mediaSession' in navigator)) {
			return;
		}
		const actions: Array<[MediaSessionAction, MediaSessionActionHandler]> = [
			['play', () => toggleMusicPlay()],
			['pause', () => pauseMusic()],
			['previoustrack', () => previousMusicTrack()],
			['nexttrack', () => nextMusicTrack()],
			['seekto', (details) => {
				if (details.seekTime !== undefined) {
					seekMusic(details.seekTime);
				}
			}]
		];
		for (const [action, handler] of actions) {
			try {
				navigator.mediaSession.setActionHandler(action, handler);
			} catch {
				// Algunos navegadores no admiten todas las acciones.
			}
		}
	}

	function updateProgress(event: Event) {
		const input = event.currentTarget as HTMLInputElement;
		seekMusic(Number(input.value));
	}

	function updateVolume(event: Event) {
		const input = event.currentTarget as HTMLInputElement;
		setMusicVolume(Number(input.value));
	}
</script>

<audio bind:this={audioElement} preload="metadata"></audio>
<span class="music-sr" aria-live="polite">
	{current ? `Pista: ${current.title}` : ''}
</span>

{#if musicPlayer.tracks.length > 0 && current}
	<aside class="music-player-bar" aria-label="Reproductor de música">
		<div class="music-player-main">
			<a
				class="music-player-cover"
				href={albumHref ?? '/music'}
				aria-label={current.album ? `Ir al álbum ${current.album}` : 'Ir a Música'}
			>
				<span aria-hidden="true">{current.album ? current.album.slice(0, 1) : '♪'}</span>
				{#if current.albumId > 0}
					<img src={musicCoverURL(current.albumId)} alt="" loading="lazy" />
				{/if}
			</a>
			<div class="music-player-copy">
				<strong>{current.title}</strong>
				<span>
					{#if artistHref}
						<a href={artistHref}>{current.artist || 'Artista desconocido'}</a>
					{:else}
						{current.artist || 'Artista desconocido'}
					{/if}
					{#if current.album}
						· <a href={albumHref ?? '/music'}>{current.album}</a>
					{/if}
				</span>
			</div>
			<div class="music-player-controls" role="group" aria-label="Controles de música">
				<button type="button" class="music-control" aria-label="Pista anterior" onclick={() => previousMusicTrack()}>
					⏮
				</button>
				<button
					type="button"
					class="music-control music-control--main"
					aria-label={musicPlayer.playing ? 'Pausar' : 'Reproducir'}
					onclick={() => toggleMusicPlay()}
				>
					{musicPlayer.playing ? '⏸' : '▶'}
				</button>
				<button type="button" class="music-control" aria-label="Pista siguiente" onclick={() => nextMusicTrack()}>
					⏭
				</button>
			</div>
			<div class="music-player-progress">
				<span>{formatTrackDuration(musicPlayer.currentTime)}</span>
				<!-- svelte-ignore a11y_no_redundant_roles -->
				<input
					type="range"
					min="0"
					max={progressMax}
					step="0.1"
					value={musicPlayer.currentTime}
					role="slider"
					aria-label="Progreso de la pista"
					aria-valuemin="0"
					aria-valuemax={progressMax}
					aria-valuenow={musicPlayer.currentTime}
					aria-valuetext={progressText}
					oninput={updateProgress}
				/>
				<span>{formatTrackDuration(progressMax)}</span>
			</div>
			<div class="music-player-extra">
				<button
					type="button"
					class="music-control"
					aria-label={musicPlayer.muted ? 'Activar sonido' : 'Silenciar'}
					aria-pressed={musicPlayer.muted}
					onclick={() => toggleMusicMute()}
				>
					{musicPlayer.muted ? '🔇' : '🔊'}
				</button>
				<label class="music-volume">
					<span class="music-sr">Volumen</span>
					<input
						type="range"
						min="0"
						max="1"
						step="0.01"
						value={musicPlayer.volume}
						aria-label="Volumen"
						oninput={updateVolume}
					/>
				</label>
				<button
					type="button"
					class="music-control"
					aria-label="Aleatorio"
					aria-pressed={musicPlayer.shuffle}
					onclick={() => toggleMusicShuffle()}
				>
					⇄
				</button>
				<button
					type="button"
					class="music-control"
					aria-label={repeatLabel}
					aria-pressed={musicPlayer.repeat !== 'off'}
					onclick={() => cycleMusicRepeat()}
				>
					{musicPlayer.repeat === 'one' ? '1' : '↻'}
				</button>
			</div>
			<details class="music-queue">
				<summary>Cola ({musicPlayer.tracks.length})</summary>
				<div class="music-queue-panel">
					<ol>
						{#each musicPlayer.tracks as track, index (index)}
							<li class:music-queue-current={index === musicPlayer.index}>
								<span>{track.title}</span>
								<small>{track.artist}</small>
								<button
									type="button"
									class="music-control"
									aria-label={`Quitar ${track.title} de la cola`}
									onclick={() => removeFromMusicQueue(index)}
								>
									×
								</button>
							</li>
						{/each}
					</ol>
					<button type="button" class="music-queue-clear" onclick={() => clearMusicQueue()}>
						Vaciar cola
					</button>
				</div>
			</details>
		</div>
		{#if musicPlayer.notice}
			<p class="music-notice" role="status">{musicPlayer.notice}</p>
		{/if}
	</aside>
{/if}

<style>
	.music-sr {
		position: absolute;
		width: 1px;
		height: 1px;
		padding: 0;
		overflow: hidden;
		clip: rect(0 0 0 0);
		white-space: nowrap;
		border: 0;
	}
	.music-player-bar {
		position: fixed;
		z-index: 30;
		right: 0;
		bottom: 0;
		left: 0;
		border-top: 1px solid var(--dorado);
		background: color-mix(in srgb, var(--superficie) 96%, transparent);
		box-shadow: 0 -1rem 2rem -1.2rem var(--sombra);
	}
	.music-player-main {
		display: grid;
		grid-template-columns: 3.25rem minmax(10rem, 1fr) auto minmax(12rem, 1.2fr) auto auto;
		gap: 0.65rem;
		align-items: center;
		width: min(100%, 92rem);
		margin: 0 auto;
		padding: 0.5rem clamp(0.7rem, 2vw, 1.5rem);
	}
	.music-player-cover {
		position: relative;
		display: grid;
		place-items: center;
		width: 3.25rem;
		height: 3.25rem;
		overflow: hidden;
		border: 1px solid var(--borde-suave);
		border-radius: 3px;
		background: var(--tinte);
		color: var(--granate);
		font-family: 'Fraunces', Georgia, serif;
		font-size: 1.2rem;
		text-decoration: none;
	}
	.music-player-cover img {
		position: absolute;
		inset: 0;
		width: 100%;
		height: 100%;
		object-fit: cover;
	}
	.music-player-copy {
		display: grid;
		gap: 0.1rem;
		min-width: 0;
	}
	.music-player-copy strong {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		font-family: 'Fraunces', Georgia, serif;
		font-size: 0.95rem;
		color: var(--granate);
	}
	.music-player-copy span {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		font-size: 0.72rem;
		color: var(--texto-apagado);
	}
	.music-player-copy a {
		color: var(--naranja);
		text-decoration: none;
	}
	.music-player-copy a:hover {
		text-decoration: underline;
	}
	.music-player-controls,
	.music-player-extra {
		display: flex;
		align-items: center;
		gap: 0.2rem;
	}
	.music-control {
		display: inline-grid;
		place-items: center;
		min-width: 40px;
		min-height: 40px;
		padding: 0.25rem;
		border: 1px solid transparent;
		border-radius: 3px;
		background: transparent;
		color: var(--texto-tenue);
		font: inherit;
		font-size: 0.9rem;
		cursor: pointer;
	}
	.music-control:hover {
		border-color: var(--borde-suave);
		color: var(--granate);
	}
	.music-control[aria-pressed='true'] {
		color: var(--naranja);
	}
	.music-control:focus-visible,
	.music-queue summary:focus-visible,
	.music-queue-clear:focus-visible {
		outline: 2px solid var(--naranja);
		outline-offset: 2px;
	}
	.music-control--main {
		border-color: var(--dorado);
		background: var(--tinte);
		color: var(--granate);
	}
	.music-player-progress {
		display: grid;
		grid-template-columns: auto minmax(4rem, 1fr) auto;
		gap: 0.4rem;
		align-items: center;
		color: var(--texto-apagado);
		font-size: 0.66rem;
		font-variant-numeric: tabular-nums;
	}
	.music-player-progress input {
		width: 100%;
		accent-color: var(--naranja);
	}
	.music-volume {
		display: flex;
		align-items: center;
	}
	.music-volume input {
		width: 4.5rem;
		accent-color: var(--naranja);
	}
	.music-queue {
		position: relative;
	}
	.music-queue summary {
		display: grid;
		place-items: center;
		min-height: 40px;
		padding: 0.25rem 0.5rem;
		border: 1px solid transparent;
		border-radius: 3px;
		color: var(--texto-tenue);
		font-size: 0.66rem;
		font-weight: 600;
		cursor: pointer;
		list-style: none;
		white-space: nowrap;
	}
	.music-queue summary::-webkit-details-marker {
		display: none;
	}
	.music-queue[open] summary {
		border-color: var(--borde-suave);
		color: var(--granate);
	}
	.music-queue-panel {
		position: absolute;
		right: 0;
		bottom: calc(100% + 0.5rem);
		width: min(20rem, calc(100vw - 1rem));
		max-height: min(55vh, 22rem);
		overflow: auto;
		padding: 0.65rem;
		border: 1px solid var(--borde-medio);
		border-radius: 4px;
		background: var(--superficie);
		box-shadow: 0 1rem 2.5rem -1rem var(--sombra);
	}
	.music-queue-panel ol {
		display: grid;
		gap: 0.1rem;
		margin: 0;
		padding: 0;
		list-style: none;
	}
	.music-queue-panel li {
		display: grid;
		grid-template-columns: minmax(0, 1fr) auto;
		gap: 0.15rem 0.4rem;
		align-items: center;
		padding: 0.35rem 0.2rem;
		border-bottom: 1px dashed var(--borde-suave);
	}
	.music-queue-panel li span {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		font-size: 0.78rem;
		color: var(--granate);
	}
	.music-queue-panel li small {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		color: var(--texto-apagado);
	}
	.music-queue-panel li .music-control {
		grid-row: 1 / 3;
		grid-column: 2;
		min-width: 40px;
		min-height: 40px;
	}
	.music-queue-current {
		background: var(--tinte);
	}
	.music-queue-clear {
		min-height: 40px;
		margin-top: 0.6rem;
		padding: 0.4rem 0.55rem;
		border: 1px solid var(--borde-suave);
		border-radius: 3px;
		background: transparent;
		color: var(--naranja);
		font: inherit;
		font-size: 0.68rem;
		cursor: pointer;
	}
	.music-notice {
		margin: 0;
		padding: 0.3rem clamp(0.7rem, 2vw, 1.5rem);
		background: var(--fondo-error);
		color: var(--texto-error);
		font-size: 0.74rem;
		text-align: center;
	}
	@media (max-width: 1000px) {
		.music-player-main {
			grid-template-columns: 2.8rem minmax(8rem, 1fr) auto auto auto;
		}
		.music-player-progress {
			grid-column: 1 / -1;
		}
		.music-player-cover {
			width: 2.8rem;
			height: 2.8rem;
		}
	}
	@media (max-width: 640px) {
		.music-player-main {
			grid-template-columns: 2.6rem minmax(0, 1fr) auto;
			gap: 0.35rem;
			padding: 0.45rem 0.55rem;
		}
		.music-player-controls {
			flex-wrap: nowrap;
		}
		.music-player-extra {
			grid-column: 1 / -1;
			justify-content: center;
		}
		.music-queue {
			grid-column: 1 / -1;
			justify-self: end;
		}
		.music-volume {
			display: none;
		}
		.music-player-copy strong {
			font-size: 0.85rem;
		}
		.music-player-copy span {
			font-size: 0.65rem;
		}
		.music-player-cover {
			width: 2.6rem;
			height: 2.6rem;
		}
	}
</style>
