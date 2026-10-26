<script lang="ts">
	import type { Credits } from '$lib/api';
	import { jobLabel } from '$lib/rows';
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

	type Entry = { key: string; name: string; photo: string | null; subtitle: string };

	const entries = $derived.by<Entry[]>(() => {
		const result: Entry[] = [];
		const crew = credits?.crew ?? [];
		const castCredits = credits?.cast ?? [];
		if (crew.length > 0) {
			for (const [index, member] of crew.entries()) {
				if (!member.name.trim()) {
					continue;
				}
				result.push({
					key: `crew-${index}-${member.name}`,
					name: member.name,
					photo: member.photo_url.trim() || null,
					subtitle: jobLabel(member.job)
				});
			}
		} else {
			for (const [index, name] of directors.entries()) {
				if (name.trim()) {
					result.push({
						key: `director-${index}-${name}`,
						name,
						photo: null,
						subtitle: 'Dirección'
					});
				}
			}
		}
		if (castCredits.length > 0) {
			for (const [index, member] of castCredits.entries()) {
				if (!member.name.trim()) {
					continue;
				}
				result.push({
					key: `cast-${index}-${member.name}`,
					name: member.name,
					photo: member.photo_url.trim() || null,
					subtitle: member.character ? `como ${member.character}` : ''
				});
			}
		} else {
			for (const [index, name] of cast.entries()) {
				if (name.trim()) {
					result.push({ key: `cast-name-${index}-${name}`, name, photo: null, subtitle: '' });
				}
			}
		}
		return result;
	});
</script>

{#if entries.length > 0}
	<RowScroller title="Reparto y equipo">
		{#snippet children()}
			{#each entries as entry (entry.key)}
				<li>
					<PersonCard name={entry.name} photoUrl={entry.photo} subtitle={entry.subtitle} />
				</li>
			{/each}
		{/snippet}
	</RowScroller>
{/if}
