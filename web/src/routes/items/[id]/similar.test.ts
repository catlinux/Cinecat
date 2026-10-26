import { describe, expect, it, vi } from 'vitest';
import type { MediaItem, Series, SeriesDetail } from '$lib/api';
import { load as loadSeries } from '../../series/[id]/+page.server';
import { load as loadItem } from './+page.server';

function jsonResponse(body: unknown, status = 200): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

function makeItem(overrides: Partial<MediaItem> = {}): MediaItem {
	return {
		id: 7,
		library_id: 1,
		path: '/media/peliculas/El Padrino (1972).mkv',
		title: 'El Padrino',
		file_title: 'El Padrino (1972)',
		year: 1972,
		kind: 'movie',
		season: null,
		episode: null,
		tmdb_id: 238,
		overview: null,
		poster_url: null,
		poster_custom: false,
		original_title: null,
		imdb_id: null,
		tagline: null,
		genres: [],
		runtime_minutes: null,
		rating: null,
		directors: [],
		cast: [],
		backdrop_url: null,
		backdrop_custom: false,
		overview_language: null,
		match_status: null,
		match_score: null,
		watched: false,
		progress: null,
		collections: [],
		series_id: null,
		series_title: null,
		episode_title: null,
		still_url: null,
		...overrides
	};
}

const similarMovie = makeItem({ id: 8, title: 'El padrino II' });
const similarSeries: Series = {
	id: 9,
	title: 'Los Soprano',
	file_title: 'Los Soprano',
	year: 1999,
	poster_url: null,
	genres: ['Drama'],
	season_count: 6,
	episode_count: 86,
	watched_count: 0,
	match_status: 'auto'
};

const seriesDetail: SeriesDetail = {
	...similarSeries,
	id: 5,
	title: 'The Big Bang Theory',
	file_title: 'The Big Bang theory',
	folder: 'The Big Bang theory',
	original_title: null,
	tagline: null,
	overview: null,
	overview_language: 'es-ES',
	poster_url: null,
	backdrop_url: null,
	rating: null,
	directors: [],
	cast: [],
	tmdb_id: null,
	match_score: null,
	poster_custom: false,
	backdrop_custom: false,
	season_count: 1,
	episode_count: 1,
	watched_count: 0,
	seasons: [{ season: 1, name: 'Temporada 1', overview: null, poster_url: null, episode_count: 1, watched_count: 0 }],
	next_episode: null
};

function itemEvent(fetchImpl: typeof fetch) {
	return {
		params: { id: '7' },
		locals: { sessionToken: 'token-123' },
		fetch: fetchImpl,
		url: new URL('http://localhost/items/7'),
		parent: async () => ({ isAdmin: false })
	};
}

function seriesEvent(fetchImpl: typeof fetch) {
	return {
		params: { id: '5' },
		locals: { sessionToken: 'token-123' },
		fetch: fetchImpl,
		url: new URL('http://localhost/series/5')
	};
}

function itemFetch(similarResponse: () => Response) {
	return vi.fn(async (input: RequestInfo | URL) => {
		const url = new URL(String(input));
		if (url.pathname === '/api/v1/items/7') {
			return jsonResponse(makeItem());
		}
		if (url.pathname === '/api/v1/items/7/similar') {
			return similarResponse();
		}
		if (url.pathname === '/api/v1/items/7/file') {
			return jsonResponse({});
		}
		if (url.pathname === '/api/v1/items/7/play') {
			return jsonResponse({});
		}
		throw new Error(`URL inesperada: ${url}`);
	});
}

function seriesFetch(similarResponse: () => Response) {
	return vi.fn(async (input: RequestInfo | URL) => {
		const url = new URL(String(input));
		if (url.pathname === '/api/v1/series/5') {
			return jsonResponse(seriesDetail);
		}
		if (url.pathname === '/api/v1/series/5/seasons/1') {
			return jsonResponse({
				season: 1,
				name: 'Temporada 1',
				overview: null,
				poster_url: null,
				items: []
			});
		}
		if (url.pathname === '/api/v1/series/5/similar') {
			return similarResponse();
		}
		throw new Error(`URL inesperada: ${url}`);
	});
}

describe('similares de una película', () => {
	it('carga la lista sin bloquear el resto de la ficha', async () => {
		const fetchMock = itemFetch(() => jsonResponse({ items: [similarMovie] }));
		const result = (await loadItem(itemEvent(fetchMock) as never)) as {
			item: MediaItem;
			similarItems: Promise<MediaItem[] | null>;
		};

		expect(result.item.title).toBe('El Padrino');
		expect(await result.similarItems).toEqual([similarMovie]);
		const similarCall = fetchMock.mock.calls.find(([input]) =>
			new URL(String(input)).pathname.endsWith('/similar')
		);
		expect(new URL(String(similarCall?.[0])).searchParams.get('limit')).toBe('12');
	});

	it('devuelve una lista vacía si no hay similares', async () => {
		const fetchMock = itemFetch(() => jsonResponse({ items: [] }));
		const result = (await loadItem(itemEvent(fetchMock) as never)) as {
			similarItems: Promise<MediaItem[] | null>;
		};

		expect(await result.similarItems).toEqual([]);
	});

	it('oculta la fila si la API de similares falla', async () => {
		const fetchMock = itemFetch(() => jsonResponse({ error: 'fallo' }, 500));
		const result = (await loadItem(itemEvent(fetchMock) as never)) as {
			item: MediaItem;
			similarItems: Promise<MediaItem[] | null>;
		};

		expect(await result.similarItems).toBeNull();
		expect(result.item.title).toBe('El Padrino');
	});
});

describe('similares de una serie', () => {
	it('carga la lista sin bloquear el resto de la ficha', async () => {
		const fetchMock = seriesFetch(() => jsonResponse({ series: [similarSeries] }));
		const result = (await loadSeries(seriesEvent(fetchMock) as never)) as {
			series: SeriesDetail;
			similarSeries: Promise<Series[]>;
		};

		expect(result.series.title).toBe('The Big Bang Theory');
		expect(await result.similarSeries).toEqual([similarSeries]);
	});

	it('devuelve una lista vacía si no hay similares', async () => {
		const fetchMock = seriesFetch(() => jsonResponse({ series: [] }));
		const result = (await loadSeries(seriesEvent(fetchMock) as never)) as {
			similarSeries: Promise<Series[]>;
		};

		expect(await result.similarSeries).toEqual([]);
	});

	it('oculta la fila si la API de similares falla', async () => {
		const fetchMock = seriesFetch(() => jsonResponse({ error: 'fallo' }, 500));
		const result = (await loadSeries(seriesEvent(fetchMock) as never)) as {
			series: SeriesDetail;
			similarSeries: Promise<Series[]>;
		};

		expect(await result.similarSeries).toEqual([]);
		expect(result.series.title).toBe('The Big Bang Theory');
	});
});
