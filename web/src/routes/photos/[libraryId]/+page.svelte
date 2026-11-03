<script lang="ts">
	import PhotoViewer from '$lib/components/PhotoViewer.svelte';
	import Pagination from '$lib/components/Pagination.svelte';
	import {
		PHOTO_SORTS,
		PHOTO_SORT_LABELS,
		photoAspectRatio,
		photoCountLabel,
		photoDateLabel,
		photoFolderHref,
		photoThumbURL
	} from '$lib/photos';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	let viewerIndex = $state<number | null>(null);
</script>

<svelte:head>
	<title>{data.libraryName} · Fotos · {data.ui?.server_name ?? 'Cinecat'}</title>
</svelte:head>

<section class="library photos-library">
	<nav class="breadcrumbs" aria-label="Ruta de la biblioteca">
		<a href={photoFolderHref(data.libraryID, '', data.sort)}>{data.libraryName}</a>
		{#each data.crumbs as crumb, index (crumb.path)}
			<span class="breadcrumb-separator" aria-hidden="true">›</span>
			{#if index === data.crumbs.length - 1}
				<span aria-current="page">{crumb.name}</span>
			{:else}
				<a href={photoFolderHref(data.libraryID, crumb.path, data.sort)}>{crumb.name}</a>
			{/if}
		{/each}
	</nav>

	<div class="library-head">
		<p class="kicker">Álbum de casa</p>
		<h1>{data.libraryName}</h1>
		<p class="library-count">{photoCountLabel(data.total)}</p>
	</div>

	{#if data.errorMessage}
		<p class="error" role="alert">{data.errorMessage}</p>
	{:else}
		<div class="photo-toolbar">
			<form method="GET" action={`/photos/${data.libraryID}`} class="photo-sort-form">
				<label>
					<span>Orden</span>
					<select name="sort">
						{#each PHOTO_SORTS as option (option)}
							<option value={option} selected={data.sort === option}>
								{PHOTO_SORT_LABELS[option]}
							</option>
						{/each}
					</select>
				</label>
				{#if data.folder}
					<input type="hidden" name="folder" value={data.folder} />
				{/if}
				<button type="submit" class="button button-secondary">Aplicar</button>
			</form>
			<p class="photo-toolbar-note">Las miniaturas se cargan al acercarte a ellas.</p>
		</div>

		{#if data.folders.length > 0}
			<section class="photo-section">
				<h2>Carpetas</h2>
				<div class="photo-folder-grid">
					{#each data.folders as folder (folder)}
						<a
							class="photo-folder-card"
							href={photoFolderHref(data.libraryID, data.folder ? `${data.folder}/${folder}` : folder, data.sort)}
						>
							<span aria-hidden="true">▸</span>
							<strong>{folder}</strong>
						</a>
					{/each}
				</div>
			</section>
		{/if}

		{#if data.items.length > 0}
			<section class="photo-section">
				<h2>Fotografías</h2>
				<div class="photo-grid">
					{#each data.items as photo, index (photo.id)}
						<button
							type="button"
							class="photo-card"
							onclick={() => (viewerIndex = index)}
							aria-label={`Abrir ${photo.name}`}
						>
							<span
								class="photo-frame"
								style={`aspect-ratio: ${photoAspectRatio(photo.width, photo.height)};`}
							>
								<img
									src={photoThumbURL(photo.id)}
									alt={photo.name}
									loading="lazy"
									decoding="async"
								/>
							</span>
							<span class="photo-caption">
								<strong>{photo.name}</strong>
								{#if photoDateLabel(photo.taken_at)}
									<span>{photoDateLabel(photo.taken_at)}</span>
								{/if}
							</span>
						</button>
					{/each}
				</div>
			</section>
		{:else if data.folders.length === 0}
			<div class="empty-state">
				<p class="kicker">Carpeta vacía</p>
				<h2>No hay fotos en esta carpeta</h2>
				<p>Prueba con otra carpeta o vuelve a la biblioteca.</p>
			</div>
		{/if}

		<Pagination current={data.page} last={data.last} url={data.currentUrl} />
	{/if}
</section>

{#if viewerIndex !== null}
	<PhotoViewer
		photos={data.items}
		startIndex={viewerIndex}
		onclose={() => (viewerIndex = null)}
	/>
{/if}

<style>
	.photo-toolbar {
		display: flex;
		flex-wrap: wrap;
		align-items: end;
		justify-content: space-between;
		gap: 0.8rem;
		margin: 1rem 0 1.6rem;
		padding: 0.9rem 0;
		border-top: 1px solid var(--borde-suave);
		border-bottom: 1px solid var(--borde-suave);
	}
	.photo-sort-form {
		display: flex;
		flex-wrap: wrap;
		align-items: end;
		gap: 0.6rem;
	}
	.photo-sort-form label {
		display: grid;
		gap: 0.25rem;
		color: var(--granate);
		font-size: 0.65rem;
		font-weight: 600;
		letter-spacing: 0.1em;
		text-transform: uppercase;
	}
	.photo-sort-form select {
		padding: 0.5rem 0.65rem;
		border: 1px solid var(--borde-medio);
		border-radius: 3px;
		background: var(--crema-clara);
		color: var(--tinta);
		font-family: Inter, system-ui, sans-serif;
		font-size: 0.85rem;
		letter-spacing: normal;
		text-transform: none;
	}
	.photo-sort-form .button {
		margin: 0;
		padding: 0.55rem 0.9rem;
	}
	.photo-toolbar-note {
		margin: 0;
		color: var(--texto-apagado);
		font-size: 0.7rem;
	}
	.photo-section {
		margin-top: 1.6rem;
	}
	.photo-section h2 {
		margin: 0 0 0.7rem;
		font-family: 'Fraunces', Georgia, serif;
		font-size: clamp(1.2rem, 2vw, 1.55rem);
		color: var(--granate);
	}
	.photo-folder-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(13rem, 1fr));
		gap: 0.7rem;
	}
	.photo-folder-card {
		display: flex;
		align-items: baseline;
		gap: 0.5rem;
		min-width: 0;
		padding: 0.8rem 0.9rem;
		border: 1px solid var(--borde-suave);
		border-radius: 4px;
		background: var(--superficie);
		color: var(--granate);
		text-decoration: none;
	}
	.photo-folder-card:hover,
	.photo-folder-card:focus-visible {
		border-color: var(--naranja);
	}
	.photo-folder-card > span {
		color: var(--naranja);
	}
	.photo-folder-card strong {
		overflow-wrap: anywhere;
		font-family: 'Fraunces', Georgia, serif;
		font-weight: 600;
	}
	.photo-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(12rem, 1fr));
		gap: 1.1rem;
	}
	.photo-card {
		display: grid;
		gap: 0.45rem;
		min-width: 0;
		padding: 0;
		border: 0;
		background: transparent;
		color: inherit;
		text-align: left;
		cursor: pointer;
	}
	.photo-frame {
		display: block;
		overflow: hidden;
		border: 1px solid var(--borde-suave);
		border-radius: 4px;
		background: var(--superficie);
	}
	.photo-card:hover .photo-frame,
	.photo-card:focus-visible .photo-frame {
		border-color: var(--naranja);
	}
	.photo-frame img {
		display: block;
		width: 100%;
		height: 100%;
		object-fit: cover;
	}
	.photo-caption {
		display: grid;
		gap: 0.1rem;
		min-width: 0;
	}
	.photo-caption strong {
		overflow-wrap: anywhere;
		color: var(--tinta);
		font-size: 0.78rem;
		font-weight: 600;
		line-height: 1.25;
	}
	.photo-caption span {
		color: var(--texto-apagado);
		font-size: 0.68rem;
	}
	@media (max-width: 620px) {
		.photo-grid {
			grid-template-columns: repeat(auto-fill, minmax(8.5rem, 1fr));
			gap: 0.75rem;
		}
	}
</style>
