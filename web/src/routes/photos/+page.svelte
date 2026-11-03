<script lang="ts">
	import { photoCountLabel, photoLibraryHref } from '$lib/photos';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const total = $derived(
		data.libraries.reduce((sum, library) => sum + (library.item_count ?? 0), 0)
	);
</script>

<svelte:head>
	<title>Fotos · {data.ui?.server_name ?? 'Cinecat'}</title>
</svelte:head>

<section class="library photos-library">
	<div class="library-head">
		<p class="kicker">Álbum de casa</p>
		<h1>Fotos</h1>
		{#if data.libraries.length > 0}
			<p class="library-count">
				{data.libraries.length}
				{data.libraries.length === 1 ? 'biblioteca' : 'bibliotecas'} · {photoCountLabel(total)}
			</p>
		{/if}
	</div>

	{#if data.errorMessage}
		<p class="error" role="alert">{data.errorMessage}</p>
	{:else if data.libraries.length > 0}
		<div class="photo-library-grid">
			{#each data.libraries as library (library.id)}
				<a class="photo-library-card" href={photoLibraryHref(library.id)}>
					<span class="photo-library-kicker">Biblioteca</span>
					<strong>{library.name}</strong>
					<span>{photoCountLabel(library.item_count ?? 0)}</span>
				</a>
			{/each}
		</div>
	{:else}
		<div class="empty-state">
			<p class="kicker">Sin fotografías</p>
			<h2>Todavía no hay ninguna biblioteca de fotos</h2>
			<p>
				Crea una biblioteca de tipo «Fotos», añade sus carpetas y lánzala desde el servidor
				para ver aquí tus imágenes.
			</p>
			{#if data.isAdmin}
				<a class="button button-secondary" href="/admin/libraries">Ir a Administración → Bibliotecas</a>
			{/if}
		</div>
	{/if}
</section>

<style>
	.photo-library-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(16rem, 1fr));
		gap: 1rem;
		margin-top: 1.5rem;
	}
	.photo-library-card {
		display: grid;
		gap: 0.25rem;
		min-width: 0;
		padding: 1.2rem;
		border: 1px solid var(--borde-suave);
		border-radius: 4px;
		background: var(--superficie);
		color: inherit;
		text-decoration: none;
	}
	.photo-library-card:hover,
	.photo-library-card:focus-visible {
		border-color: var(--naranja);
	}
	.photo-library-kicker {
		color: var(--texto-apagado);
		font-size: 0.64rem;
		font-weight: 700;
		letter-spacing: 0.12em;
		text-transform: uppercase;
	}
	.photo-library-card strong {
		overflow-wrap: anywhere;
		color: var(--granate);
		font-family: 'Fraunces', Georgia, serif;
		font-size: 1.35rem;
		line-height: 1.15;
	}
	.photo-library-card > span:last-child {
		color: var(--texto-apagado);
		font-size: 0.76rem;
	}
</style>
