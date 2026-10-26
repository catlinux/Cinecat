import { render } from 'svelte/server';
import { describe, expect, it, vi } from 'vitest';
import type {
	MediaItem,
	Series,
	SeriesDetail,
	SeriesSeasonDetail
} from '$lib/api';
import { DEFAULT_PUBLIC_UI } from '$lib/settings';
import SeriesPage from './[id]/+page.svelte';
import { actions, load as loadDetail } from './[id]/+page.server';
import SeriesListPage from './+page.svelte';
import { load as loadList } from './+page.server';

function jsonResponse(body: unknown, status = 200): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

function emptyResponse(status = 204): Response {
	return new Response(null, { status });
}

function episode(
	id: number,
	season: number,
	episodeNumber: number,
	title: string,
	options: Partial<MediaItem> = {}
): MediaItem {
	return {
		id,
		library_id: 1,
		path: `/media/series/${id}.mkv`,
		title,
		file_title: `file-${id}`,
		year: 2010,
		kind: 'episode',
		season,
		episode: episodeNumber,
		tmdb_id: null,
		overview: null,
		poster_url: null,
		poster_custom: false,
		original_title: null,
		imdb_id: null,
		tagline: null,
		genres: [],
		runtime_minutes: 42,
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
		series_id: 5,
		series_title: 'The Big Bang Theory',
		episode_title: title,
		still_url: null,
		...options
	};
}

const seriesItem: Series = {
	id: 5,
	title: 'The Big Bang Theory',
	file_title: 'The Big Bang theory',
	year: 2007,
	poster_url: 'https://images.example/serie.jpg',
	genres: ['Comedia'],
	season_count: 2,
	episode_count: 5,
	watched_count: 2,
	match_status: 'auto'
};

const season1 = {
	season: 1,
	name: 'Temporada 1',
	overview: 'La primera temporada.',
	poster_url: 'https://images.example/t1.jpg',
	episode_count: 2,
	watched_count: 1
};
const season2 = {
	season: 2,
	name: 'Temporada 2',
	overview: null,
	poster_url: null,
	episode_count: 3,
	watched_count: 1
};

const firstEpisode = episode(11, 1, 1, 'Piloto', {
	overview: 'Una sinopsis larga del primer episodio.',
	still_url: 'https://images.example/still-11.jpg',
	watched: true
});
const secondEpisode = episode(12, 1, 2, 'Segundo', {
	overview: 'Sinopsis del segundo.',
	progress: {
		position_seconds: 600,
		duration_seconds: 2400,
		updated_at: '2026-10-26T10:00:00Z'
	}
});
const seasonDetail: SeriesSeasonDetail = {
	season: 1,
	name: 'Temporada 1',
	overview: 'La primera temporada.',
	poster_url: null,
	items: [firstEpisode, secondEpisode]
};

const detail: SeriesDetail = {
	...seriesItem,
	folder: 'The Big Bang theory',
	original_title: 'The Big Bang Theory (original)',
	tagline: 'Una comedia de científicos.',
	overview: 'Sinopsis completa de la serie.',
	overview_language: 'es-ES',
	poster_url: 'https://images.example/serie.jpg',
	backdrop_url: 'https://images.example/fondo.jpg',
	genres: ['Comedia'],
	rating: 8.2,
	directors: ['Chuck Lorre'],
	cast: Array.from({ length: 13 }, (_, index) => `Actor ${index + 1}`),
	tmdb_id: 1418,
	match_status: 'auto',
	match_score: 0.99,
	poster_custom: false,
	backdrop_custom: false,
	season_count: 2,
	episode_count: 3,
	watched_count: 1,
	seasons: [season1, season2],
	next_episode: secondEpisode
};

function listData(overrides: Record<string, unknown> = {}) {
	return {
		series: [seriesItem],
		total: 1,
		page: 1,
		last: 1,
		pageSize: 48,
		filters: { q: '', letter: null, genre: null, sort: 'title', unmatched: false },
		genres: [{ name: 'Comedia', count: 1 }],
		isAdmin: false,
		unmatchedCount: 0,
		filterChips: [],
		clearHref: '/series',
		currentUrl: '/series',
		errorMessage: null,
		...overrides
	};
}

function detailData(overrides: Record<string, unknown> = {}) {
	return {
		series: detail,
		season: seasonDetail,
		selectedSeason: 1,
		firstEpisode,
		...overrides
	};
}

function listEvent(
	fetchImpl: typeof fetch,
	path = '/series',
	isAdmin = true,
	ui = DEFAULT_PUBLIC_UI
) {
	return {
		locals: { sessionToken: 'token-123' },
		fetch: fetchImpl,
		url: new URL(`http://localhost${path}`),
		parent: async () => ({ isAdmin, ui })
	};
}

function detailEvent(fetchImpl: typeof fetch, path = '/series/5') {
	return {
		params: { id: '5' },
		locals: { sessionToken: 'token-123' },
		fetch: fetchImpl,
		url: new URL(`http://localhost${path}`)
	};
}

function actionEvent(
	fetchImpl: typeof fetch,
	action: string,
	fields: Record<string, string> = {},
	token: string | null = 'token-123'
) {
	return {
		request: new Request(`http://localhost/series/5?/${action}`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
			body: new URLSearchParams(fields)
		}),
		locals: { sessionToken: token },
		fetch: fetchImpl,
		params: { id: '5' }
	};
}

describe('carga del listado de series', () => {
	it('pide columnas × filas resultados por página', async () => {
		const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
			const url = new URL(String(input));
			if (url.pathname === '/api/v1/series') {
				return jsonResponse({ series: [seriesItem], total: 500, limit: 50, offset: 50 });
			}
			if (url.pathname === '/api/v1/series/genres') {
				return jsonResponse({ genres: [] });
			}
			throw new Error(`URL inesperada: ${url}`);
		});
		const ui = { ...DEFAULT_PUBLIC_UI, grid_columns: 5, grid_rows: 10 };

		const result = (await loadList(
			listEvent(fetchMock, '/series?page=2', false, ui) as never
		)) as { pageSize: number };

		const listCall = fetchMock.mock.calls.find(
			([input]) => new URL(String(input)).pathname === '/api/v1/series'
		);
		const listURL = new URL(String(listCall?.[0]));
		expect(listURL.searchParams.get('limit')).toBe('50');
		expect(listURL.searchParams.get('offset')).toBe('50');
		expect(result.pageSize).toBe(50);
	});

	it('traduce filtros y página a la API', async () => {
		const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
			const url = new URL(String(input));
			if (url.pathname === '/api/v1/series') {
				if (url.searchParams.get('unmatched') === 'true' && url.searchParams.get('limit') === '1') {
					return jsonResponse({ series: [], total: 3, limit: 1, offset: 0 });
				}
				return jsonResponse({ series: [seriesItem], total: 100, limit: 48, offset: 48 });
			}
			if (url.pathname === '/api/v1/series/genres') {
				return jsonResponse({ genres: [{ name: 'Comedia', count: 4 }] });
			}
			throw new Error(`URL inesperada: ${url}`);
		});

		const result = (await loadList(
			listEvent(fetchMock, '/series?q=robot&sort=year&letter=M&genre=Comedia&page=2') as never
		)) as {
			page: number;
			last: number;
			total: number;
			unmatchedCount: number;
			filters: { q: string; letter: string | null; genre: string | null; sort: string };
			genres: Array<{ name: string; count: number }>;
		};

		const mainCall = fetchMock.mock.calls.find(([input]) => {
			const url = new URL(String(input));
			return url.pathname === '/api/v1/series' && url.searchParams.get('limit') === '48';
		});
		expect(mainCall).toBeTruthy();
		const mainUrl = new URL(String(mainCall?.[0]));
		expect(mainUrl.searchParams.get('q')).toBe('robot');
		expect(mainUrl.searchParams.get('letter')).toBe('M');
		expect(mainUrl.searchParams.get('genre')).toBe('Comedia');
		expect(mainUrl.searchParams.get('sort')).toBe('year');
		expect(mainUrl.searchParams.get('offset')).toBe('48');
		expect(result.page).toBe(2);
		expect(result.last).toBe(3);
		expect(result.total).toBe(100);
		expect(result.unmatchedCount).toBe(3);
		expect(result.genres).toEqual([{ name: 'Comedia', count: 4 }]);
		expect(result.filters).toMatchObject({
			q: 'robot',
			letter: 'M',
			genre: 'Comedia',
			sort: 'year'
		});
	});

	it('redirige a la última página si se pasa de rango', async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			jsonResponse({ series: [], total: 100, limit: 48, offset: 0 })
		);
		await expect(
			loadList(listEvent(fetchMock, '/series?page=99') as never)
		).rejects.toMatchObject({ status: 303, location: '/series?page=3' });
	});

	it('redirige al login con 401', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ error: 'sin sesión' }, 401));
		await expect(loadList(listEvent(fetchMock) as never)).rejects.toMatchObject({
			status: 303,
			location: '/login'
		});
	});

	it('devuelve un estado vacío con mensaje si la API falla', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ error: 'fallo' }, 500));
		const result = (await loadList(listEvent(fetchMock) as never)) as {
			series: Series[];
			total: number;
			errorMessage: string | null;
		};
		expect(result.series).toEqual([]);
		expect(result.total).toBe(0);
		expect(result.errorMessage).toBeTruthy();
	});

	it('no consulta las series sin identificar para quien no es admin', async () => {
		const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
			const url = new URL(String(input));
			if (url.pathname === '/api/v1/series') {
				return jsonResponse({ series: [], total: 0, limit: 48, offset: 0 });
			}
			if (url.pathname === '/api/v1/series/genres') {
				return jsonResponse({ genres: [] });
			}
			throw new Error(`URL inesperada: ${url}`);
		});
		const result = (await loadList(listEvent(fetchMock, '/series?unmatched=1', false) as never)) as {
			filters: { unmatched: boolean };
			unmatchedCount: number;
		};
		expect(result.filters.unmatched).toBe(false);
		expect(result.unmatchedCount).toBe(0);
		expect(
			fetchMock.mock.calls.some(([input]) => new URL(String(input)).searchParams.get('unmatched') === 'true')
		).toBe(false);
	});
});

describe('carga de la ficha de serie', () => {
	it('carga la serie y la temporada pedida', async () => {
		const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
			const url = new URL(String(input));
			if (url.pathname === '/api/v1/series/5') {
				return jsonResponse(detail);
			}
			if (url.pathname === '/api/v1/series/5/seasons/1') {
				return jsonResponse(seasonDetail);
			}
			if (url.pathname === '/api/v1/series/5/seasons/2') {
				return jsonResponse({ ...seasonDetail, season: 2, name: 'Temporada 2', items: [detail.next_episode] });
			}
			throw new Error(`URL inesperada: ${url}`);
		});

		const result = (await loadDetail(detailEvent(fetchMock, '/series/5?season=1') as never)) as {
			series: SeriesDetail;
			season: SeriesSeasonDetail;
			selectedSeason: number;
			firstEpisode: MediaItem | null;
		};
		expect(result.series.id).toBe(5);
		expect(result.selectedSeason).toBe(1);
		expect(result.season.items).toHaveLength(2);
		expect(result.firstEpisode).toBeNull();
	});

	it('usa la primera temporada si la pedida no es válida', async () => {
		const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
			const url = new URL(String(input));
			if (url.pathname === '/api/v1/series/5') {
				return jsonResponse(detail);
			}
			if (url.pathname === '/api/v1/series/5/seasons/1') {
				return jsonResponse(seasonDetail);
			}
			throw new Error(`URL inesperada: ${url}`);
		});
		const result = (await loadDetail(detailEvent(fetchMock, '/series/5?season=9') as never)) as {
			selectedSeason: number;
			season: SeriesSeasonDetail;
		};
		expect(result.selectedSeason).toBe(1);
		expect(result.season.season).toBe(1);
	});

	it('carga el primer episodio cuando ya está todo visto', async () => {
		const watchedDetail: SeriesDetail = {
			...detail,
			watched_count: detail.episode_count,
			next_episode: null
		};
		const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
			const url = new URL(String(input));
			if (url.pathname === '/api/v1/series/5') {
				return jsonResponse(watchedDetail);
			}
			if (url.pathname === '/api/v1/series/5/seasons/1') {
				return jsonResponse(seasonDetail);
			}
			throw new Error(`URL inesperada: ${url}`);
		});
		const result = (await loadDetail(detailEvent(fetchMock) as never)) as {
			selectedSeason: number;
			firstEpisode: MediaItem | null;
		};
		expect(result.selectedSeason).toBe(1);
		expect(result.firstEpisode?.id).toBe(firstEpisode.id);
	});

	it('devuelve 404 si la serie no existe', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ error: 'no existe' }, 404));
		await expect(loadDetail(detailEvent(fetchMock) as never)).rejects.toMatchObject({
			status: 404
		});
	});

	it('redirige al login si no hay sesión', async () => {
		await expect(
			loadDetail({
				params: { id: '5' },
				locals: { sessionToken: null },
				fetch: vi.fn(),
				url: new URL('http://localhost/series/5')
			} as never)
		).rejects.toMatchObject({ status: 303, location: '/login' });
	});
});

describe('acciones de visionado de series', () => {
	it('marca y desmarca la serie completa', async () => {
		for (const [watched, method] of [
			['1', 'PUT'],
			['0', 'DELETE']
		] as const) {
			const fetchMock = vi.fn().mockResolvedValue(emptyResponse());
			const result = (await actions.watchedSeries!(
				actionEvent(fetchMock, 'watchedSeries', { watched }) as never
			)) as { success?: boolean; message?: string };
			expect(result.success).toBe(true);
			const [url, init] = fetchMock.mock.calls[0];
			expect(url).toBe('http://api.test/api/v1/series/5/watched');
			expect(init.method).toBe(method);
		}
	});

	it('marca y desmarca una temporada', async () => {
		for (const [watched, method] of [
			['1', 'PUT'],
			['0', 'DELETE']
		] as const) {
			const fetchMock = vi.fn().mockResolvedValue(emptyResponse());
			const result = (await actions.watchedSeason!(
				actionEvent(fetchMock, 'watchedSeason', { season: '2', watched }) as never
			)) as { success?: boolean };
			expect(result.success).toBe(true);
			const [url, init] = fetchMock.mock.calls[0];
			expect(url).toBe('http://api.test/api/v1/series/5/seasons/2/watched');
			expect(init.method).toBe(method);
		}
	});

	it('marca y desmarca un episodio', async () => {
		for (const [watched, method] of [
			['1', 'PUT'],
			['0', 'DELETE']
		] as const) {
			const fetchMock = vi.fn().mockResolvedValue(emptyResponse());
			const result = (await actions.watchedEpisode!(
				actionEvent(fetchMock, 'watchedEpisode', { itemId: '12', watched }) as never
			)) as { success?: boolean };
			expect(result.success).toBe(true);
			const [url, init] = fetchMock.mock.calls[0];
			expect(url).toBe('http://api.test/api/v1/items/12/watched');
			expect(init.method).toBe(method);
		}
	});

	it('valida el estado y traduce errores', async () => {
		const fetchMock = vi.fn();
		const invalid = (await actions.watchedSeries!(
			actionEvent(fetchMock, 'watchedSeries', { watched: 'quizá' }) as never
		)) as { status: number };
		expect(invalid.status).toBe(400);
		expect(fetchMock).not.toHaveBeenCalled();

		const notFound = vi.fn().mockResolvedValue(jsonResponse({ error: 'no existe' }, 404));
		const result = (await actions.watchedSeries!(
			actionEvent(notFound, 'watchedSeries', { watched: '1' }) as never
		)) as { status: number };
		expect(result.status).toBe(404);

		const failure = vi.fn().mockResolvedValue(jsonResponse({ error: 'fallo' }, 502));
		const failed = (await actions.watchedSeason!(
			actionEvent(failure, 'watchedSeason', { season: '1', watched: '1' }) as never
		)) as { status: number };
		expect(failed.status).toBe(502);
	});

	it('redirige al login sin sesión', async () => {
		await expect(
			actions.watchedEpisode!(
				actionEvent(vi.fn(), 'watchedEpisode', { itemId: '12', watched: '1' }, null) as never
			)
		).rejects.toMatchObject({ status: 303, location: '/login' });
	});
});

describe('render del listado de series', () => {
	it('muestra las series con recuentos y progreso', () => {
		const { body } = render(SeriesListPage, {
			props: { data: listData() as never }
		});
		expect(body).toContain('Series');
		expect(body).toContain('The Big Bang Theory');
		expect(body).toContain('2 temporadas · 5 episodios');
		expect(body).toContain('2/5 vistos');
		expect(body).toContain('href="/series/5"');
	});

	it('distingue los estados vacíos con y sin filtros', () => {
		const empty = render(SeriesListPage, {
			props: { data: listData({ series: [], total: 0 }) as never }
		}).body;
		expect(empty).toContain('Todavía no hay series en el estante');
		const filtered = render(SeriesListPage, {
			props: {
				data: listData({
					series: [],
					total: 0,
					filterChips: [{ key: 'q', label: 'Búsqueda: “x”', href: '/series' }]
				}) as never
			}
		}).body;
		expect(filtered).toContain('Ninguna serie coincide');
		expect(filtered).toContain('Limpiar filtros');
	});
});

describe('render de la ficha de serie', () => {
	it('muestra cabecera, créditos, temporadas y episodios', () => {
		const { body } = render(SeriesPage, {
			props: { data: detailData() as never }
		});
		expect(body).toContain('The Big Bang Theory');
		expect(body).toContain('The Big Bang Theory (original)');
		expect(body).toContain('Reparto y equipo');
		expect(body).toContain('Chuck Lorre');
		expect(body).toContain('Actor 13');
		expect(body).toContain('https://www.themoviedb.org/tv/1418');
		expect(body).toContain('https://images.example/t1.jpg');
		expect(body).toContain('season-card');
		expect(body).toContain('1/2 vistos');
		expect(body).toContain('season-poster');
		expect(body).toContain('https://images.example/still-11.jpg');
		expect(body).toContain('E1 · Piloto');
		expect(body).toContain('E2 · Segundo');
		expect(body).toContain('Marcar serie como vista');
		expect(body).toContain('Marcar temporada como vista');
		expect(body).toContain('Marcar como visto');
		expect(body).toContain('Continuar');
	});

	it('muestra «Más como este» al final, después de temporadas y antes de administración', () => {
		const similar = { ...seriesItem, id: 6, title: 'Young Sheldon' };
		const withSimilar = render(SeriesPage, {
			props: {
				data: detailData({
					similarSeries: [similar],
					isAdmin: true,
					corregir: true
				}) as never
			}
		}).body;
		expect(withSimilar).toContain('Más como este');
		expect(withSimilar).toContain('Young Sheldon');
		const similarAt = withSimilar.indexOf('Más como este');
		expect(withSimilar.indexOf('Temporada 1')).toBeLessThan(similarAt);
		expect(withSimilar.indexOf('E2 · Segundo')).toBeLessThan(similarAt);
		expect(similarAt).toBeLessThan(withSimilar.indexOf('Cambiar imagen'));
		expect(similarAt).toBeLessThan(withSimilar.indexOf('Corregir ficha'));

		const withoutSimilar = render(SeriesPage, {
			props: {
				data: detailData({ similarSeries: [] }) as never
			}
		}).body;
		expect(withoutSimilar).not.toContain('Más como este');
	});

	it('empieza, continúa o vuelve a ver según el estado', () => {
		const noWatched = render(SeriesPage, {
			props: { data: detailData({ series: { ...detail, watched_count: 0, next_episode: firstEpisode } }) as never }
		}).body;
		expect(noWatched).toContain('Empezar · T1 E1');

		const partial = render(SeriesPage, {
			props: { data: detailData() as never }
		}).body;
		expect(partial).toContain('Continuar · T1 E2 · Segundo');

		const allWatched = render(SeriesPage, {
			props: {
				data: detailData({
					series: { ...detail, watched_count: detail.episode_count, next_episode: null },
					firstEpisode
				}) as never
			}
		}).body;
		expect(allWatched).toContain('Volver a ver');
		expect(allWatched).toContain('Marcar serie como no vista');
	});

	it('avisa de serie sin identificar y sinopsis no castellana', () => {
		const { body } = render(SeriesPage, {
			props: {
				data: detailData({
					series: {
						...detail,
						match_status: null,
						overview_language: 'en-US'
					}
				}) as never
			}
		});
		expect(body).toContain('Sin identificar');
		expect(body).toContain('Sinopsis no disponible en castellano.');
	});

	it('usa el cartel de la serie si el episodio no tiene fotograma', () => {
		const fallback = episode(14, 1, 3, 'Sin fotograma', {
			still_url: null,
			series_poster_url: 'https://images.example/serie-respaldo.jpg'
		});
		const { body } = render(SeriesPage, {
			props: {
				data: detailData({
					season: { ...seasonDetail, items: [fallback] }
				}) as never
			}
		});
		expect(body).toContain('https://images.example/serie-respaldo.jpg');
		expect(body).toContain('episode-still');
	});

	it('representa episodios sin sinopsis y sin still', () => {
		const bare = episode(13, 2, 3, 'Sin extras', { overview: null, still_url: null, progress: null });
		const { body } = render(SeriesPage, {
			props: {
				data: detailData({
					season: { ...seasonDetail, season: 2, name: 'Temporada 2', items: [bare] },
					selectedSeason: 2
				}) as never
			}
		});
		expect(body).toContain('E3 · Sin extras');
		expect(body).toContain('E3');
		expect(body).not.toContain('Sinopsis del segundo.');
	});

	it('muestra el estado vacío sin temporadas', () => {
		const { body } = render(SeriesPage, {
			props: {
				data: {
					series: { ...detail, seasons: [], season_count: 0, episode_count: 0, watched_count: 0 },
					season: null,
					selectedSeason: null,
					firstEpisode: null
				} as never
			}
		});
		expect(body).toContain('Esta serie todavía no tiene episodios');
	});
});
