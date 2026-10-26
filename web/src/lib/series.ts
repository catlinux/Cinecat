import type { MediaItem, Series, SeriesDetail, SeriesSort } from './api';
import { mutateHref, type FilterChip } from './movies';

export const SERIES_PAGE_SIZE = 48;

export const SERIES_SORTS: Array<{ value: SeriesSort; label: string }> = [
	{ value: 'title', label: 'Título' },
	{ value: 'year', label: 'Año' },
	{ value: 'added', label: 'Recientes' }
];

export interface SeriesFilters {
	q: string;
	letter: string | null;
	genre: string | null;
	sort: SeriesSort;
	unmatched: boolean;
}

export interface SeriesPlayChoice {
	item: MediaItem;
	label: string;
}

export function readSeriesFilters(url: URL, isAdmin: boolean): SeriesFilters {
	const q = (url.searchParams.get('q') ?? '').trim();
	const letterRaw = (url.searchParams.get('letter') ?? '').trim().toUpperCase();
	const letter = letterRaw === '#' || /^[A-Z]$/.test(letterRaw) ? letterRaw : null;
	const genreRaw = (url.searchParams.get('genre') ?? '').trim();
	const rawSort = (url.searchParams.get('sort') ?? '').trim();
	const sort: SeriesSort = SERIES_SORTS.some((option) => option.value === rawSort)
		? (rawSort as SeriesSort)
		: 'title';
	const unmatchedRaw = (url.searchParams.get('unmatched') ?? '').trim();
	const unmatched = isAdmin && (unmatchedRaw === '1' || unmatchedRaw === 'true');

	return { q, letter, genre: genreRaw || null, sort, unmatched };
}

export function seriesApiQuery(filters: SeriesFilters, page: number): URLSearchParams {
	const params = new URLSearchParams();
	params.set('limit', String(SERIES_PAGE_SIZE));
	params.set('offset', String((Math.max(1, page) - 1) * SERIES_PAGE_SIZE));
	if (filters.q) {
		params.set('q', filters.q);
	}
	if (filters.letter) {
		params.set('letter', filters.letter);
	}
	if (filters.genre) {
		params.set('genre', filters.genre);
	}
	params.set('sort', filters.sort);
	if (filters.unmatched) {
		params.set('unmatched', 'true');
	}
	return params;
}

export function seriesFilterChips(filters: SeriesFilters, currentUrl: string): FilterChip[] {
	const chips: FilterChip[] = [];
	if (filters.q) {
		chips.push({
			key: 'q',
			label: `Búsqueda: “${filters.q}”`,
			href: mutateHref(currentUrl, { q: null })
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
	if (filters.unmatched) {
		chips.push({
			key: 'unmatched',
			label: 'Sin identificar',
			href: mutateHref(currentUrl, { unmatched: null })
		});
	}
	return chips;
}

export function clearSeriesFiltersHref(currentUrl: string): string {
	return mutateHref(currentUrl, {
		q: null,
		letter: null,
		genre: null,
		unmatched: null
	});
}

export function formatSeriesCounts(seasons: number, episodes: number): string {
	const safeSeasons = Number.isFinite(seasons) && seasons > 0 ? Math.floor(seasons) : 0;
	const safeEpisodes = Number.isFinite(episodes) && episodes > 0 ? Math.floor(episodes) : 0;
	return `${safeSeasons} ${safeSeasons === 1 ? 'temporada' : 'temporadas'} · ${safeEpisodes} ${
		safeEpisodes === 1 ? 'episodio' : 'episodios'
	}`;
}

export function formatSeriesProgress(watched: number, total: number): string {
	const safeTotal = Number.isFinite(total) && total > 0 ? Math.floor(total) : 0;
	const safeWatched =
		Number.isFinite(watched) && watched > 0 ? Math.min(safeTotal, Math.floor(watched)) : 0;
	if (safeTotal === 0) {
		return '';
	}
	if (safeWatched >= safeTotal) {
		return 'Vista';
	}
	return `${safeWatched}/${safeTotal} vistos`;
}

export function seriesWatched(series: Pick<Series, 'watched_count' | 'episode_count'>): boolean {
	return series.episode_count > 0 && series.watched_count >= series.episode_count;
}

export function seasonWatched(season: Pick<SeriesDetail['seasons'][number], 'watched_count' | 'episode_count'>): boolean {
	return season.episode_count > 0 && season.watched_count >= season.episode_count;
}

export function formatEpisodeCode(item: Pick<MediaItem, 'season' | 'episode'>): string {
	if (item.season !== null && item.episode !== null) {
		return `T${item.season} E${item.episode}`;
	}
	if (item.episode !== null) {
		return `Episodio ${item.episode}`;
	}
	return '';
}

export function formatEpisodeHeading(item: MediaItem): string {
	const code = item.episode !== null ? `E${item.episode}` : '';
	const title = item.episode_title?.trim() || item.title.trim();
	if (code && title) {
		return `${code} · ${title}`;
	}
	return code || title || 'Episodio';
}

export function selectSeriesSeason(
	seasons: Array<{ season: number }>,
	nextEpisode: Pick<MediaItem, 'season'> | null,
	raw: string | null
): number | null {
	if (seasons.length === 0) {
		return null;
	}
	const values = seasons.map((item) => item.season);
	const rawSeason = (raw ?? '').trim();
	if (rawSeason !== '') {
		const requested = Number.parseInt(rawSeason, 10);
		return Number.isInteger(requested) && values.includes(requested)
			? requested
			: values[0];
	}
	if (nextEpisode?.season !== null && nextEpisode?.season !== undefined &&
		values.includes(nextEpisode.season)) {
		return nextEpisode.season;
	}
	return values[0];
}

export function seriesPlayChoice(
	series: Pick<SeriesDetail, 'episode_count' | 'watched_count'>,
	nextEpisode: MediaItem | null,
	firstEpisode: MediaItem | null
): SeriesPlayChoice | null {
	if (series.episode_count <= 0) {
		return null;
	}
	const allWatched = series.watched_count >= series.episode_count;
	const item = allWatched ? firstEpisode ?? nextEpisode : nextEpisode ?? firstEpisode;
	if (!item) {
		return null;
	}
	if (allWatched) {
		return { item, label: 'Volver a ver' };
	}
	const code = formatEpisodeCode(item);
	const hasStarted = series.watched_count > 0 || item.progress !== null;
	if (!hasStarted) {
		return { item, label: code ? `Empezar · ${code}` : 'Empezar' };
	}
	const title = item.title.trim();
	return { item, label: `Continuar · ${code}${title ? ` · ${title}` : ''}` };
}
