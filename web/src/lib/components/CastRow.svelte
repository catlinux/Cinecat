<script lang="ts">
	import type { Credits } from '$lib/api';
	import { creditRows } from '$lib/rows';
	import PersonCard from './PersonCard.svelte';
	import RowScroller from './RowScroller.svelte';

	let {
		credits = null,
		cast = [],
		directors = []
	}: {
		credits?: Credits | null;
		cast?: string[];
		directors?: string[];
	} = $props();

	const entries = $derived(creditRows(credits, cast, directors));
</script>

{#if entries.length > 0}
	<RowScroller title="Reparto y equipo">
		{#snippet children()}
			{#each entries as entry (entry.key)}
				<li class="row-fixed">
					<PersonCard name={entry.name} photoUrl={entry.photo} subtitle={entry.subtitle} />
				</li>
			{/each}
		{/snippet}
	</RowScroller>
{/if}
