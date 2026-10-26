<script lang="ts">
	import type { MediaItem } from '$lib/api';
	import { splitCast } from '$lib/media';

	let { item }: { item: MediaItem } = $props();

	const cast = $derived(splitCast(item.cast));
	const hasLinks = $derived(Boolean(item.imdb_id || item.tmdb_id));
	const hasFacts = $derived(
		item.directors.length > 0 || cast.visible.length > 0 || hasLinks
	);
</script>

{#if hasFacts}
	<dl class="item-facts">
		{#if item.directors.length > 0}
			<div class="item-fact">
				<dt>Dirección</dt>
				<dd>{item.directors.join(', ')}</dd>
			</div>
		{/if}

		{#if cast.visible.length > 0}
			<div class="item-fact">
				<dt>Reparto</dt>
				<dd>
					<ul class="cast-list">
						{#each cast.visible as name, index (index)}
							<li>{name}</li>
						{/each}
					</ul>
					{#if cast.hidden.length > 0}
						<details class="cast-more">
							<summary>Ver resto del reparto ({cast.hidden.length})</summary>
							<ul class="cast-list">
								{#each cast.hidden as name, index (index)}
									<li>{name}</li>
								{/each}
							</ul>
						</details>
					{/if}
				</dd>
			</div>
		{/if}

		{#if hasLinks}
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
		{/if}
	</dl>
{/if}
