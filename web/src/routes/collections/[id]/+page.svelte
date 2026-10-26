<script lang="ts">
	import CollectionAdmin from '$lib/components/CollectionAdmin.svelte';
	import PosterCard from '$lib/components/PosterCard.svelte';
	import {
		firstUnwatched,
		formatCollectionCount,
		formatViewedCount,
		hasUnwatched,
		viewedCount
	} from '$lib/collections';
	import { posterColors } from '$lib/poster-color';
	import type { PageData } from './$types';

	interface ActionForm {
		success?: boolean;
		manage?: boolean;
		message?: string;
	}

	let { data, form }: { data: PageData; form?: ActionForm | null } = $props();

	const collection = $derived(data.collection);
	const colors = $derived(posterColors(collection.name));
	const viewed = $derived(viewedCount(collection.items));
	const playItem = $derived(firstUnwatched(collection.items));
	const pending = $derived(hasUnwatched(collection.items));
	const actionForm = $derived((form ?? null) as ActionForm | null);
	const manageOpen = $derived(Boolean(data.manage || actionForm?.manage));
</script>

<svelte:head>
	<title>{collection.name} · Cinecat</title>
</svelte:head>

<a class="back" href="/collections">← Colecciones</a>

<section
	class="collection-hero"
	style={`--poster-a: ${colors.from}; --poster-b: ${colors.to};`}
>
	{#if collection.backdrop_url}
		<div class="collection-hero-media" aria-hidden="true">
			<img src={collection.backdrop_url} alt="" loading="lazy" />
		</div>
	{:else if collection.poster_url}
		<div class="collection-hero-media collection-hero-media--poster" aria-hidden="true">
			<img src={collection.poster_url} alt="" loading="lazy" />
		</div>
	{/if}

	<div class="collection-hero-content">
		<p class="kicker">Colección</p>
		<h1>{collection.name}</h1>
		{#if collection.overview}
			<p class="collection-overview">{collection.overview}</p>
		{/if}
		<p class="collection-meta">
			{formatCollectionCount(collection.item_count)}
			{#if viewed > 0}
				· {formatViewedCount(viewed, collection.items.length)}
			{/if}
		</p>
		{#if playItem}
			<a class="button" href={`/items/${playItem.id}/play`}>
				{pending ? 'Reproducir la primera sin ver' : 'Reproducir la primera'}
			</a>
		{/if}
	</div>
</section>

{#if actionForm?.message}
	<p
		class={actionForm.success ? 'success' : 'error'}
		role={actionForm.success ? 'status' : 'alert'}
	>
		{actionForm.message}
	</p>
{/if}

{#if collection.items.length > 0}
	<div class="poster-grid collection-movies">
		{#each collection.items as item (item.id)}
			<PosterCard {item} posterUrl={item.poster_url} />
		{/each}
	</div>
{:else}
	<div class="empty-state">
		<p class="kicker">Estante vacío</p>
		<h2>Esta colección todavía no tiene películas</h2>
		{#if data.isAdmin}
			<p>Puedes añadirlas desde la gestión de la colección.</p>
		{:else}
			<p>Vuelve más adelante para ver sus películas.</p>
		{/if}
	</div>
{/if}

{#if data.isAdmin}
	<CollectionAdmin
		collection={data.collection}
		searchQuery={data.searchQuery}
		searchResults={data.searchResults}
		searchError={data.searchError}
		open={manageOpen}
	/>
{/if}
