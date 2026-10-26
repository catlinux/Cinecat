<script lang="ts">
	import { enhance } from '$app/forms';
	import CollectionCard from '$lib/components/CollectionCard.svelte';
	import Pagination from '$lib/components/Pagination.svelte';
	import { COLLECTION_SORTS, collectionFiltersActive } from '$lib/collections';
	import type { PageData } from './$types';

	interface ActionForm {
		message?: string;
	}

	let { data, form }: { data: PageData; form?: ActionForm | null } = $props();

	const hasFilters = $derived(collectionFiltersActive(data.filters));
	const totalLabel = $derived(
		`${data.total} ${data.total === 1 ? 'colección' : 'colecciones'}`
	);
</script>

<svelte:head>
	<title>Colecciones · {data.ui?.server_name ?? 'Cinecat'}</title>
</svelte:head>

<section class="library collection-library">
	<div class="library-head">
		<p class="kicker">Sagas y ciclos</p>
		<h1>Colecciones</h1>
		<p class="library-count">{totalLabel}</p>
	</div>

	<form class="collection-toolbar" method="GET" action="/collections">
		<label>
			Buscar por nombre
			<input name="q" type="search" value={data.filters.q} />
		</label>
		<label>
			Orden
			<select name="sort">
				{#each COLLECTION_SORTS as option (option.value)}
					<option value={option.value} selected={option.value === data.filters.sort}>
						{option.label}
					</option>
				{/each}
			</select>
		</label>
		<button class="button" type="submit">Filtrar</button>
		{#if hasFilters}
			<a class="clear-filters" href="/collections">Limpiar filtros</a>
		{/if}
	</form>

	{#if data.isAdmin}
		<details class="admin-card collection-create" open={Boolean(form)}>
			<summary class="button">Nueva colección</summary>
			<form method="POST" action="?/create" use:enhance>
				<label>
					Nombre
					<input name="name" type="text" maxlength="120" required />
				</label>
				<label>
					Descripción (opcional)
					<textarea name="overview" rows="3" maxlength="2000"></textarea>
				</label>
				<button class="button" type="submit">Crear colección</button>
			</form>
		</details>
	{/if}

	{#if form?.message}
		<p class="error" role="alert">{form.message}</p>
	{/if}

	{#if data.errorMessage}
		<p class="error" role="alert">{data.errorMessage}</p>
	{/if}

	{#if data.collections.length > 0}
		<div class="collection-grid">
			{#each data.collections as collection (collection.id)}
				<CollectionCard {collection} />
			{/each}
		</div>
	{:else if !data.errorMessage}
		<div class="empty-state">
			{#if hasFilters}
				<p class="kicker">Sin coincidencias</p>
				<h2>Ninguna colección coincide</h2>
				<p>Prueba con otro nombre o cambia el orden.</p>
				<a class="clear-filters" href="/collections">Limpiar filtros</a>
			{:else}
				<p class="kicker">Estante vacío</p>
				<h2>Todavía no hay colecciones</h2>
				{#if data.isAdmin}
					<p>Las sagas de TMDB aparecerán aquí y también puedes crear las tuyas.</p>
				{:else}
					<p>Las sagas de TMDB aparecerán aquí cuando tengan películas.</p>
				{/if}
			{/if}
		</div>
	{/if}

	<Pagination current={data.page} last={data.last} url={data.currentUrl} />
</section>
