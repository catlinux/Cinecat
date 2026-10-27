<script lang="ts">
	import type { Series } from '$lib/api';
	import SeriesCard from './SeriesCard.svelte';
	import RowScroller from './RowScroller.svelte';

	let {
		title,
		kicker = '',
		items,
		href = null
	}: {
		title: string;
		kicker?: string;
		items: Series[];
		href?: string | null;
	} = $props();
</script>

{#if items.length > 0}
	<RowScroller {title} {kicker} {href}>
		{#snippet children()}
			{#each items as series (series.id)}
				<li>
					<SeriesCard {series} />
				</li>
			{/each}
		{/snippet}
	</RowScroller>
{/if}
