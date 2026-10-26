<script lang="ts">
	import type { MediaItem } from '$lib/api';
	import PosterCard from './PosterCard.svelte';

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

<section class="poster-row-section">
	<div class="poster-row-head">
		<div>
			{#if kicker}
				<p class="kicker">{kicker}</p>
			{/if}
			<h2>{title}</h2>
		</div>
		{#if href}
			<a class="text-link" href={href}>Ver todas →</a>
		{/if}
	</div>

	{#if items.length > 0}
		<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
		<div class="poster-row-scroll" role="region" aria-label={title} tabindex="0">
			<ul class="poster-row-list">
				{#each items as item (item.id)}
					<li class="poster-row-item">
						<PosterCard {item} posterUrl={item.poster_url} />
					</li>
				{/each}
			</ul>
		</div>
	{:else if emptyMessage}
		<p class="admin-note">{emptyMessage}</p>
	{/if}
</section>
