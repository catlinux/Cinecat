<script lang="ts">
	import type { MediaItem } from '$lib/api';

	let {
		previous = null,
		next = null
	}: {
		previous?: MediaItem | null;
		next?: MediaItem | null;
	} = $props();

	function label(item: MediaItem): string {
		if (item.season !== null && item.episode !== null) {
			return `T${item.season} E${item.episode} · ${item.title}`;
		}
		return item.title;
	}
</script>

{#if previous || next}
	<nav class="episode-nav" aria-label="Episodios">
		{#if previous}
			<a class="episode-nav-link" href={`/items/${previous.id}`}>
				<span class="episode-nav-label">← Episodio anterior</span>
				<span class="episode-nav-title">{label(previous)}</span>
			</a>
		{/if}
		{#if next}
			<a class="episode-nav-link episode-nav-link--next" href={`/items/${next.id}`}>
				<span class="episode-nav-label">Siguiente episodio →</span>
				<span class="episode-nav-title">{label(next)}</span>
			</a>
		{/if}
	</nav>
{/if}
