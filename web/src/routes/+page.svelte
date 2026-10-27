<script lang="ts">
	import CategoryArt from '$lib/components/CategoryArt.svelte';
	import PosterRow from '$lib/components/PosterRow.svelte';
	import SeriesRow from '$lib/components/SeriesRow.svelte';
	import { DEFAULT_PUBLIC_UI } from '$lib/settings';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const ui = $derived({ ...DEFAULT_PUBLIC_UI, ...(data.ui ?? {}) });

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
		{
			id: 'movies',
			title: 'Películas',
			description: 'La cartelera completa, con fichas, filtros y búsqueda por reparto o dirección.',
			kicker: 'En cartel',
			href: '/movies',
			art: 'movies',
			count: data.errorMessage ? null : data.moviesTotal
		},
		{
			id: 'collections',
			title: 'Colecciones',
			description: 'Sagas y ciclos reunidos en un mismo estante.',
			kicker: 'Sagas',
			href: '/collections',
			art: 'collections',
			count: data.collectionsTotal
		},
		{
			id: 'series',
			title: 'Series',
			description: 'Temporadas, episodios y ese capítulo que siempre vuelve.',
			kicker: 'Series',
			href: '/series',
			art: 'series',
			count: data.seriesTotal
		},
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
			{#if data.continueWatching.length > 0}
				<PosterRow
					title="Continuar viendo"
					kicker="En pausa"
					items={data.continueWatching}
				/>
			{/if}

			{#if data.nextUp.length > 0}
				<PosterRow title="Siguiente episodio" kicker="Series" items={data.nextUp} />
			{/if}

			<PosterRow
				title="Películas añadidas recientemente"
				kicker="Estrenos"
				items={data.recent}
				href="/movies?sort=added"
				emptyMessage="Todavía no hay películas recientes. La cartelera está esperando."
			/>

			{#if ui.show_series && data.recentSeries.length > 0}
				<SeriesRow
					title="Series añadidas recientemente"
					kicker="Estrenos"
					items={data.recentSeries}
					href="/series?sort=added"
				/>
			{/if}
		</div>
	{/if}
</section>
