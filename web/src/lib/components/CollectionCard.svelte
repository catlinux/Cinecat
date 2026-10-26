<script lang="ts">
	import type { Collection } from '$lib/api';
	import { formatCollectionCount } from '$lib/collections';
	import { posterColors } from '$lib/poster-color';

	let { collection }: { collection: Collection } = $props();

	const colors = $derived(posterColors(collection.name));
</script>

<a
	class="collection-card"
	href={`/collections/${collection.id}`}
	style={`--poster-a: ${colors.from}; --poster-b: ${colors.to};`}
>
	<div class="collection-frame">
		{#if collection.poster_url}
			<img
				src={collection.poster_url}
				alt={`Cartel de la colección ${collection.name}`}
				loading="lazy"
			/>
		{:else}
			<div class="collection-fallback">
				<span>{collection.name}</span>
			</div>
		{/if}
	</div>
	<div class="collection-caption">
		<h2>{collection.name}</h2>
		<p>{formatCollectionCount(collection.item_count)}</p>
	</div>
</a>
