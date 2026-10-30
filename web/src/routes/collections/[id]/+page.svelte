<script lang="ts">
	import CollectionAdmin from '$lib/components/CollectionAdmin.svelte';
	import ListToggle from '$lib/components/ListToggle.svelte';
	import PosterCard from '$lib/components/PosterCard.svelte';
	import type { CollectionCandidate } from '$lib/api';
	import {
		firstUnwatched,
		formatCollectionCount,
		formatMissingYear,
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
		candidates?: CollectionCandidate[];
		matchQuery?: string;
		matchSearched?: boolean;
	}

	let { data, form }: { data: PageData; form?: ActionForm | null } = $props();

	const collection = $derived(data.collection);
	const colors = $derived(posterColors(collection.name));
	const viewed = $derived(viewedCount(collection.items));
	const playItem = $derived(firstUnwatched(collection.items));
	const pending = $derived(hasUnwatched(collection.items));
	const missingMovies = $derived(collection.missing ?? []);
	const actionForm = $derived((form ?? null) as ActionForm | null);
	const matchCandidates = $derived(actionForm?.candidates ?? []);
	const matchQuery = $derived(actionForm?.matchQuery ?? '');
	const matchSearched = $derived(Boolean(actionForm?.matchSearched));
	const manageOpen = $derived(Boolean(data.manage || actionForm?.manage));
</script>

<svelte:head>
	<title>{collection.name} · {data.ui?.server_name ?? 'Cinecat'}</title>
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
		<div class="detail-actions">
			{#if playItem}
				<a class="button" href={`/items/${playItem.id}/play`}>
					{pending ? 'Reproducir la primera sin ver' : 'Reproducir la primera'}
				</a>
			{/if}
			{#if collection.favorite !== undefined || collection.watchlist !== undefined}
				<ListToggle kind="collection" id={collection.id} list="favorite" active={collection.favorite ?? false} label="Favorito" />
				<ListToggle
					kind="collection"
					id={collection.id}
					list="watchlist"
					active={collection.watchlist ?? false}
					label="Próximas a ver"
				/>
			{/if}
		</div>
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

{#if missingMovies.length > 0}
	<section class="collection-missing" aria-labelledby="collection-missing-title">
		<div class="collection-missing-head">
			<p class="kicker">Saga incompleta</p>
			<h2 id="collection-missing-title">Faltan en tu biblioteca</h2>
			<p>Estas películas de la colección todavía no están en tu biblioteca.</p>
		</div>
		<ul class="missing-grid">
			{#each missingMovies as movie (movie.tmdb_id)}
				{@const movieColors = posterColors(movie.title)}
				<li class="missing-card">
					<div
						class="missing-frame"
						style={`--poster-a: ${movieColors.from}; --poster-b: ${movieColors.to};`}
					>
						{#if movie.poster_url}
							<img src={movie.poster_url} alt={`Cartel de ${movie.title}`} loading="lazy" />
						{:else}
							<span>{movie.title}</span>
						{/if}
					</div>
					<h3>{movie.title}</h3>
					<p>{formatMissingYear(movie.year)}</p>
				</li>
			{/each}
		</ul>
	</section>
{/if}

{#if data.isAdmin}
	<CollectionAdmin
		collection={data.collection}
		searchQuery={data.searchQuery}
		searchResults={data.searchResults}
		searchError={data.searchError}
		candidates={matchCandidates}
		matchQuery={matchQuery}
		matchSearched={matchSearched}
		open={manageOpen}
	/>
{/if}

<style>
	.collection-missing {
		margin-top: 2.4rem;
		padding-top: 1.4rem;
		border-top: 1px solid var(--borde-suave);
	}

	.collection-missing-head {
		max-width: 42rem;
	}

	.collection-missing-head h2 {
		margin: 0.25rem 0 0.35rem;
		color: var(--granate);
		font-family: 'Fraunces', Georgia, serif;
		font-size: clamp(1.6rem, 3vw, 2.2rem);
		font-weight: 600;
		line-height: 1.05;
	}

	.collection-missing-head p:last-child {
		margin: 0;
		color: var(--texto-apagado);
		font-size: 0.9rem;
		line-height: 1.5;
	}

	.missing-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(7.5rem, 1fr));
		gap: 1.4rem 1rem;
		margin: 1.2rem 0 0;
		padding: 0;
		list-style: none;
	}

	.missing-card {
		min-width: 0;
	}

	.missing-frame {
		position: relative;
		aspect-ratio: 2 / 3;
		overflow: hidden;
		border: 1px solid var(--borde-suave);
		border-radius: 4px;
		background: linear-gradient(155deg, var(--poster-a), var(--poster-b));
		box-shadow: 0 1rem 1.8rem -1rem var(--sombra);
	}

	.missing-frame img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}

	.missing-frame span {
		position: absolute;
		inset: 0;
		display: flex;
		align-items: flex-end;
		padding: 1rem 0.8rem;
		background: linear-gradient(180deg, transparent 25%, rgba(20, 10, 11, 0.78));
		color: var(--texto-sobre-oscuro);
		font-family: 'Fraunces', Georgia, serif;
		font-size: 1.1rem;
		font-weight: 600;
		line-height: 1.05;
	}

	.missing-card h3 {
		margin: 0.55rem 0 0.15rem;
		overflow: hidden;
		color: var(--granate);
		font-family: 'Fraunces', Georgia, serif;
		font-size: 0.98rem;
		font-weight: 600;
		line-height: 1.15;
		text-overflow: ellipsis;
	}

	.missing-card p {
		margin: 0;
		color: var(--texto-apagado);
		font-size: 0.78rem;
	}
</style>
