<script lang="ts">
	import type { MusicAlbum } from '$lib/api';
	import { albumInitials, albumMeta, musicCoverURL } from '$lib/music';
	import { posterColors } from '$lib/poster-color';

	let { album }: { album: MusicAlbum } = $props();

	const colors = $derived(posterColors(album.title));
	const initials = $derived(albumInitials(album.title));
</script>

<a
	class="album-card"
	href={`/music/albums/${album.id}`}
	style={`--album-a: ${colors.from}; --album-b: ${colors.to};`}
>
	<div class="album-frame">
		<div class="album-fallback">
			<span aria-hidden="true">{initials}</span>
		</div>
		<img
			class="album-cover"
			src={musicCoverURL(album.id)}
			alt=""
			loading="lazy"
			decoding="async"
		/>
		{#if album.year}
			<span class="album-year">{album.year}</span>
		{/if}
	</div>
	<div class="album-caption">
		<h2>{album.title}</h2>
		<p class="album-artist">{album.artist}</p>
		<p class="album-meta">{albumMeta(album)}</p>
	</div>
</a>

<style>
	.album-card {
		display: flex;
		flex-direction: column;
		gap: 0.65rem;
		min-width: 0;
		color: inherit;
		text-decoration: none;
	}
	.album-frame {
		position: relative;
		overflow: hidden;
		aspect-ratio: 1;
		border: 1px solid var(--borde-suave);
		border-radius: 4px;
		background:
			linear-gradient(145deg, var(--album-a), var(--album-b)),
			repeating-linear-gradient(90deg, rgb(255 255 255 / 6%) 0 1px, transparent 1px 15px);
		box-shadow: 0 1rem 1.8rem -1rem var(--sombra);
		transition:
			transform 160ms ease,
			box-shadow 160ms ease;
	}
	.album-card:hover .album-frame,
	.album-card:focus-visible .album-frame {
		transform: translateY(-3px) rotate(-0.3deg);
		box-shadow: 0 1.4rem 2rem -1rem var(--sombra);
	}
	.album-fallback {
		position: absolute;
		inset: 0;
		display: grid;
		place-items: center;
		background: linear-gradient(180deg, transparent 20%, rgb(20 10 11 / 42%));
		color: var(--crema-clara);
		font-family: 'Fraunces', Georgia, serif;
		font-size: clamp(1.6rem, 4vw, 2.4rem);
		font-weight: 600;
		letter-spacing: 0.04em;
		text-shadow: 0 2px 12px rgb(0 0 0 / 35%);
	}
	.album-cover {
		position: absolute;
		inset: 0;
		width: 100%;
		height: 100%;
		object-fit: cover;
	}
	.album-year {
		position: absolute;
		top: 0.55rem;
		right: 0.55rem;
		z-index: 1;
		padding: 0.18rem 0.42rem;
		border-radius: 2px;
		background: rgb(28 13 15 / 72%);
		color: var(--crema-clara);
		font-size: 0.62rem;
		font-weight: 700;
		letter-spacing: 0.06em;
	}
	.album-caption h2 {
		margin: 0;
		overflow-wrap: anywhere;
		font-family: 'Fraunces', Georgia, serif;
		font-size: 1.02rem;
		font-weight: 600;
		line-height: 1.15;
		color: var(--granate);
	}
	.album-caption p {
		margin: 0.18rem 0 0;
		overflow-wrap: anywhere;
		font-size: 0.76rem;
		line-height: 1.35;
		color: var(--texto-apagado);
	}
	.album-caption .album-artist {
		color: var(--texto-tenue);
	}
	.album-meta {
		font-size: 0.68rem !important;
	}
</style>
