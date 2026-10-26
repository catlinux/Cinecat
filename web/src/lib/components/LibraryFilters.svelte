<script lang="ts">
	import type { GenreCount } from '$lib/api';
	import { genreOptions, mutateHref, type MovieFilters } from '$lib/movies';

	let {
		filters,
		genres,
		isAdmin,
		unmatchedCount,
		currentUrl
	}: {
		filters: MovieFilters;
		genres: GenreCount[];
		isAdmin: boolean;
		unmatchedCount: number;
		currentUrl: string;
	} = $props();

	const orderedGenres = $derived(genreOptions(genres));
	const letters = ['#', ...'ABCDEFGHIJKLMNOPQRSTUVWXYZ'];
	const letterLinks = $derived(
		letters.map((letter) => ({
			letter,
			active: filters.letter === letter,
			href: mutateHref(currentUrl, {
				letter: filters.letter === letter ? null : letter
			})
		}))
	);
	const unmatchedHref = $derived(
		mutateHref(currentUrl, { unmatched: filters.unmatched ? null : '1' })
	);
	const currentGenreMissing = $derived(
		Boolean(filters.genre) && !orderedGenres.some((genre) => genre.name === filters.genre)
	);
</script>

<div class="movies-toolbar">
	<form class="filters-form" method="GET" action="/movies">
		{#if filters.letter}
			<input type="hidden" name="letter" value={filters.letter} />
		{/if}
		{#if filters.unmatched}
			<input type="hidden" name="unmatched" value="1" />
		{/if}

		<label class="filter-field filter-search">
			<span>Buscar</span>
			<input
				type="search"
				name="q"
				value={filters.q}
				placeholder="Título, actor, directora…"
				autocomplete="off"
			/>
		</label>

		<label class="filter-field">
			<span>Buscar en</span>
			<select name="field">
				<option value="all" selected={filters.field === 'all'}>Todo</option>
				<option value="title" selected={filters.field === 'title'}>Título</option>
				<option value="cast" selected={filters.field === 'cast'}>Reparto</option>
				<option value="director" selected={filters.field === 'director'}>Dirección</option>
			</select>
		</label>

		<label class="filter-field">
			<span>Género</span>
			<select name="genre">
				<option value="" selected={!filters.genre}>Todos</option>
				{#if currentGenreMissing}
					<option value={filters.genre ?? ''} selected>{filters.genre}</option>
				{/if}
				{#each orderedGenres as genre (genre.name)}
					<option value={genre.name} selected={filters.genre === genre.name}>
						{genre.name} ({genre.count})
					</option>
				{/each}
			</select>
		</label>

		<label class="filter-field">
			<span>Orden</span>
			<select name="sort">
				<option value="title" selected={filters.sort === 'title'}>Título</option>
				<option value="year" selected={filters.sort === 'year'}>Año</option>
				<option value="added" selected={filters.sort === 'added'}>Recientes</option>
			</select>
		</label>

		<label class="filter-field">
			<span>Visionado</span>
			<select name="watched">
				<option value="" selected={!filters.watched}>Todas</option>
				<option value="false" selected={filters.watched === 'false'}>Sin ver</option>
				<option value="true" selected={filters.watched === 'true'}>Vistas</option>
			</select>
		</label>

		<button type="submit" class="button filter-submit">Aplicar</button>
	</form>

	<nav class="letter-strip" aria-label="Filtrar por letra">
		<span class="filter-inline-label">Letra</span>
		{#each letterLinks as link (link.letter)}
			<a
				href={link.href}
				class:letter-active={link.active}
				aria-current={link.active ? 'page' : undefined}>{link.letter}</a
			>
		{/each}
	</nav>

	{#if isAdmin}
		<a
			class="unmatched-link"
			class:unmatched-active={filters.unmatched}
			href={unmatchedHref}
			aria-current={filters.unmatched ? 'page' : undefined}
		>
			Sin identificar ({unmatchedCount})
		</a>
	{/if}
</div>
