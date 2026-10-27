<script lang="ts">
	import { posterColors } from '$lib/poster-color';
	import { personInitials } from '$lib/rows';

	let {
		name,
		photoUrl = null,
		subtitle = '',
		href = null
	}: {
		name: string;
		photoUrl?: string | null;
		subtitle?: string;
		href?: string | null;
	} = $props();

	const colors = $derived(posterColors(name));
	const initials = $derived(personInitials(name));
</script>

{#snippet cardContent()}
	<div class="person-photo">
		{#if photoUrl}
			<img src={photoUrl} alt={name} loading="lazy" />
		{:else}
			<div class="person-fallback" aria-hidden="true">{initials}</div>
		{/if}
	</div>
	<h3>{name}</h3>
	{#if subtitle}<p>{subtitle}</p>{/if}
{/snippet}

{#if href}
	<a
		class="person-card person-card-link"
		{href}
		style={`--person-a: ${colors.from}; --person-b: ${colors.to};`}
	>
		{@render cardContent()}
	</a>
{:else}
	<article
		class="person-card"
		style={`--person-a: ${colors.from}; --person-b: ${colors.to};`}
	>
		{@render cardContent()}
	</article>
{/if}

<style>
	.person-card {
		display: block;
		width: 7.5rem;
		max-width: 7.5rem;
	}
	.person-photo {
		position: relative;
		aspect-ratio: 2 / 3;
		overflow: hidden;
		border-radius: 6px;
		background: linear-gradient(145deg, var(--person-a), var(--person-b));
	}
	.person-card-link {
		color: inherit;
		text-decoration: none;
	}

	.person-card-link:focus-visible {
		outline: 2px solid var(--naranja);
		outline-offset: 3px;
	}

	.person-photo img {
		display: block;
		width: 100%;
		height: 100%;
		object-fit: cover;
	}
	.person-fallback {
		display: grid;
		place-items: center;
		width: 100%;
		height: 100%;
		color: rgba(255, 250, 240, 0.92);
		font-family: 'Fraunces', Georgia, serif;
		font-size: 2rem;
		font-weight: 700;
		letter-spacing: 0.03em;
	}
	h3 {
		margin: 0.45rem 0 0;
		font-family: 'Fraunces', Georgia, serif;
		font-size: 0.82rem;
		line-height: 1.2;
	}
	p {
		margin: 0.15rem 0 0;
		color: var(--tinta);
		opacity: 0.72;
		font-size: 0.7rem;
		line-height: 1.25;
	}
</style>
