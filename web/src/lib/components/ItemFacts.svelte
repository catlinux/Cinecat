<script lang="ts">
	import type { MediaItem } from '$lib/api';

	let { item }: { item: MediaItem } = $props();

	const hasLinks = $derived(Boolean(item.imdb_id || item.tmdb_id));
</script>

{#if hasLinks}
	<dl class="item-facts">
		<div class="item-fact">
			<dt>Enlaces</dt>
			<dd class="item-links">
				{#if item.imdb_id}
					<a
						href={`https://www.imdb.com/title/${item.imdb_id}/`}
						target="_blank"
						rel="noopener noreferrer">Ficha en IMDb</a
					>
				{/if}
				{#if item.tmdb_id}
					<a
						href={`https://www.themoviedb.org/movie/${item.tmdb_id}`}
						target="_blank"
						rel="noopener noreferrer">Ficha en TMDB</a
					>
				{/if}
			</dd>
		</div>
	</dl>
{/if}
