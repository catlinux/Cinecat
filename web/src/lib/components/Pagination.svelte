<script lang="ts">
	import { pageHref, pageWindow } from '$lib/pagination';

	let {
		current,
		last,
		url
	}: {
		current: number;
		last: number;
		url: string;
	} = $props();

	const pages = $derived(pageWindow(current, last));
</script>

{#if last > 1}
	<nav class="pagination" aria-label="Paginación">
		{#if current > 1}
			<a href={pageHref(url, current - 1)} aria-label="Página anterior">Anterior</a>
		{:else}
			<span class="pagination-disabled" aria-disabled="true">Anterior</span>
		{/if}

		{#each pages as page}
			{#if page === null}
				<span class="pagination-ellipsis" aria-hidden="true">…</span>
			{:else}
				<a
					href={pageHref(url, page)}
					aria-current={page === current ? 'page' : undefined}
					aria-label={`Página ${page}`}>{page}</a
				>
			{/if}
		{/each}

		{#if current < last}
			<a href={pageHref(url, current + 1)} aria-label="Página siguiente">Siguiente</a>
		{:else}
			<span class="pagination-disabled" aria-disabled="true">Siguiente</span>
		{/if}
	</nav>
{/if}
