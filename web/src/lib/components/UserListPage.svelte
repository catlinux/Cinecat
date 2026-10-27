<script lang="ts">
	import type { Collection, ListsBlockKind, MediaItem, Series, UserListName } from '$lib/api';
	import { listPageHref, listPageKicker, listPageTitle, listSectionCountLabel } from '$lib/lists';
	import CollectionCard from './CollectionCard.svelte';
	import Pagination from './Pagination.svelte';
	import PosterCard from './PosterCard.svelte';
	import SeriesCard from './SeriesCard.svelte';
	import VideoCard from './VideoCard.svelte';

	let {
		list,
		data
	}: {
		list: UserListName;
		data: {
			kind: ListsBlockKind | null;
			page: number;
			pageSize: number;
			items: MediaItem[];
			series: Series[];
			collections: Collection[];
			totals: { items: number; series: number; collections: number };
			currentUrl: string;
			errorMessage: string | null;
			actionError: boolean;
			ui?: { server_name?: string };
		};
	} = $props();

	const movies = $derived(data.items.filter((item) => item.library_kind !== 'clips'));
	const videos = $derived(data.items.filter((item) => item.library_kind === 'clips'));
	const totalCount = $derived(data.totals.items + data.totals.series + data.totals.collections);
	const hasAny = $derived(totalCount > 0);

	function href(kind: ListsBlockKind, page: number): string {
		return listPageHref(data.currentUrl, kind, page);
	}

	function lastFor(total: number): number {
		return Math.max(1, Math.ceil(total / data.pageSize));
	}

	function pageFor(kind: ListsBlockKind): number {
		return data.kind === kind ? data.page : 1;
	}
</script>

<svelte:head>
	<title>{listPageTitle(list)} · {data.ui?.server_name ?? 'Cinecat'}</title>
</svelte:head>

<section class="library user-lists-page">
	<div class="library-head">
		<p class="kicker">{listPageKicker(list)}</p>
		<h1>{listPageTitle(list)}</h1>
		<p class="library-count">
			{totalCount} {totalCount === 1 ? 'elemento' : 'elementos'}
		</p>
	</div>

	{#if data.errorMessage}
		<p class="error" role="alert">{data.errorMessage}</p>
	{:else if data.actionError}
		<p class="error" role="alert">No se pudo actualizar la lista.</p>
	{/if}

	{#if !data.errorMessage && hasAny}
		{#if movies.length > 0}
			<section class="list-section">
				<h2>Películas <span>{listSectionCountLabel(movies.length, 'película', 'películas')}</span></h2>
				<div class="poster-grid">
					{#each movies as item (item.id)}
						<PosterCard {item} posterUrl={item.poster_url} />
					{/each}
				</div>
				<Pagination
					current={pageFor('items')}
					last={lastFor(data.totals.items)}
					url={href('items', pageFor('items'))}
				/>
			</section>
		{/if}

		{#if videos.length > 0}
			<section class="list-section">
				<h2>Vídeos <span>{listSectionCountLabel(videos.length, 'vídeo', 'vídeos')}</span></h2>
				<div class="video-grid">
					{#each videos as item (item.id)}
						<VideoCard {item} />
					{/each}
				</div>
				<Pagination
					current={pageFor('items')}
					last={lastFor(data.totals.items)}
					url={href('items', pageFor('items'))}
				/>
			</section>
		{/if}

		{#if data.series.length > 0}
			<section class="list-section">
				<h2>Series <span>{listSectionCountLabel(data.totals.series, 'serie', 'series')}</span></h2>
				<div class="series-grid">
					{#each data.series as series (series.id)}
						<SeriesCard {series} />
					{/each}
				</div>
				<Pagination
					current={pageFor('series')}
					last={lastFor(data.totals.series)}
					url={href('series', pageFor('series'))}
				/>
			</section>
		{/if}

		{#if data.collections.length > 0}
			<section class="list-section">
				<h2>Colecciones <span>{listSectionCountLabel(data.totals.collections, 'colección', 'colecciones')}</span></h2>
				<div class="collection-grid">
					{#each data.collections as collection (collection.id)}
						<CollectionCard {collection} />
					{/each}
				</div>
				<Pagination
					current={pageFor('collections')}
					last={lastFor(data.totals.collections)}
					url={href('collections', pageFor('collections'))}
				/>
			</section>
		{/if}
	{:else if !data.errorMessage}
		<div class="empty-state">
			<p class="kicker">Estantería vacía</p>
			<h2>{list === 'favorite' ? 'Todavía no tienes favoritos' : 'No hay nada en próximas a ver'}</h2>
			<p>
				{list === 'favorite'
					? 'Marca películas, series o colecciones con el corazón para tenerlas a mano.'
					: 'Marca títulos para ver más tarde y aparecerán aquí.'}
			</p>
			<p>
				<a class="button button-secondary" href="/movies">Ir a Películas</a>
				<a class="button button-secondary" href="/series">Ir a Series</a>
			</p>
		</div>
	{/if}
</section>
