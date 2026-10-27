<script lang="ts">
	import FolderTile from '$lib/components/FolderTile.svelte';
	import Pagination from '$lib/components/Pagination.svelte';
	import VideoCard from '$lib/components/VideoCard.svelte';
	import {
		videoBreadcrumbs,
		videoCountLabel,
		videoFolderHref,
		videoHeading,
		videoRootHref
	} from '$lib/videos';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const breadcrumbs = $derived(
		videoBreadcrumbs(data.libraryID, data.libraryName, data.root, data.crumbs)
	);
	const heading = $derived(
		data.mode === 'browse'
			? videoHeading(data.path, data.crumbs, data.root?.name ?? data.libraryName)
			: data.libraryName
	);
	const total = $derived(data.mode === 'browse' ? data.total : data.roots.reduce((sum, root) => sum + root.item_count, 0));
</script>

<svelte:head>
	<title>{data.libraryName} · Vídeos · {data.ui?.server_name ?? 'Cinecat'}</title>
</svelte:head>

<section class="library videos-library">
	<nav class="breadcrumbs" aria-label="Ruta de la biblioteca">
		{#each breadcrumbs as crumb, index (crumb.href)}
			{#if index > 0}<span class="breadcrumb-separator" aria-hidden="true">›</span>{/if}
			{#if index === breadcrumbs.length - 1}
				<span aria-current="page">{crumb.name}</span>
			{:else}
				<a href={crumb.href}>{crumb.name}</a>
			{/if}
		{/each}
	</nav>

	<div class="library-head">
		<p class="kicker">Videoclips y conciertos</p>
		<h1>{heading}</h1>
		<p class="library-count">
			{videoCountLabel(total)}
			{#if data.mode === 'browse' && data.path}
				· {data.path}
			{/if}
		</p>
	</div>

	{#if data.mode === 'roots'}
		{#if data.roots.length > 0}
			<div class="folder-grid">
				{#each data.roots as root (root.id)}
					<FolderTile
						name={root.name}
						itemCount={root.item_count}
						href={videoRootHref(data.libraryID, root.id)}
					/>
				{/each}
			</div>
		{:else}
			<div class="empty-state">
				<p class="kicker">Sin carpetas</p>
				<h2>Esta biblioteca no tiene carpetas</h2>
				<p>Añade una carpeta desde Administración → Bibliotecas.</p>
			</div>
		{/if}
	{:else}
		{#if data.folders.length > 0}
			<section class="video-folder-section">
				<h2>Carpetas</h2>
				<div class="folder-grid">
					{#each data.folders as folder (folder.path)}
						<FolderTile
							name={folder.name}
							itemCount={folder.item_count}
							href={videoFolderHref(data.libraryID, data.root?.id ?? 0, folder.path)}
						/>
					{/each}
				</div>
			</section>
		{/if}

		{#if data.items.length > 0}
			<section class="video-files-section">
				<h2>Vídeos</h2>
				<div class="video-grid">
					{#each data.items as item (item.id)}
						<VideoCard {item} />
					{/each}
				</div>
			</section>
		{:else if data.folders.length === 0}
			<div class="empty-state">
				<p class="kicker">Carpeta vacía</p>
				<h2>No hay vídeos en esta carpeta</h2>
				<p>Prueba con otra carpeta o vuelve a la biblioteca.</p>
			</div>
		{/if}

		<Pagination current={data.page} last={data.last} url={data.currentUrl} />
	{/if}
</section>
