<script lang="ts">
	import type { SeriesSeason } from '$lib/api';
	import { posterColors } from '$lib/poster-color';
	import { personInitials } from '$lib/rows';

	let {
		seasons,
		activeSeason,
		seriesPoster = null,
		seriesTitle
	}: {
		seasons: SeriesSeason[];
		activeSeason: number | null;
		seriesPoster?: string | null;
		seriesTitle: string;
	} = $props();

	const colors = $derived(posterColors(seriesTitle));
	const initials = $derived(personInitials(seriesTitle));
</script>

<nav class="season-cards" aria-label="Temporadas">
	<ul>
		{#each seasons as season (season.season)}
			{@const active = season.season === activeSeason}
			{@const poster = season.poster_url || seriesPoster}
			<li>
				<a
					class="season-card"
					class:season-card-active={active}
					href={`?season=${season.season}`}
					aria-current={active ? 'page' : undefined}
				>
					<span
						class="season-card-frame"
						style={`--poster-a: ${colors.from}; --poster-b: ${colors.to};`}
					>
						{#if poster}
							<img src={poster} alt="" loading="lazy" />
						{:else}
							<span class="season-card-fallback" aria-hidden="true">{initials}</span>
						{/if}
					</span>
					<span class="season-card-name">{season.name}</span>
					<span class="season-card-count">
						{season.watched_count}/{season.episode_count} vistos
					</span>
				</a>
			</li>
		{/each}
	</ul>
</nav>

<style>
	.season-cards {
		margin: 1.5rem 0 1rem;
		overflow-x: auto;
		scrollbar-width: thin;
	}
	.season-cards ul {
		display: flex;
		gap: 0.75rem;
		margin: 0;
		padding: 0 0 0.3rem;
		list-style: none;
	}
	.season-cards li {
		flex: 0 0 6.5rem;
	}
	.season-card {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
		color: var(--texto-tenue);
		text-decoration: none;
	}
	.season-card-frame {
		position: relative;
		display: block;
		aspect-ratio: 2 / 3;
		overflow: hidden;
		border: 2px solid transparent;
		border-radius: 4px;
		background: linear-gradient(145deg, var(--poster-a), var(--poster-b));
	}
	.season-card-frame img {
		display: block;
		width: 100%;
		height: 100%;
		object-fit: cover;
	}
	.season-card-fallback {
		display: grid;
		place-items: center;
		width: 100%;
		height: 100%;
		color: rgba(255, 250, 240, 0.92);
		font-family: 'Fraunces', Georgia, serif;
		font-size: 1.4rem;
		font-weight: 700;
	}
	.season-card-active .season-card-frame {
		border-color: var(--naranja);
		box-shadow: 0 0 0 2px var(--tinte);
	}
	.season-card-name {
		margin-top: 0.15rem;
		overflow-wrap: anywhere;
		font-family: 'Fraunces', Georgia, serif;
		font-size: 0.78rem;
		font-weight: 600;
		line-height: 1.2;
	}
	.season-card-count {
		color: var(--texto-apagado);
		font-size: 0.66rem;
		line-height: 1.25;
	}
	.season-card-active .season-card-name,
	.season-card:hover .season-card-name {
		color: var(--naranja);
	}

	@media (max-width: 600px) {
		.season-cards {
			overflow: visible;
		}
		.season-cards ul {
			flex-wrap: wrap;
		}
	}
</style>
