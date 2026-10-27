<script lang="ts">
	import AlbumCard from '$lib/components/AlbumCard.svelte';
	import { playQueue } from '$lib/music-player.svelte';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const artist = $derived(data.artist);
	const queue = $derived(data.queueTracks ?? []);

	function playAll() {
		if (queue.length > 0) {
			playQueue(queue, 0);
		}
	}
</script>

<svelte:head>
	<title>{artist.name} · Música · {data.ui?.server_name ?? 'Cinecat'}</title>
</svelte:head>

<a class="back" href="/music?vista=artistas">← Volver a artistas</a>

<section class="music-artist">
	<header class="music-artist-head">
		<p class="kicker">Artista</p>
		<h1>{artist.name}</h1>
		<p class="music-artist-meta">
			{artist.album_count} {artist.album_count === 1 ? 'álbum' : 'álbumes'} ·
			{artist.track_count} {artist.track_count === 1 ? 'pista' : 'pistas'}
		</p>
		<button type="button" class="button" disabled={queue.length === 0} onclick={playAll}>
			Reproducir todo
		</button>
	</header>

	{#if artist.albums.length > 0}
		<div class="album-grid">
			{#each artist.albums as album (album.id)}
				<AlbumCard {album} />
			{/each}
		</div>
	{:else}
		<div class="empty-state">
			<p class="kicker">Sin discos</p>
			<h2>Este artista no tiene álbumes</h2>
		</div>
	{/if}
</section>
