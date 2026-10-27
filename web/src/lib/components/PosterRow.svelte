<script lang="ts">
	import type { MediaItem } from '$lib/api';
	import PosterCard from './PosterCard.svelte';
	import RowScroller from './RowScroller.svelte';
	import VideoCard from './VideoCard.svelte';

	let {
		title,
		kicker = '',
		items,
		href = null,
		emptyMessage = null
	}: {
		title: string;
		kicker?: string;
		items: MediaItem[];
		href?: string | null;
		emptyMessage?: string | null;
	} = $props();
</script>

{#if items.length > 0}
	<RowScroller {title} {kicker} {href}>
		{#snippet children()}
			{#each items as item (item.id)}
				<li class:row-wide={item.kind === 'episode' || item.library_kind === 'clips'}>
					{#if item.library_kind === 'clips'}
						<VideoCard {item} />
					{:else}
						<PosterCard {item} posterUrl={item.poster_url} />
					{/if}
				</li>
			{/each}
		{/snippet}
	</RowScroller>
{:else if emptyMessage}
	<p class="admin-note">{emptyMessage}</p>
{/if}
