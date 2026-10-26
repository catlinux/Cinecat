<script lang="ts">
	import { enhance } from '$app/forms';
	import type { Candidate, MediaItem } from '$lib/api';
	import { matchStatusLabel, truncateSynopsis } from '$lib/media';

	interface MatchSearch {
		q: string;
		year: number | null;
	}

	type MatchDialogKind = 'movie' | 'series';

	interface MatchDialogItem {
		title: string;
		file_title: string;
		year: number | null;
		match_status: MediaItem['match_status'];
		match_score: number | null;
		path?: string;
		folder?: string;
	}

	interface MatchDialogActions {
		candidates: string;
		match: string;
		nomatch: string;
		automatic: string;
	}

	const movieActions: MatchDialogActions = {
		candidates: 'candidates',
		match: 'match',
		nomatch: 'nomatch',
		automatic: 'automatic'
	};

	let {
		item,
		candidates,
		search,
		open,
		kind = 'movie',
		defaultQuery = null,
		actions = movieActions
	}: {
		item: MatchDialogItem;
		candidates: Candidate[];
		search: MatchSearch | null;
		open: boolean;
		kind?: MatchDialogKind;
		defaultQuery?: string | null;
		actions?: MatchDialogActions;
	} = $props();

	const status = $derived(matchStatusLabel(item));
	const query = $derived(search ? search.q : (defaultQuery ?? item.file_title));
	const year = $derived(search ? (search.year ?? '') : (item.year ?? ''));
	const contextPath = $derived(kind === 'series' ? (item.folder ?? '') : (item.path ?? ''));
	const targetLabel = $derived(kind === 'series' ? 'serie' : 'película');
</script>

<details class="admin-card match-panel" id="corregir" open={open}>
	<summary class="button match-summary">Corregir ficha</summary>

	<div class="match-body">
		<p class="kicker">Corrección de ficha de {targetLabel}</p>
		<h2>{item.title}</h2>
		<p class="match-status"><span>Estado actual:</span> {status}</p>
		<p class="admin-item-path">
			<strong>{item.file_title}</strong><br />{contextPath}
		</p>

		<form
			class="match-search"
			method="POST"
			action={`?/${actions.candidates}`}
			use:enhance
		>
			<label>
				Título
				<input name="q" type="search" value={query} required />
			</label>
			<label>
				Año (opcional)
				<input name="year" type="text" inputmode="numeric" value={year} />
			</label>
			<button class="button" type="submit">Buscar en TMDB</button>
		</form>

		{#if search && candidates.length === 0}
			<p class="admin-note" role="status">Sin candidatos: prueba otro título o año</p>
		{/if}

		{#if candidates.length > 0}
			<ul class="match-candidates">
				{#each candidates as candidate, index (index)}
					<li class="match-candidate">
						{#if candidate.poster_url}
							<img
								class="match-poster"
								src={candidate.poster_url}
								alt={`Cartel de ${candidate.title}`}
								loading="lazy"
							/>
						{:else}
							<span class="match-poster match-poster--empty" aria-hidden="true"
								>{candidate.title.slice(0, 1)}</span
							>
						{/if}

						<div class="match-candidate-info">
							<p class="match-title">{candidate.title}</p>
							{#if candidate.original_title && candidate.original_title !== candidate.title}
								<p class="match-original"><em>{candidate.original_title}</em></p>
							{/if}
							<p class="match-meta">
								{candidate.year > 0 ? candidate.year : 'Año desconocido'}
							</p>
							{#if candidate.overview}
								<p class="match-overview">{truncateSynopsis(candidate.overview)}</p>
							{/if}
						</div>

						<form method="POST" action={`?/${actions.match}`} use:enhance>
							<input type="hidden" name="tmdb_id" value={candidate.tmdb_id} />
							<button
								class="button"
								type="submit"
								aria-label={`Es esta: ${candidate.title}`}>Es esta</button
							>
						</form>
					</li>
				{/each}
			</ul>
		{/if}

		<div class="match-actions">
			<form method="POST" action={`?/${actions.nomatch}`} use:enhance>
				<button class="button-danger" type="submit">No es ninguna</button>
			</form>
			<form method="POST" action={`?/${actions.automatic}`} use:enhance>
				<button class="button match-auto" type="submit">Volver a automático</button>
			</form>
		</div>
	</div>
</details>
