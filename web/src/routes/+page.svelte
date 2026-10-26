<script lang="ts">
	import CategoryArt from '$lib/components/CategoryArt.svelte';
	import PosterRow from '$lib/components/PosterRow.svelte';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	type PortalCategory = {
		id: string;
		title: string;
		description: string;
		kicker: string;
		href: string | null;
		art: 'movies' | 'collections' | 'series';
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
			kicker: 'En preparación',
			href: null,
			art: 'collections',
			count: null
		},
		{
			id: 'series',
			title: 'Series',
			description: 'Temporadas, episodios y ese capítulo que siempre vuelve.',
			kicker: 'En preparación',
			href: null,
			art: 'series',
			count: null
		}
	]);

	const movieCount = $derived(
		data.errorMessage
			? 'Abrir cartelera'
			: `${data.moviesTotal} ${data.moviesTotal === 1 ? 'película' : 'películas'}`
	);
</script>

<svelte:head>
	<title>Cinecat · Tu cine de barrio</title>
</svelte:head>

<section class="portal">
	<div class="portal-hero">
		<p class="kicker">Bienvenido a la sala</p>
		<h1>Tu cine de barrio</h1>
		<p class="portal-lede">
			Una biblioteca personal con olor a palomitas, carteles pintados a mano y la sesión de
			siempre esperándote.
		</p>
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
							{category.id === 'movies' ? movieCount : 'Entrar'}
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
				title="Añadido recientemente"
				kicker="Estrenos"
				items={data.recent}
				href="/movies?sort=added"
				emptyMessage="Todavía no hay películas recientes. La cartelera está esperando."
			/>
		</div>
	{/if}
</section>
