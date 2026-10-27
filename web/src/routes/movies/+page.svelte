<script lang="ts">
	import LibraryFilters from '$lib/components/LibraryFilters.svelte';
	import Pagination from '$lib/components/Pagination.svelte';
	import PosterCard from '$lib/components/PosterCard.svelte';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const hasFilters = $derived(data.filterChips.length > 0);
</script>

<svelte:head>
	<title>Películas · {data.ui?.server_name ?? 'Cinecat'}</title>
</svelte:head>

<section class="library">
	<div class="library-head">
		<p class="kicker">Cartelera</p>
		<h1>Películas</h1>
		<p class="library-count">
			{data.total} {data.total === 1 ? 'título' : 'títulos'}
		</p>
	</div>

	<LibraryFilters
		filters={data.filters}
		genres={data.genres}
		isAdmin={data.isAdmin}
		unmatchedCount={data.unmatchedCount}
		currentUrl={data.currentUrl}
		showCollection
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

	{#if data.items.length > 0}
		<div class="poster-grid">
			{#each data.items as item (item.id)}
				<PosterCard {item} posterUrl={item.poster_url} />
			{/each}
		</div>
	{:else if !data.errorMessage}
		<div class="empty-state">
			<p class="kicker">{hasFilters ? 'Sin coincidencias' : 'Sala vacía'}</p>
			{#if hasFilters}
				<h2>Ninguna película coincide</h2>
				<p>Prueba a quitar algún filtro o a buscar de otra forma.</p>
				<a class="clear-filters" href={data.clearHref}>Limpiar filtros</a>
			{:else}
				<h2>Todavía no hay películas en cartel</h2>
				<p>
					Crea una biblioteca, lánzala desde el servidor y vuelve a pasar por taquilla. Los
					carteles aparecerán aquí.
				</p>
			{/if}
		</div>
	{/if}

	<Pagination current={data.page} last={data.last} url={data.currentUrl} />
</section>
