<script lang="ts">
	import AlbumCard from '$lib/components/AlbumCard.svelte';
	import Pagination from '$lib/components/Pagination.svelte';
	import TrackList from '$lib/components/TrackList.svelte';
	import {
		MUSIC_ORDERS,
		MUSIC_ORDER_LABELS,
		MUSIC_VIEWS,
		MUSIC_VIEW_LABELS,
		musicCountLabel,
		musicLetterHref,
		musicSearchPlaceholder,
		musicTabHref
	} from '$lib/music';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const letters = ['#', ...'ABCDEFGHIJKLMNOPQRSTUVWXYZ'];
</script>

<svelte:head>
	<title>Música · {data.ui?.server_name ?? 'Cinecat'}</title>
</svelte:head>

<section class="library music-library">
	<div class="library-head">
		<p class="kicker">Discoteca</p>
		<h1>Música</h1>
		{#if data.errorMessage}
			<p class="library-count">La colección no está disponible ahora mismo</p>
		{:else}
			<p class="library-count">{musicCountLabel(data.view, data.total)}</p>
		{/if}
	</div>

	<nav class="music-tabs" aria-label="Vistas de música">
		{#each MUSIC_VIEWS as view (view)}
			<a
				href={musicTabHref(data.currentUrl, view)}
				class:music-tab-active={data.view === view}
				aria-current={data.view === view ? 'page' : undefined}
			>
				{MUSIC_VIEW_LABELS[view]}
			</a>
		{/each}
	</nav>

	{#if data.errorMessage}
		<p class="error" role="alert">{data.errorMessage}</p>
	{:else}
		<form class="music-toolbar" method="GET" action="/music">
			{#if data.view !== 'albumes'}
				<input type="hidden" name="vista" value={data.view} />
			{/if}
			{#if data.letter}
				<input type="hidden" name="letra" value={data.letter} />
			{/if}
			<label class="music-search">
				<span>Buscar</span>
				<input
					type="search"
					name="q"
					value={data.q}
					placeholder={musicSearchPlaceholder(data.view)}
					autocomplete="off"
				/>
			</label>
			{#if data.view === 'albumes'}
				<label class="music-sort">
					<span>Orden</span>
					<select name="orden">
						{#each MUSIC_ORDERS as order (order)}
							<option value={order} selected={data.order === order}>
								{MUSIC_ORDER_LABELS[order]}
							</option>
						{/each}
					</select>
				</label>
			{/if}
			<button type="submit" class="button music-toolbar-submit">Aplicar</button>
		</form>

		{#if data.view !== 'pistas'}
			<nav class="music-letter-strip" aria-label="Filtrar por letra">
				<span class="music-filter-label">Letra</span>
				{#each letters as letter (letter)}
					<a
						href={musicLetterHref(data.currentUrl, letter, data.letter === letter)}
						class:music-letter-active={data.letter === letter}
						aria-current={data.letter === letter ? 'page' : undefined}>{letter}</a
					>
				{/each}
			</nav>
		{/if}

		{#if data.view === 'albumes'}
			{#if data.albums.length > 0}
				<div class="album-grid">
					{#each data.albums as album (album.id)}
						<AlbumCard {album} />
					{/each}
				</div>
			{:else}
				<div class="empty-state">
					<p class="kicker">Sin discos</p>
					<h2>No hay álbumes que coincidan</h2>
					<p>Prueba con otra búsqueda o quita el filtro de letra.</p>
				</div>
			{/if}
		{:else if data.view === 'artistas'}
			{#if data.artists.length > 0}
				<div class="artist-grid">
					{#each data.artists as artist (artist.id)}
						<a class="artist-card" href={`/music/artists/${artist.id}`}>
							<span class="artist-card-kicker">Artista</span>
							<strong>{artist.name}</strong>
							<span>
								{artist.album_count} {artist.album_count === 1 ? 'álbum' : 'álbumes'} ·
								{artist.track_count} {artist.track_count === 1 ? 'pista' : 'pistas'}
							</span>
						</a>
					{/each}
				</div>
			{:else}
				<div class="empty-state">
					<p class="kicker">Sin artistas</p>
					<h2>No hay artistas que coincidan</h2>
					<p>Prueba con otro nombre o quita el filtro de letra.</p>
				</div>
			{/if}
		{:else if !data.q}
			<div class="empty-state">
				<p class="kicker">Buscar en la discoteca</p>
				<h2>Escribe un título, artista o álbum</h2>
				<p>Los resultados aparecerán aquí con sus acciones de reproducción.</p>
			</div>
		{:else if data.tracks.length > 0}
			<TrackList tracks={data.tracks} showAlbum />
		{:else}
			<div class="empty-state">
				<p class="kicker">Sin pistas</p>
				<h2>Ninguna pista coincide</h2>
				<p>Prueba con otras palabras o busca por el nombre del álbum.</p>
			</div>
		{/if}

		<Pagination current={data.page} last={data.last} url={data.currentUrl} />
	{/if}
</section>
