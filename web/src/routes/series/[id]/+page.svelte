<script lang="ts">
	import { enhance } from '$app/forms';
	import EpisodeRow from '$lib/components/EpisodeRow.svelte';
	import { formatGenres, formatRating, overviewInSpanish, splitCast } from '$lib/media';
	import { posterColors } from '$lib/poster-color';
	import {
		formatSeriesCounts,
		seasonWatched,
		seriesPlayChoice,
		seriesWatched
	} from '$lib/series';
	import type { PageData } from './$types';

	interface ActionForm {
		success?: boolean;
		message?: string;
	}

	let { data, form }: { data: PageData; form?: ActionForm | null } = $props();

	const series = $derived(data.series);
	const colors = $derived(posterColors(series.title));
	const firstAvailable = $derived(data.firstEpisode ?? data.season?.items[0] ?? null);
	const playChoice = $derived(seriesPlayChoice(series, series.next_episode, firstAvailable));
	const allWatched = $derived(seriesWatched(series));
	const actionForm = $derived((form ?? null) as ActionForm | null);
	const originalTitle = $derived(
		series.original_title &&
			series.original_title.trim() !== '' &&
			series.original_title !== series.title
			? series.original_title
			: ''
	);
	const meta = $derived(
		[
			series.year !== null ? String(series.year) : '',
			formatGenres(series.genres),
			formatRating(series.rating),
			formatSeriesCounts(series.season_count, series.episode_count)
		]
			.filter(Boolean)
			.join(' · ')
	);
	const languageWarning = $derived(
		Boolean(series.overview) && !overviewInSpanish(series.overview_language)
	);
	const unidentified = $derived(series.match_status === null || series.match_status === 'none');
	const cast = $derived(splitCast(series.cast));
	const currentSeason = $derived(
		series.seasons.find((season) => season.season === data.selectedSeason) ?? null
	);
	const currentSeasonWatched = $derived(currentSeason ? seasonWatched(currentSeason) : false);
</script>

<svelte:head>
	<title>{series.title} · Cinecat</title>
</svelte:head>

<a class="back" href="/series">← Series</a>

<section class="detail detail-movie series-detail">
	{#if series.backdrop_url}
		<div class="detail-backdrop" aria-hidden="true">
			<img src={series.backdrop_url} alt="" loading="lazy" />
		</div>
	{/if}

	<div
		class="detail-poster"
		class:detail-poster--image={Boolean(series.poster_url)}
		style={`--poster-a: ${colors.from}; --poster-b: ${colors.to};`}
		aria-hidden="true"
	>
		<span class="poster-kind">Serie</span>
		{#if series.poster_url}
			<img
				class="detail-poster-img"
				src={series.poster_url}
				alt={`Cartel de ${series.title}`}
				loading="lazy"
			/>
		{:else}
			<span class="detail-poster-title">{series.title}</span>
		{/if}
	</div>

	<div class="detail-info">
		<p class="kicker">Serie</p>
		<h1>{series.title}</h1>

		{#if originalTitle}
			<p class="detail-original-title"><em>{originalTitle}</em></p>
		{/if}

		{#if meta}
			<p class="detail-meta">{meta}</p>
		{/if}

		{#if unidentified}
			<p class="detail-status">Sin identificar</p>
		{/if}

		{#if series.tagline}
			<blockquote class="detail-tagline">
				<p>«{series.tagline}»</p>
			</blockquote>
		{/if}

		{#if series.overview}
			<p class="detail-overview">{series.overview}</p>
		{/if}

		{#if languageWarning}
			<p class="detail-note">Sinopsis no disponible en castellano.</p>
		{/if}

		{#if series.tmdb_id}
			<p class="series-tmdb">
				<a
					class="text-link"
					href={`https://www.themoviedb.org/tv/${series.tmdb_id}`}
					target="_blank"
					rel="noreferrer"
				>
					Ver en TMDB
				</a>
			</p>
		{/if}

		<div class="detail-actions">
			{#if playChoice}
				<a class="button" href={`/items/${playChoice.item.id}/play`}>{playChoice.label}</a>
			{/if}
			{#if series.episode_count > 0}
				<form method="POST" action="?/watchedSeries" use:enhance>
					<input type="hidden" name="watched" value={allWatched ? '0' : '1'} />
					<button class="button button-secondary" type="submit" aria-pressed={allWatched}>
						{allWatched ? 'Marcar serie como no vista' : 'Marcar serie como vista'}
					</button>
				</form>
			{/if}
		</div>

		{#if actionForm?.message}
			<p
				class={actionForm.success ? 'success' : 'error'}
				role={actionForm.success ? 'status' : 'alert'}
			>
				{actionForm.message}
			</p>
		{/if}

		{#if series.directors.length > 0}
			<section class="series-credits">
				<h2>Creada por</h2>
				<p>{series.directors.join(', ')}</p>
			</section>
		{/if}

		{#if cast.visible.length > 0}
			<section class="series-credits">
				<h2>Reparto</h2>
				<p>{cast.visible.join(', ')}</p>
				{#if cast.hidden.length > 0}
					<details class="series-cast-more">
						<summary>Ver todo el reparto</summary>
						<p>{cast.hidden.join(', ')}</p>
					</details>
				{/if}
			</section>
		{/if}

		<!-- Aquí irá «Corregir ficha» de la serie -->
	</div>
</section>

{#if series.seasons.length > 0}
	<nav class="season-tabs" aria-label="Temporadas">
		{#each series.seasons as season (season.season)}
			<a
				class="season-tab"
				class:season-tab-active={season.season === data.selectedSeason}
				href={`?season=${season.season}`}
				aria-current={season.season === data.selectedSeason ? 'page' : undefined}
			>
				Temporada {season.season} · {season.watched_count}/{season.episode_count} vistos
			</a>
		{/each}
	</nav>

	{#if data.season}
		<section class="season-panel">
			<div class="season-head">
				<div>
					<p class="kicker">Temporada {data.season.season}</p>
					<h2>{data.season.name}</h2>
					{#if data.season.overview}
						<p class="season-overview">{data.season.overview}</p>
					{/if}
				</div>
				<form method="POST" action="?/watchedSeason" use:enhance>
					<input type="hidden" name="season" value={data.season.season} />
					<input
						type="hidden"
						name="watched"
						value={currentSeasonWatched ? '0' : '1'}
					/>
					<button
						class="button button-secondary"
						type="submit"
						aria-pressed={currentSeasonWatched}
					>
						{currentSeasonWatched
							? 'Marcar temporada como no vista'
							: 'Marcar temporada como vista'}
					</button>
				</form>
			</div>

			<ul class="episode-list">
				{#each data.season.items as item (item.id)}
					<li><EpisodeRow {item} /></li>
				{/each}
			</ul>
		</section>
	{/if}
{:else}
	<div class="empty-state">
		<p class="kicker">Estante vacío</p>
		<h2>Esta serie todavía no tiene episodios</h2>
		<p>Cuando el escáner encuentre archivos, aparecerán aquí.</p>
	</div>
{/if}
