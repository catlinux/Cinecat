<script lang="ts">
	import CategoryArt from '$lib/components/CategoryArt.svelte';
	import PosterCard from '$lib/components/PosterCard.svelte';
	import PosterRow from '$lib/components/PosterRow.svelte';
	import RowScroller from '$lib/components/RowScroller.svelte';
	import SeriesCard from '$lib/components/SeriesCard.svelte';
	import SeriesRow from '$lib/components/SeriesRow.svelte';
	import VideoCard from '$lib/components/VideoCard.svelte';
	import { DEFAULT_PORTAL_ROWS, DEFAULT_PUBLIC_UI } from '$lib/settings';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const continueEpisodes = $derived(data.continueWatching.filter((item) => item.kind === 'episode'));
	const continueMovies = $derived(data.continueWatching.filter((item) => item.kind !== 'episode'));
	const ui = $derived({ ...DEFAULT_PUBLIC_UI, ...(data.ui ?? {}) });
	const portalRows = $derived(
		ui.portal_rows?.length ? ui.portal_rows : DEFAULT_PORTAL_ROWS
	);
	const recentlyWatched = $derived(data.recentlyWatched ?? []);

	type PortalCategory = {
		id: string;
		title: string;
		description: string;
		kicker: string;
		href: string | null;
		art: 'movies' | 'collections' | 'series' | 'videos' | 'music';
		count: number | null;
	};

	// Para activar una categoría, dale href; para añadir Imágenes o Vídeos
	// musicales, añade otra entrada a este mismo array.
	const categories = $derived<PortalCategory[]>([
		...(data.hasVideo !== false
			? [
					{
						id: 'movies',
						title: 'Películas',
						description: 'La cartelera completa, con fichas, filtros y búsqueda por reparto o dirección.',
						kicker: 'En cartel',
						href: '/movies',
						art: 'movies' as const,
						count: data.errorMessage ? null : data.moviesTotal
					},
					...(ui.show_collections
						? [
								{
									id: 'collections',
									title: 'Colecciones',
									description: 'Sagas y ciclos reunidos en un mismo estante.',
									kicker: 'Sagas',
									href: '/collections',
									art: 'collections' as const,
									count: data.collectionsTotal
								}
							]
						: []),
					...(ui.show_series
						? [
								{
									id: 'series',
									title: 'Series',
									description: 'Temporadas, episodios y ese capítulo que siempre vuelve.',
									kicker: 'Series',
									href: '/series',
									art: 'series' as const,
									count: data.seriesTotal
								}
							]
						: [])
				]
			: []),
		...(data.hasClips
			? [
					{
						id: 'videos',
						title: 'Vídeos',
						description: 'Videoclips y conciertos, ordenados por carpetas.',
						kicker: 'Videoclips',
						href: '/videos',
						art: 'videos' as const,
						count: null
					}
				]
			: []),
		...(data.hasMusic
			? [
					{
						id: 'music',
						title: 'Música',
						description: 'Álbumes, artistas y la banda sonora de la casa.',
						kicker: 'Discoteca',
						href: '/music',
						art: 'music' as const,
						count: null
					}
				]
			: [])
	]);

	const movieCount = $derived(
		data.errorMessage
			? 'Abrir cartelera'
			: `${data.moviesTotal} ${data.moviesTotal === 1 ? 'película' : 'películas'}`
	);
	const collectionCount = $derived(
		data.collectionsTotal === null
			? 'Entrar'
			: `${data.collectionsTotal} ${data.collectionsTotal === 1 ? 'colección' : 'colecciones'}`
	);
	const seriesCount = $derived(
		data.seriesTotal == null
			? 'Entrar'
			: `${data.seriesTotal} ${data.seriesTotal === 1 ? 'serie' : 'series'}`
	);

	const watchlistEntries = $derived(
		(data.watchlist ?? []).filter((entry) =>
			entry.kind === 'series' ? ui.show_series && Boolean(entry.series) : Boolean(entry.item)
		)
	);
	const watchlistMovieEntries = $derived(
		watchlistEntries.filter((entry) => entry.kind === 'item')
	);
	const watchlistSeriesEntries = $derived(
		watchlistEntries.filter((entry) => entry.kind === 'series')
	);
	const favoriteEntries = $derived(
		(data.favorites ?? []).filter((entry) =>
			entry.kind === 'series' ? ui.show_series && Boolean(entry.series) : Boolean(entry.item)
		)
	);

	function categoryMeta(category: PortalCategory): string {
		if (category.id === 'movies') {
			return movieCount;
		}
		if (category.id === 'collections') {
			return collectionCount;
		}
		if (category.id === 'series') {
			return seriesCount;
		}
		return 'Entrar';
	}
</script>

<svelte:head>
	<title>{ui.portal_title} · {ui.server_name}</title>
</svelte:head>

<section class="portal">
	<div class="portal-hero">
		<p class="kicker">Bienvenido a la sala</p>
		<h1>{ui.portal_title}</h1>
		{#if ui.portal_intro}
			<p class="portal-lede">{ui.portal_intro}</p>
		{/if}
	</div>

	{#if data.errorMessage}
		<p class="error" role="alert">{data.errorMessage}</p>
	{/if}

	{#if data.hasAnyLibraryAccess === false}
		<div class="admin-card portal-empty" role="status">
			<h2>Todavía no tienes ninguna biblioteca asignada</h2>
			<p class="admin-note">Pídele acceso al administrador.</p>
		</div>
	{:else}
	<div class="portal-categories">
		{#each categories as category (category.id)}
			{#if category.href}
				<a class="portal-category" href={category.href}>
					<CategoryArt art={category.art} />
					<div class="portal-category-copy">
						<p class="kicker">{category.kicker}</p>
						<h2>{category.title}</h2>
						<p>{category.description}</p>
						<span class="portal-category-meta">
							{categoryMeta(category)}
							<span aria-hidden="true">→</span>
						</span>
					</div>
				</a>
			{:else}
				<div class="portal-category portal-category--soon" aria-disabled="true">
					<CategoryArt art={category.art} />
					<div class="portal-category-copy">
						<p class="kicker">{category.kicker}</p>
						<h2>{category.title}</h2>
						<p>{category.description}</p>
						<span class="soon-badge">Próximamente</span>
					</div>
				</div>
			{/if}
		{/each}
	</div>

	{#if !data.errorMessage}
		<div class="portal-rows">
			{#each portalRows as row (row.id)}
				{#if row.visible}
					{#if row.id === 'recent_movies' && data.recent.length > 0}
						<PosterRow
							title={row.title}
							kicker={row.kicker}
							items={data.recent}
							href="/movies?sort=added"
						/>
					{:else if row.id === 'continue_movies' && continueMovies.length > 0}
						<PosterRow title={row.title} kicker={row.kicker} items={continueMovies} />
					{:else if row.id === 'watchlist_movies' && watchlistMovieEntries.length > 0}
						<RowScroller title={row.title} kicker={row.kicker} href="/watchlist">
							{#snippet children()}
								{#each watchlistMovieEntries as entry, index (`watch-movie-${entry.item?.id ?? index}`)}
									<li class:row-wide={entry.item?.library_kind === 'clips'}>
										{#if entry.item}
											{#if entry.item.library_kind === 'clips'}
												<VideoCard item={entry.item} />
											{:else}
												<PosterCard item={entry.item} posterUrl={entry.item.poster_url} />
											{/if}
										{/if}
									</li>
								{/each}
							{/snippet}
						</RowScroller>
					{:else if row.id === 'watched_movies' && recentlyWatched.length > 0}
						<PosterRow title={row.title} kicker={row.kicker} items={recentlyWatched} />
					{:else if row.id === 'recent_series' && ui.show_series && data.recentSeries.length > 0}
						<SeriesRow
							title={row.title}
							kicker={row.kicker}
							items={data.recentSeries}
							href="/series?sort=added"
						/>
					{:else if row.id === 'continue_series' && continueEpisodes.length > 0}
						<PosterRow title={row.title} kicker={row.kicker} items={continueEpisodes} />
					{:else if row.id === 'next_up' && data.nextUp.length > 0}
						<PosterRow title={row.title} kicker={row.kicker} items={data.nextUp} />
					{:else if row.id === 'watchlist_series' && watchlistSeriesEntries.length > 0}
						<RowScroller title={row.title} kicker={row.kicker} href="/watchlist">
							{#snippet children()}
								{#each watchlistSeriesEntries as entry, index (`watch-series-${entry.series?.id ?? index}`)}
									<li>
										{#if entry.series}
											<SeriesCard series={entry.series} />
										{/if}
									</li>
								{/each}
							{/snippet}
						</RowScroller>
					{:else if row.id === 'favorites' && favoriteEntries.length > 0}
						<RowScroller title={row.title} kicker={row.kicker} href="/favorites">
							{#snippet children()}
								{#each favoriteEntries as entry, index (`favorite-${entry.kind}-${entry.item?.id ?? entry.series?.id ?? index}`)}
									<li class:row-wide={entry.kind === 'item' && entry.item?.library_kind === 'clips'}>
										{#if entry.kind === 'item' && entry.item}
											{#if entry.item.library_kind === 'clips'}
												<VideoCard item={entry.item} />
											{:else}
												<PosterCard item={entry.item} posterUrl={entry.item.poster_url} />
											{/if}
										{:else if entry.kind === 'series' && entry.series}
											<SeriesCard series={entry.series} />
										{/if}
									</li>
								{/each}
							{/snippet}
						</RowScroller>
					{/if}
				{/if}
			{/each}
		</div>
	{/if}
	{/if}

</section>

<style>
	.portal-empty {
		margin-top: 1.5rem;
	}
	.portal-empty h2 {
		margin-bottom: 0.5rem;
	}
</style>
