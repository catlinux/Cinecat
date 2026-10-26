<script lang="ts">
	import LibraryFilters from '$lib/components/LibraryFilters.svelte';
	import Pagination from '$lib/components/Pagination.svelte';
	import SeriesCard from '$lib/components/SeriesCard.svelte';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const hasFilters = $derived(data.filterChips.length > 0);
</script>

<svelte:head>
	<title>Series · Cinecat</title>
</svelte:head>

<section class="library">
	<div class="library-head">
		<p class="kicker">Series</p>
		<h1>Series</h1>
		<p class="library-count">
			{data.total} {data.total === 1 ? 'serie' : 'series'}
		</p>
	</div>

	<LibraryFilters
		filters={data.filters}
		genres={data.genres}
		isAdmin={data.isAdmin}
		unmatchedCount={data.unmatchedCount}
		currentUrl={data.currentUrl}
		action="/series"
		showField={false}
		showWatched={false}
		searchPlaceholder="Título, reparto, creadores…"
	/>

	{#if data.filterChips.length > 0}
		<div class="filter-chips" aria-label="Filtros activos">
			{#each data.filterChips as chip (chip.key)}
				<a class="filter-chip" href={chip.href} aria-label={`Quitar filtro ${chip.label}`}>
					{chip.label}<span aria-hidden="true">×</span>
				</a>
			{/each}
			<a class="clear-filters" href={data.clearHref}>Limpiar filtros</a>
		</div>
	{/if}

	{#if data.errorMessage}
		<p class="error" role="alert">{data.errorMessage}</p>
	{/if}

	{#if data.series.length > 0}
		<div class="series-grid">
			{#each data.series as series (series.id)}
				<SeriesCard {series} />
			{/each}
		</div>
	{:else if !data.errorMessage}
		<div class="empty-state">
			<p class="kicker">{hasFilters ? 'Sin coincidencias' : 'Estante vacío'}</p>
			{#if hasFilters}
				<h2>Ninguna serie coincide</h2>
				<p>Prueba a quitar algún filtro o a buscar de otra forma.</p>
				<a class="clear-filters" href={data.clearHref}>Limpiar filtros</a>
			{:else}
				<h2>Todavía no hay series en el estante</h2>
				<p>Las temporadas y los episodios aparecerán aquí cuando estén en la biblioteca.</p>
			{/if}
		</div>
	{/if}

	<Pagination current={data.page} last={data.last} url={data.currentUrl} />
</section>
