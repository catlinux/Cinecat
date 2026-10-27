import { render } from 'svelte/server';
import { describe, expect, it, vi } from 'vitest';
import type { Collection, MediaItem, Series } from '$lib/api';
import UserListPage from '$lib/components/UserListPage.svelte';
import { DEFAULT_PUBLIC_UI } from '$lib/settings';
import { load as loadFavorites } from './favorites/+page.server';
import { load as loadWatchlist } from './watchlist/+page.server';

function jsonResponse(body: unknown, status = 200): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

function listEvent(fetchImpl: typeof fetch, url: string) {
	return {
		locals: { sessionToken: 'token-123' },
		fetch: fetchImpl,
		url: new URL(url),
		parent: async () => ({ ui: DEFAULT_PUBLIC_UI })
	} as never;
}

const item: MediaItem = {
	id: 1,
	library_id: 1,
	library_kind: 'video',
	path: '/media/peli.mkv',
	title: 'Película',
	file_title: 'Película',
	year: 2020,
	kind: 'movie',
	season: null,
	episode: null,
	tmdb_id: null,
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
	still_url: null
};
const clip: MediaItem = { ...item, id: 2, library_kind: 'clips', title: 'Vídeo' };
const series: Series = {
	id: 3,
	title: 'Serie',
	file_title: 'Serie',
	year: 2021,
	poster_url: null,
	genres: [],
	season_count: 1,
	episode_count: 4,
	watched_count: 0,
	match_status: null
};
const collection: Collection = {
	id: 4,
	name: 'Colección',
	overview: null,
	poster_url: null,
	backdrop_url: null,
	item_count: 2,
	source: 'manual',
	edited: false
};

function payload(overrides: Record<string, unknown> = {}) {
	return {
		items: [item, clip],
		series: [series],
		collections: [collection],
		totals: { items: 2, series: 1, collections: 1 },
		limit: 48,
		offset: 0,
		...overrides
	};
}

describe('loads de favoritos y próximas a ver', () => {
	it('carga todos los bloques y separa películas y vídeos', async () => {
		const fetchMock = vi.fn(async (_input: RequestInfo | URL) => jsonResponse(payload()));
		const result = (await loadFavorites(
			listEvent(fetchMock as never, 'http://localhost/favorites')
		)) as any;

		expect(result.items).toHaveLength(2);
		expect(result.series).toHaveLength(1);
		expect(result.collections).toHaveLength(1);
		expect(result.totals).toEqual({ items: 2, series: 1, collections: 1 });
		expect(String(fetchMock.mock.calls[0][0])).toContain('/api/v1/lists/favorite?limit=48&offset=0');
	});

	it('pagina por bloque y calcula el último', async () => {
		const fetchMock = vi.fn(async (_input: RequestInfo | URL) =>
			jsonResponse(payload({ total: 120, totals: { items: 120, series: 0, collections: 0 } }))
		);
		const result = (await loadWatchlist(
			listEvent(fetchMock as never, 'http://localhost/watchlist?kind=items&page=2')
		)) as any;

		expect(result.page).toBe(2);
		expect(result.last).toBe(3);
		expect(String(fetchMock.mock.calls[0][0])).toContain('kind=items&limit=48&offset=48');
	});

	it('valida kind y page y tolera parámetros inválidos', async () => {
		const fetchMock = vi.fn(async (_input: RequestInfo | URL) => jsonResponse(payload()));
		const result = (await loadFavorites(
			listEvent(fetchMock as never, 'http://localhost/favorites?kind=raro&page=0')
		)) as any;
		expect(result.kind).toBeNull();
		expect(result.page).toBe(1);
		const requested = String(fetchMock.mock.calls[0][0]);
		expect(requested).not.toContain('kind=');
		expect(requested).toContain('offset=0');
	});

	it('redirige si la página supera el último bloque', async () => {
		const fetchMock = vi.fn(async () =>
			jsonResponse(payload({ total: 10, totals: { items: 10, series: 0, collections: 0 } }))
		);
		await expect(
			loadFavorites(listEvent(fetchMock as never, 'http://localhost/favorites?kind=items&page=9'))
		).rejects.toMatchObject({ status: 303, location: '/favorites?kind=items' });
	});

	it('renderiza las cuatro secciones y el estado vacío', () => {
		const base = {
			kind: null,
			page: 1,
			pageSize: 48,
			currentUrl: '/favorites',
			errorMessage: null,
			actionError: false,
			ui: DEFAULT_PUBLIC_UI
		};
		const full = render(UserListPage, {
			props: {
				list: 'favorite',
				data: {
					...base,
					items: [item, clip],
					series: [series],
					collections: [collection],
					totals: { items: 2, series: 1, collections: 1 }
				}
			}
		}).body;
		expect(full).toContain('Películas');
		expect(full).toContain('Vídeos');
		expect(full).toContain('Series');
		expect(full).toContain('Colecciones');
		expect(full).toContain('1 película');
		expect(full).toContain('1 vídeo');

		const empty = render(UserListPage, {
			props: {
				list: 'watchlist',
				data: {
					...base,
					items: [],
					series: [],
					collections: [],
					totals: { items: 0, series: 0, collections: 0 }
				}
			}
		}).body;
		expect(empty).toContain('No hay nada en próximas a ver');
		expect(empty).toContain('href="/movies"');
	});

	it('redirige al login con 401 y muestra error genérico con otros fallos', async () => {
		const unauthorized = vi.fn(async () => jsonResponse({ error: 'sesión' }, 401));
		await expect(
			loadFavorites(listEvent(unauthorized as never, 'http://localhost/favorites'))
		).rejects.toMatchObject({ status: 303, location: '/login' });

		const broken = vi.fn(async () => jsonResponse({ error: 'fallo' }, 500));
		const result = (await loadWatchlist(
			listEvent(broken as never, 'http://localhost/watchlist')
		)) as any;
		expect(result.errorMessage).toBeTruthy();
		expect(result.items).toEqual([]);
	});
});
