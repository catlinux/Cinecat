<script lang="ts">
	import TrackList from '$lib/components/TrackList.svelte';
	import {
		albumInitials,
		albumQueueTracks,
		formatTotalDuration,
		musicCoverURL
	} from '$lib/music';
	import {
		appendManyToMusicQueue,
		playQueue
	} from '$lib/music-player.svelte';
	import { posterColors } from '$lib/poster-color';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const album = $derived(data.album);
	const colors = $derived(posterColors(album.title));
	const queue = $derived(albumQueueTracks(album));
	const duration = $derived(formatTotalDuration(album.duration_seconds));

	function playAll() {
		if (queue.length > 0) {
			playQueue(queue, 0);
		}
	}

	function playRandom() {
		if (queue.length > 0) {
			playQueue(queue, 0, { shuffle: true });
		}
	}

	function addAll() {
		if (queue.length > 0) {
			appendManyToMusicQueue(queue);
		}
	}
</script>

<svelte:head>
	<title>{album.title} · Música · {data.ui?.server_name ?? 'Cinecat'}</title>
</svelte:head>

<a class="back" href="/music">← Volver a Música</a>

<section class="music-album">
	<header class="music-album-head">
		<div
			class="music-album-cover"
			style={`--album-a: ${colors.from}; --album-b: ${colors.to};`}
		>
			<span class="music-album-initials" aria-hidden="true">{albumInitials(album.title)}</span>
			<img src={musicCoverURL(album.id)} alt={`Carátula de ${album.title}`} />
		</div>
		<div class="music-album-copy">
			<p class="kicker">Álbum</p>
			<h1>{album.title}</h1>
			<p class="music-album-artist">
				<a href={`/music/artists/${album.artist_id}`}>{album.artist}</a>
			</p>
			<p class="music-album-meta">
				{#if album.year}{album.year} · {/if}
				{album.track_count} {album.track_count === 1 ? 'pista' : 'pistas'}
				{#if duration}· {duration}{/if}
			</p>
			<div class="music-album-actions">
				<button type="button" class="button" disabled={queue.length === 0} onclick={playAll}>
					Reproducir
				</button>
				<button
					type="button"
					class="button button-secondary"
					disabled={queue.length === 0}
					onclick={playRandom}
				>
					Aleatorio
				</button>
				<button
					type="button"
					class="button button-secondary"
					disabled={queue.length === 0}
					onclick={addAll}
				>
					Añadir a la cola
				</button>
			</div>
		</div>
	</header>

	<TrackList tracks={album.tracks} {album} />
</section>
