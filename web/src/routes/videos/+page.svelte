<script lang="ts">
	import FolderTile from '$lib/components/FolderTile.svelte';
	import { videoCountLabel } from '$lib/videos';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const total = $derived(
		data.libraries.reduce((sum, library) => sum + (library.item_count ?? 0), 0)
	);
</script>

<svelte:head>
	<title>Vídeos · {data.ui?.server_name ?? 'Cinecat'}</title>
</svelte:head>

<section class="library videos-library">
	<div class="library-head">
		<p class="kicker">Videoclips y conciertos</p>
		<h1>Vídeos</h1>
		{#if data.libraries.length > 0}
			<p class="library-count">
				{data.libraries.length}
				{data.libraries.length === 1 ? 'biblioteca' : 'bibliotecas'} · {videoCountLabel(total)}
			</p>
		{/if}
	</div>

	{#if data.errorMessage}
		<p class="error" role="alert">{data.errorMessage}</p>
	{:else if data.libraries.length > 0}
		<div class="video-library-grid">
			{#each data.libraries as library (library.id)}
				<FolderTile
					name={library.name}
					itemCount={library.item_count ?? 0}
					href={`/videos/${library.id}`}
				/>
			{/each}
		</div>
	{:else}
		<div class="empty-state">
			<p class="kicker">Sala de vídeos</p>
			<h2>Aún no hay vídeos en Cinecat</h2>
			<p>
				Crea una biblioteca de tipo «Vídeos» y escanéala para ver aquí tus videoclips y
				conciertos.
			</p>
			{#if data.isAdmin}
				<a class="button button-secondary" href="/admin/libraries">Ir a Administración → Bibliotecas</a>
			{/if}
		</div>
	{/if}
</section>
