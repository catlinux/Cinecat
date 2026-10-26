import type { GenreCount } from './api';

export const PAGE_SIZE = 48;

export const SEARCH_FIELDS = ['all', 'title', 'cast', 'director'] as const;
export type SearchField = (typeof SEARCH_FIELDS)[number];

export const MOVIE_SORTS = ['title', 'year', 'added'] as const;
export type MovieSort = (typeof MOVIE_SORTS)[number];

export type WatchedFilter = 'true' | 'false' | null;

export interface MovieFilters {
	q: string;
	field: SearchField;
	letter: string | null;
	genre: string | null;
	sort: MovieSort;
	watched: WatchedFilter;
	unmatched: boolean;
}

export type LibraryFilterState = Omit<MovieFilters, 'field' | 'watched'> & {
	field?: SearchField;
	watched?: WatchedFilter;
};

export interface FilterChip {
	key: string;
	label: string;
	href: string;
}

export function readMovieFilters(url: URL, isAdmin: boolean): MovieFilters {
	const q = (url.searchParams.get('q') ?? '').trim();
	const fieldRaw = (url.searchParams.get('field') ?? '').trim();
	const field = (SEARCH_FIELDS as readonly string[]).includes(fieldRaw)
		? (fieldRaw as SearchField)
		: 'all';

	const letterRaw = (url.searchParams.get('letter') ?? '').trim().toUpperCase();
	const letter = letterRaw === '#' || /^[A-Z]$/.test(letterRaw) ? letterRaw : null;

	const readWatched = (url.searchParams.get('watched') ?? '').trim().toLowerCase();
	const watched: WatchedFilter =
		readWatched === 'true' || readWatched === 'false'
			? (readWatched as 'true' | 'false')
			: null;

	const rawSort = (url.searchParams.get('sort') ?? '').trim();
	const sort = (MOVIE_SORTS as readonly string[]).includes(rawSort)
		? (rawSort as MovieSort)
		: 'title';

	const unmatchedRaw = (url.searchParams.get('unmatched') ?? '').trim();
	const unmatched = isAdmin && (unmatchedRaw === '1' || unmatchedRaw === 'true');

	const genreRaw = (url.searchParams.get('genre') ?? '').trim();

	return {
		q,
		field,
		letter,
		genre: genreRaw || null,
		sort,
		watched,
		unmatched
	};
}

export function movieApiQuery(filters: MovieFilters, page: number): URLSearchParams {
	const params = new URLSearchParams();
	params.set('kind', 'movie');
	params.set('limit', String(PAGE_SIZE));
	params.set('offset', String((Math.max(1, page) - 1) * PAGE_SIZE));
	if (filters.q) {
		params.set('q', filters.q);
		params.set('field', filters.field);
	}
	if (filters.letter) {
		params.set('letter', filters.letter);
	}
	if (filters.genre) {
		params.set('genre', filters.genre);
	}
	params.set('sort', filters.sort);
	if (filters.watched) {
		params.set('watched', filters.watched);
	}
	if (filters.unmatched) {
		params.set('unmatched', 'true');
	}
	return params;
}

export function mutateHref(
	currentUrl: string,
	changes: Record<string, string | null>
): string {
	const url = new URL(currentUrl, 'http://cinecat.local');
	for (const [key, value] of Object.entries(changes)) {
		if (value === null) {
			url.searchParams.delete(key);
		} else {
			url.searchParams.set(key, value);
		}
	}
	url.searchParams.delete('page');
	return `${url.pathname}${url.search}`;
}

export function filterChips(filters: MovieFilters, currentUrl: string): FilterChip[] {
	const chips: FilterChip[] = [];
	if (filters.q) {
		chips.push({
			key: 'q',
			label: `Búsqueda: “${filters.q}”`,
			href: mutateHref(currentUrl, { q: null, field: null })
		});
	}
	if (filters.letter) {
		chips.push({
			key: 'letter',
			label: `Letra: ${filters.letter}`,
			href: mutateHref(currentUrl, { letter: null })
		});
	}
	if (filters.genre) {
		chips.push({
			key: 'genre',
			label: `Género: ${filters.genre}`,
			href: mutateHref(currentUrl, { genre: null })
		});
	}
	if (filters.watched) {
		chips.push({
			key: 'watched',
			label: filters.watched === 'true' ? 'Vistas' : 'Sin ver',
			href: mutateHref(currentUrl, { watched: null })
		});
	}
	if (filters.unmatched) {
		chips.push({
			key: 'unmatched',
			label: 'Sin identificar',
			href: mutateHref(currentUrl, { unmatched: null })
		});
	}
	return chips;
}

export function clearFiltersHref(currentUrl: string): string {
	return mutateHref(currentUrl, {
		q: null,
		field: null,
		letter: null,
		genre: null,
		watched: null,
		unmatched: null
	});
}

export function genreOptions(genres: GenreCount[]): GenreCount[] {
	return [...genres].sort((a, b) => a.name.localeCompare(b.name, 'es'));
}
