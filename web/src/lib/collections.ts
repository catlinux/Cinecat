import type { CollectionSort, MediaItem } from './api';

export const COLLECTION_SORTS: Array<{ value: CollectionSort; label: string }> = [
	{ value: 'name', label: 'Nombre' },
	{ value: 'added', label: 'Añadidas' },
	{ value: 'size', label: 'Tamaño' }
];

export interface CollectionFilters {
	q: string;
	sort: CollectionSort;
	page: number;
}

export function readCollectionFilters(url: URL): CollectionFilters {
	const q = (url.searchParams.get('q') ?? '').trim();
	const rawSort = (url.searchParams.get('sort') ?? '').trim();
	const sort: CollectionSort =
		rawSort === 'added' || rawSort === 'size' || rawSort === 'name' ? rawSort : 'name';
	const rawPage = Number.parseInt((url.searchParams.get('page') ?? '').trim(), 10);
	const page = Number.isFinite(rawPage) && rawPage >= 1 ? rawPage : 1;
	return { q, sort, page };
}

export function collectionApiQuery(
	filters: CollectionFilters,
	page: number,
	pageSize: number
): URLSearchParams {
	const params = new URLSearchParams();
	params.set('limit', String(pageSize));
	params.set('offset', String((Math.max(1, page) - 1) * pageSize));
	if (filters.q) {
		params.set('q', filters.q);
	}
	params.set('sort', filters.sort);
	return params;
}

export function collectionFiltersActive(filters: CollectionFilters): boolean {
	return filters.q !== '' || filters.sort !== 'name';
}

export function formatCollectionCount(count: number): string {
	const value = Number.isFinite(count) && count > 0 ? Math.floor(count) : 0;
	return `${value} ${value === 1 ? 'película' : 'películas'}`;
}

export function formatViewedCount(viewed: number, total: number): string {
	const safeTotal = Number.isFinite(total) && total > 0 ? Math.floor(total) : 0;
	const safeViewed =
		Number.isFinite(viewed) && viewed > 0 ? Math.min(safeTotal, Math.floor(viewed)) : 0;
	return `${safeViewed} de ${safeTotal} ${safeTotal === 1 ? 'vista' : 'vistas'}`;
}

export function viewedCount(items: MediaItem[]): number {
	return items.filter((item) => item.watched).length;
}

export function firstUnwatched(items: MediaItem[]): MediaItem | null {
	if (items.length === 0) {
		return null;
	}
	return items.find((item) => !item.watched) ?? items[0];
}

export function hasUnwatched(items: MediaItem[]): boolean {
	return items.some((item) => !item.watched);
}

export function moveItem(
	ids: number[],
	itemID: number,
	direction: 'up' | 'down'
): number[] {
	const result = [...ids];
	const index = result.indexOf(itemID);
	if (index < 0) {
		return result;
	}
	const target = direction === 'up' ? index - 1 : index + 1;
	if (target < 0 || target >= result.length) {
		return result;
	}
	[result[index], result[target]] = [result[target], result[index]];
	return result;
}

export function tmdbCollectionURL(tmdbID: number | null | undefined): string | null {
	if (tmdbID === null || tmdbID === undefined || !Number.isInteger(tmdbID) || tmdbID <= 0) {
		return null;
	}
	return `https://www.themoviedb.org/collection/${tmdbID}`;
}

export function collectionIdentificationLabel(collection: {
	tmdb_collection_id?: number | null;
}): string {
	const id = collection.tmdb_collection_id;
	if (id === null || id === undefined || !Number.isInteger(id) || id <= 0) {
		return 'Sin identificar en TMDB';
	}
	return `Identificada en TMDB con el id ${id}`;
}

export function formatMissingYear(year: number): string {
	return Number.isFinite(year) && year > 0 ? String(Math.floor(year)) : 'Año desconocido';
}
