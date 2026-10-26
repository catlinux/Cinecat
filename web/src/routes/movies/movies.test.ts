import { render } from 'svelte/server';
import { describe, expect, it, vi } from 'vitest';
import type { GenreCount, MediaItem } from '$lib/api';
import type { MovieFilters } from '$lib/movies';
import { DEFAULT_PUBLIC_UI } from '$lib/settings';
import { webVersion } from '$lib/version';
import MoviesPage from './+page.svelte';
import { load } from './+page.server';

function jsonResponse(body: unknown, status = 200): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

const item: MediaItem = {
	id: 7,
	library_id: 1,
	path: '/media/peliculas/El Padrino (1972).mkv',
	title: 'El Padrino',
	file_title: 'El Padrino',
	year: 1972,
	kind: 'movie',
	season: null,
	episode: null,
	tmdb_id: 238,
	overview: 'Una familia muy unida.',
	poster_url: null,
	poster_custom: false,
	original_title: 'The Godfather',
	imdb_id: 'tt0068646',
	tagline: null,
	genres: ['Drama', 'Crimen'],
	runtime_minutes: 175,
	rating: 8.7,
	directors: ['Francis Ford Coppola'],
	cast: ['Marlon Brando', 'Al Pacino'],
	backdrop_url: null,
	backdrop_custom: false,
	overview_language: 'es-ES',
	match_status: 'auto',
	match_score: 1,
	watched: false,
	progress: null,
	collections: [],
	series_id: null,
	series_title: null,
	episode_title: null,
	still_url: null
};

const filters: MovieFilters = {
	q: '',
	field: 'all',
	letter: null,
	genre: null,
	sort: 'title',
	watched: null,
	unmatched: false
};

const genres: GenreCount[] = [
	{ name: 'Crimen', count: 2 },
	{ name: 'Drama', count: 8 }
];

function event(fetchImpl: typeof fetch, path: string, isAdmin = false) {
	return {
		locals: { sessionToken: 'token-123' },
		fetch: fetchImpl,
		url: new URL(`http://localhost${path}`),
		parent: async () => ({ isAdmin })
	};
}

function requestedUrls(fetchImpl: ReturnType<typeof vi.fn>): URL[] {
	return fetchImpl.mock.calls.map((call) => new URL(String(call[0])));
}

describe('carga de /movies', () => {
	it('traduce filtros y página a la petición de la API', async () => {
		const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
			const url = new URL(String(input));
			if (url.pathname === '/api/v1/items') {
				return jsonResponse({ items: [item], total: 120, limit: 48, offset: 48 });
			}
			if (url.pathname === '/api/v1/genres') {
				return jsonResponse({ genres });
			}
			throw new Error(`URL inesperada: ${url}`);
		});

		const result = (await load(
			event(
				fetchMock,
				'/movies?page=2&q=matrix&field=cast&letter=M&genre=Drama&watched=true&sort=year',
				false
			) as never
		)) as {
			page: number;
			last: number;
			filterChips: Array<{ key: string }>;
			filters: MovieFilters;
		};

		const urls = requestedUrls(fetchMock);
		const itemsUrl = urls.find((url) => url.pathname === '/api/v1/items');
		expect(itemsUrl?.searchParams.get('kind')).toBe('movie');
		expect(itemsUrl?.searchParams.get('limit')).toBe('48');
		expect(itemsUrl?.searchParams.get('offset')).toBe('48');
		expect(itemsUrl?.searchParams.get('q')).toBe('matrix');
		expect(itemsUrl?.searchParams.get('field')).toBe('cast');
		expect(itemsUrl?.searchParams.get('letter')).toBe('M');
		expect(itemsUrl?.searchParams.get('genre')).toBe('Drama');
		expect(itemsUrl?.searchParams.get('watched')).toBe('true');
		expect(itemsUrl?.searchParams.get('sort')).toBe('year');
		expect(itemsUrl?.searchParams.get('unmatched')).toBeNull();
		expect(urls.some((url) => url.pathname === '/api/v1/genres')).toBe(true);
		expect(result.page).toBe(2);
		expect(result.last).toBe(3);
		expect(result.filters.q).toBe('matrix');
		expect(result.filterChips.map((chip) => chip.key)).toEqual([
			'q',
			'letter',
			'genre',
			'watched'
		]);
	});

	it('solo envía unmatched si el usuario es administrador', async () => {
		const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
			const url = new URL(String(input));
			if (url.pathname === '/api/v1/items' && url.searchParams.get('limit') === '1') {
				return jsonResponse({ items: [], total: 4, limit: 1, offset: 0 });
			}
			if (url.pathname === '/api/v1/items') {
				return jsonResponse({ items: [item], total: 4, limit: 48, offset: 0 });
			}
			if (url.pathname === '/api/v1/genres') {
				return jsonResponse({ genres });
			}
			throw new Error(`URL inesperada: ${url}`);
		});

		const result = (await load(
			event(fetchMock, '/movies?unmatched=1&sort=added', true) as never
		)) as { unmatchedCount: number; filters: MovieFilters };

		const urls = requestedUrls(fetchMock);
		const listUrl = urls.find(
			(url) => url.pathname === '/api/v1/items' && url.searchParams.get('limit') === '48'
		);
		const countUrl = urls.find(
			(url) => url.pathname === '/api/v1/items' && url.searchParams.get('limit') === '1'
		);
		expect(listUrl?.searchParams.get('unmatched')).toBe('true');
		expect(listUrl?.searchParams.get('sort')).toBe('added');
		expect(countUrl?.searchParams.get('unmatched')).toBe('true');
		expect(result.unmatchedCount).toBe(4);
		expect(result.filters.unmatched).toBe(true);
	});

	it('ignora unmatched para usuarios que no son admin', async () => {
		const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
			const url = new URL(String(input));
			if (url.pathname === '/api/v1/items') {
				return jsonResponse({ items: [], total: 0, limit: 48, offset: 0 });
			}
			if (url.pathname === '/api/v1/genres') {
				return jsonResponse({ genres: [] });
			}
			throw new Error(`URL inesperada: ${url}`);
		});

		const result = (await load(event(fetchMock, '/movies?unmatched=1', false) as never)) as {
			filters: MovieFilters;
		};

		const urls = requestedUrls(fetchMock);
		expect(result.filters.unmatched).toBe(false);
		expect(urls.every((url) => url.searchParams.get('unmatched') === null)).toBe(true);
	});

	it('redirige a la última página si se pasa de rango', async () => {
		const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
			const url = new URL(String(input));
			if (url.pathname === '/api/v1/items') {
				return jsonResponse({ items: [], total: 10, limit: 48, offset: 0 });
			}
			throw new Error(`URL inesperada: ${url}`);
		});

		await expect(
			load(event(fetchMock, '/movies?q=algo&page=99', false) as never)
		).rejects.toMatchObject({
			status: 303,
			location: '/movies?q=algo&page=1'
		});
	});

	it('normaliza una página inválida a la primera', async () => {
		const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
			const url = new URL(String(input));
			if (url.pathname === '/api/v1/items') {
				return jsonResponse({ items: [item], total: 1, limit: 48, offset: 0 });
			}
			if (url.pathname === '/api/v1/genres') {
				return jsonResponse({ genres: [] });
			}
			throw new Error(`URL inesperada: ${url}`);
		});

		const result = (await load(event(fetchMock, '/movies?page=abc', false) as never)) as {
			page: number;
		};

		expect(result.page).toBe(1);
		const listUrl = requestedUrls(fetchMock).find((url) => url.pathname === '/api/v1/items');
		expect(listUrl?.searchParams.get('offset')).toBe('0');
	});

	it('redirige al login con 401 y devuelve mensaje con otros errores', async () => {
		const unauthorized = vi.fn().mockResolvedValue(jsonResponse({ error: 'sesión' }, 401));
		await expect(
			load(event(unauthorized, '/movies', false) as never)
		).rejects.toMatchObject({
			status: 303,
			location: '/login'
		});

		const broken = vi.fn().mockResolvedValue(jsonResponse({ error: 'fallo' }, 500));
		const result = (await load(event(broken, '/movies', false) as never)) as {
			items: MediaItem[];
			errorMessage: string | null;
		};
		expect(result.items).toEqual([]);
		expect(result.errorMessage).toBeTruthy();
	});
});

function pageData(overrides: Record<string, unknown> = {}) {
	return {
		hasSession: true,
		isAdmin: false,
		isHome: false,
		activeCategory: 'movies' as const,
		serverVersion: null,
		serverCommit: null,
		webVersion,
		ui: DEFAULT_PUBLIC_UI,
		items: [item],
		total: 100,
		page: 2,
		last: 3,
		pageSize: 48,
		filters,
		genres,
		unmatchedCount: 0,
		filterChips: [],
		clearHref: '/movies',
		currentUrl: '/movies?page=2',
		errorMessage: null,
		...overrides
	};
}

describe('página /movies', () => {
	it('muestra resultados, paginación y filtros activos', () => {
		const { body } = render(MoviesPage, {
			props: {
				data: pageData({
					filterChips: [
						{ key: 'letter', label: 'Letra: P', href: '/movies' },
						{ key: 'genre', label: 'Género: Drama', href: '/movies?letter=P' }
					]
				})
			}
		});

		expect(body).toContain('El Padrino');
		expect(body).toContain('100 títulos');
		expect(body).toContain('Letra: P');
		expect(body).toContain('Género: Drama');
		expect(body).toContain('Limpiar filtros');
		expect(body).toContain('aria-label="Paginación"');
		expect(body).toContain('aria-current="page"');
		expect(body).toContain('Siguiente');
	});

	it('distingue una búsqueda sin resultados de una biblioteca vacía', () => {
		const filtered = render(MoviesPage, {
			props: {
				data: pageData({
					items: [],
					total: 0,
					page: 1,
					last: 1,
					filterChips: [{ key: 'q', label: 'Búsqueda: “nada”', href: '/movies' }]
				})
			}
		}).body;
		expect(filtered).toContain('Ninguna película coincide');
		expect(filtered).toContain('Limpiar filtros');

		const empty = render(MoviesPage, {
			props: {
				data: pageData({ items: [], total: 0, page: 1, last: 1, filterChips: [] })
			}
		}).body;
		expect(empty).toContain('Todavía no hay películas en cartel');
	});

	it('muestra el error sin romper la página', () => {
		const { body } = render(MoviesPage, {
			props: {
				data: pageData({
					items: [],
					total: 0,
					page: 1,
					last: 1,
					errorMessage: 'No se pudo cargar la cartelera.'
				})
			}
		});

		expect(body).toContain('No se pudo cargar la cartelera.');
		expect(body).toContain('role="alert"');
	});

	it('muestra el enlace de sin identificar solo al admin', () => {
		const admin = render(MoviesPage, {
			props: { data: pageData({ isAdmin: true, unmatchedCount: 4 }) }
		}).body;
		expect(admin).toContain('Sin identificar (4)');

		const user = render(MoviesPage, { props: { data: pageData() } }).body;
		expect(user).not.toContain('Sin identificar (');
	});
});
