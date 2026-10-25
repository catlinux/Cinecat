<script lang="ts">
	import type { MediaItem } from '$lib/api';
	import { itemSubtitle, kindLabel } from '$lib/media';
	import { posterColors } from '$lib/poster-color';

	let {
		item,
		posterUrl = null
	}: {
		item: MediaItem;
		posterUrl?: string | null;
	} = $props();

	const colors = $derived(posterColors(item.title));
</script>

<a
	class="poster-card"
	href={`/items/${item.id}`}
	style={`--poster-a: ${colors.from}; --poster-b: ${colors.to};`}
>
	<div class="poster-frame">
		{#if posterUrl}
			<img class="poster-image" src={posterUrl} alt={`Cartel de ${item.title}`} loading="lazy" />
		{:else}
			<div class="poster-fallback">
				<span class="poster-title">{item.title}</span>
			</div>
		{/if}
		<span class="poster-kind">{kindLabel(item.kind)}</span>
	</div>

	<div class="poster-caption">
		<h2>{item.title}</h2>
		<p>{itemSubtitle(item)}</p>
	</div>
</a>
