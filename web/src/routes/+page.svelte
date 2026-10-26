<script lang="ts">
	import PosterCard from '$lib/components/PosterCard.svelte';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();
</script>

<svelte:head>
	<title>Biblioteca · Cinecat</title>
</svelte:head>

<section class="library">
	<div class="library-head">
		<p class="kicker">Cartelera</p>
		<h1>Tu biblioteca</h1>
		<p class="library-count">
			{data.total} {data.total === 1 ? 'título en cartel' : 'títulos en cartel'}
		</p>
	</div>

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
			<p class="kicker">Sala vacía</p>
			<h2>Todavía no hay nada en cartel</h2>
			<p>
				Crea una biblioteca, lánzala desde el servidor y vuelve a pasar por taquilla. Los carteles
				aparecerán aquí.
			</p>
		</div>
	{/if}
</section>
