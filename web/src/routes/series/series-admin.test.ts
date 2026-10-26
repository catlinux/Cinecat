import { render } from 'svelte/server';
import { describe, expect, it, vi } from 'vitest';
import type {
	Candidate,
	ImageOptions,
	MediaItem,
	SeriesDetail,
	SeriesSeason,
	SeriesSeasonDetail
} from '$lib/api';
import SeriesPage from './[id]/+page.svelte';
import { actions, load } from './[id]/+page.server';

function jsonResponse(body: unknown, status = 200): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

function episode(id: number, season: number, episodeNumber: number, title: string): MediaItem {
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
		still_url: null
	};
}

const firstEpisode = episode(11, 1, 1, 'Piloto');
const season1: SeriesSeason = {
	season: 1,
	name: 'Temporada 1',
	overview: null,
	poster_url: null,
	episode_count: 1,
	watched_count: 0
};
const seasonDetail: SeriesSeasonDetail = {
	season: 1,
	name: 'Temporada 1',
	overview: null,
	poster_url: null,
	items: [firstEpisode]
};

function detail(overrides: Partial<SeriesDetail> = {}): SeriesDetail {
	return {
		id: 5,
		title: 'The Big Bang Theory',
		file_title: 'The Big Bang theory',
		folder: 'The Big Bang theory',
		original_title: 'The Big Bang Theory',
		year: 2007,
		tagline: null,
		overview: 'Una comedia de científicos.',
		overview_language: 'es-ES',
		poster_url: 'https://images.example/serie.jpg',
		backdrop_url: 'https://images.example/fondo.jpg',
		genres: ['Comedia'],
		rating: 8.2,
		directors: [],
		cast: [],
		tmdb_id: 1418,
		match_status: 'auto',
		match_score: 0.92,
		poster_custom: false,
		backdrop_custom: false,
		season_count: 1,
		episode_count: 1,
		watched_count: 0,
		seasons: [season1],
		next_episode: firstEpisode,
		...overrides
	};
}

function detailData(overrides: Record<string, unknown> = {}) {
	return {
		series: detail(),
		season: seasonDetail,
		selectedSeason: 1,
		firstEpisode: null,
		isAdmin: false,
		corregir: false,
		imagenes: false,
		...overrides
	};
}

function actionEvent(
	fetchImpl: typeof fetch,
	action: string,
	fields: Record<string, string> = {}
) {
	return {
		request: new Request(`http://localhost/series/5?/${action}`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
			body: new URLSearchParams(fields)
		}),
		locals: { sessionToken: 'token-123' },
		fetch: fetchImpl,
		params: { id: '5' }
	};
}

function loadEvent(fetchImpl: typeof fetch, path = '/series/5', isAdmin = false) {
	return {
		params: { id: '5' },
		locals: { sessionToken: 'token-123' },
		fetch: fetchImpl,
		url: new URL(`http://localhost${path}`),
		parent: async () => ({ isAdmin })
	};
}

const candidate: Candidate = {
	tmdb_id: 1418,
	title: 'The Big Bang Theory',
	original_title: 'The Big Bang Theory',
	year: 2007,
	overview: 'Una comedia de científicos.',
	poster_url: 'https://images.example/candidate.jpg',
	popularity: 90,
	vote_count: 1000
};

const imageOptions: ImageOptions = {
	posters: [
		{
			url: 'https://images.example/poster-small.jpg',
			full_url: 'https://images.example/poster-full.jpg',
			language: 'es',
			vote_average: 8.1,
			width: 500,
			height: 750
		}
	],
	backdrops: []
};

describe('acciones de corrección de serie', () => {
	it('busca candidatos con el título y el año', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ candidates: [candidate] }));
		const result = (await actions.candidates!(
			actionEvent(fetchMock, 'candidates', {
				q: 'The Big Bang theory',
				year: '2007'
			}) as never
		)) as {
			panel: boolean;
			candidates: Candidate[];
			search: { q: string; year: number | null };
		};

		expect(result.panel).toBe(true);
		expect(result.candidates).toHaveLength(1);
		expect(result.search).toEqual({ q: 'The Big Bang theory', year: 2007 });
		const [url] = fetchMock.mock.calls[0];
		const parsed = new URL(String(url));
		expect(parsed.pathname).toBe('/api/v1/series/5/candidates');
		expect(parsed.searchParams.get('q')).toBe('The Big Bang theory');
		expect(parsed.searchParams.get('year')).toBe('2007');
	});

	it('fija una coincidencia con PUT', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse(detail()));
		const result = (await actions.match!(
			actionEvent(fetchMock, 'match', { tmdb_id: '1418' }) as never
		)) as { success?: boolean; message?: string };

		expect(result.success).toBe(true);
		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe('http://api.test/api/v1/series/5/match');
		expect(init.method).toBe('PUT');
		expect(JSON.parse(init.body)).toEqual({ tmdb_id: 1418 });
	});

	it('marca «no es ninguna» con tmdb_id null', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse(detail()));
		const result = (await actions.nomatch!(actionEvent(fetchMock, 'nomatch') as never)) as {
			success?: boolean;
		};

		expect(result.success).toBe(true);
		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe('http://api.test/api/v1/series/5/match');
		expect(init.method).toBe('PUT');
		expect(JSON.parse(init.body)).toEqual({ tmdb_id: null });
	});

	it('vuelve a automático con DELETE', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse(detail()));
		const result = (await actions.automatic!(
			actionEvent(fetchMock, 'automatic') as never
		)) as { success?: boolean };

		expect(result.success).toBe(true);
		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe('http://api.test/api/v1/series/5/match');
		expect(init.method).toBe('DELETE');
	});

	it('valida los datos antes de llamar a la API', async () => {
		const fetchMock = vi.fn();
		const emptyQuery = (await actions.candidates!(
			actionEvent(fetchMock, 'candidates', { q: '   ' }) as never
		)) as { status: number };
		expect(emptyQuery.status).toBe(400);

		const invalidYear = (await actions.candidates!(
			actionEvent(fetchMock, 'candidates', { q: 'Serie', year: 'dos mil' }) as never
		)) as { status: number };
		expect(invalidYear.status).toBe(400);

		const invalidCandidate = (await actions.match!(
			actionEvent(fetchMock, 'match', { tmdb_id: '0' }) as never
		)) as { status: number };
		expect(invalidCandidate.status).toBe(400);
		expect(fetchMock).not.toHaveBeenCalled();
	});

	it('traduce los errores de permisos y de TMDB', async () => {
		const badRequest = vi.fn().mockResolvedValue(jsonResponse({ error: 'datos' }, 400));
		const badRequestResult = (await actions.candidates!(
			actionEvent(badRequest, 'candidates', { q: 'Serie' }) as never
		)) as { status: number };
		expect(badRequestResult.status).toBe(400);

		const forbidden = vi.fn().mockResolvedValue(jsonResponse({ error: 'no admin' }, 403));
		const forbiddenResult = (await actions.match!(
			actionEvent(forbidden, 'match', { tmdb_id: '1418' }) as never
		)) as { status: number; data: { message: string } };
		expect(forbiddenResult.status).toBe(403);
		expect(forbiddenResult.data.message).toContain('administradores');

		const unavailable = vi.fn().mockResolvedValue(jsonResponse({ error: 'sin TMDB' }, 503));
		const unavailableResult = (await actions.candidates!(
			actionEvent(unavailable, 'candidates', { q: 'Serie' }) as never
		)) as { status: number; data: { message: string } };
		expect(unavailableResult.status).toBe(503);
		expect(unavailableResult.data.message).toBe('TMDB no está configurado en el servidor');

		const broken = vi.fn().mockResolvedValue(jsonResponse({ error: 'fallo' }, 502));
		const brokenResult = (await actions.candidates!(
			actionEvent(broken, 'candidates', { q: 'Serie' }) as never
		)) as { status: number; data: { message: string } };
		expect(brokenResult.status).toBe(502);
		expect(brokenResult.data.message).toBe('No se pudo consultar TMDB');

		const notFound = vi.fn().mockResolvedValue(jsonResponse({ error: 'no existe' }, 404));
		const notFoundResult = (await actions.nomatch!(
			actionEvent(notFound, 'nomatch') as never
		)) as { status: number };
		expect(notFoundResult.status).toBe(404);
	});
});

describe('acciones de imágenes de serie', () => {
	it('lista, fija y restaura las imágenes', async () => {
		const listFetch = vi.fn().mockResolvedValue(jsonResponse(imageOptions));
		const listResult = (await actions.imageList!(
			actionEvent(listFetch, 'imageList', { kind: 'poster' }) as never
		)) as { imagePanel: boolean; imageKind: string; imageOptions: ImageOptions };
		expect(listResult.imagePanel).toBe(true);
		expect(listResult.imageKind).toBe('poster');
		expect(listResult.imageOptions.posters).toHaveLength(1);

		const setFetch = vi.fn().mockResolvedValue(jsonResponse(detail()));
		const setResult = (await actions.imageSet!(
			actionEvent(setFetch, 'imageSet', {
				kind: 'backdrop',
				url: 'https://images.example/nuevo.jpg'
			}) as never
		)) as { success?: boolean };
		expect(setResult.success).toBe(true);
		const [setUrl, setInit] = setFetch.mock.calls[0];
		expect(setUrl).toBe('http://api.test/api/v1/series/5/images/backdrop');
		expect(setInit.method).toBe('PUT');
		expect(JSON.parse(setInit.body)).toEqual({ url: 'https://images.example/nuevo.jpg' });

		const resetFetch = vi.fn().mockResolvedValue(jsonResponse(detail()));
		const resetResult = (await actions.imageReset!(
			actionEvent(resetFetch, 'imageReset', { kind: 'poster' }) as never
		)) as { success?: boolean };
		expect(resetResult.success).toBe(true);
		const [resetUrl, resetInit] = resetFetch.mock.calls[0];
		expect(resetUrl).toBe('http://api.test/api/v1/series/5/images/poster');
		expect(resetInit.method).toBe('DELETE');
	});

	it('traduce los errores de imágenes', async () => {
		const invalidURL = (await actions.imageSet!(
			actionEvent(vi.fn(), 'imageSet', { kind: 'poster', url: 'no-es-una-url' }) as never
		)) as { status: number };
		expect(invalidURL.status).toBe(400);

		const conflict = vi.fn().mockResolvedValue(jsonResponse({ error: 'sin identificar' }, 409));
		const conflictResult = (await actions.imageList!(
			actionEvent(conflict, 'imageList', { kind: 'poster' }) as never
		)) as { status: number; data: { message: string } };
		expect(conflictResult.status).toBe(409);
		expect(conflictResult.data.message).toContain('serie');

		const forbidden = vi.fn().mockResolvedValue(jsonResponse({ error: 'no admin' }, 403));
		const forbiddenResult = (await actions.imageSet!(
			actionEvent(forbidden, 'imageSet', {
				kind: 'poster',
				url: 'https://images.example/poster.jpg'
			}) as never
		)) as { status: number };
		expect(forbiddenResult.status).toBe(403);

		const unavailable = vi.fn().mockResolvedValue(jsonResponse({ error: 'sin TMDB' }, 503));
		const unavailableResult = (await actions.imageList!(
			actionEvent(unavailable, 'imageList', { kind: 'poster' }) as never
		)) as { status: number };
		expect(unavailableResult.status).toBe(503);

		const broken = vi.fn().mockResolvedValue(jsonResponse({ error: 'fallo' }, 502));
		const brokenResult = (await actions.imageReset!(
			actionEvent(broken, 'imageReset', { kind: 'poster' }) as never
		)) as { status: number };
		expect(brokenResult.status).toBe(502);
	});
});

describe('carga de la ficha con corrección', () => {
	it('abre los paneles solo para administradores', async () => {
		const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
			const url = new URL(String(input));
			if (url.pathname === '/api/v1/series/5') {
				return jsonResponse(detail());
			}
			if (url.pathname === '/api/v1/series/5/seasons/1') {
				return jsonResponse(seasonDetail);
			}
			throw new Error(`URL inesperada: ${url}`);
		});

		const admin = (await load(
			loadEvent(fetchMock, '/series/5?corregir=1&imagenes=1', true) as never
		)) as { isAdmin: boolean; corregir: boolean; imagenes: boolean };
		expect(admin.isAdmin).toBe(true);
		expect(admin.corregir).toBe(true);
		expect(admin.imagenes).toBe(true);

		const user = (await load(
			loadEvent(fetchMock, '/series/5?corregir=1&imagenes=1', false) as never
		)) as { isAdmin: boolean; corregir: boolean; imagenes: boolean };
		expect(user.isAdmin).toBe(false);
		expect(user.corregir).toBe(false);
		expect(user.imagenes).toBe(false);
	});
});

describe('render de la corrección de serie', () => {
	it('no muestra los paneles a quien no es administrador', () => {
		const body = render(SeriesPage, {
			props: { data: detailData({ corregir: true, imagenes: true }) as never }
		}).body;

		expect(body).not.toContain('Corregir ficha');
		expect(body).not.toContain('Cambiar imagen');
	});

	it('muestra el panel con la carpeta precargada y el estado automático', () => {
		const body = render(SeriesPage, {
			props: {
				data: detailData({ isAdmin: true, corregir: true }) as never
			}
		}).body;

		expect(body).toMatch(/<details[^>]*\bopen\b/);
		expect(body).toContain('Corrección de ficha de serie');
		expect(body).toContain('Estado actual:');
		expect(body).toContain('Identificada automáticamente (92 %)');
		expect(body).toContain('name="q" type="search" value="The Big Bang theory"');
		expect(body).toContain('action="?/candidates"');
		expect(body).toContain('action="?/nomatch"');
		expect(body).toContain('action="?/automatic"');
	});

	it('muestra los estados manual, ninguno y sin identificar', () => {
		const manual = render(SeriesPage, {
			props: {
				data: detailData({
					isAdmin: true,
					corregir: true,
					series: detail({ match_status: 'manual', match_score: null })
				}) as never
			}
		}).body;
		expect(manual).toContain('Corregida a mano');

		const none = render(SeriesPage, {
			props: {
				data: detailData({
					isAdmin: true,
					corregir: true,
					series: detail({ match_status: 'none', match_score: null })
				}) as never
			}
		}).body;
		expect(none).toContain('Marcada como «no es ninguna»');

		const unidentified = render(SeriesPage, {
			props: {
				data: detailData({
					isAdmin: true,
					corregir: true,
					series: detail({ match_status: null, match_score: null })
				}) as never
			}
		}).body;
		expect(unidentified).toContain('Sin identificar');
	});

	it('renderiza candidatos y el panel de imágenes', () => {
		const body = render(SeriesPage, {
			props: {
				data: detailData({
					isAdmin: true,
					corregir: true,
					imagenes: true,
					series: detail({ poster_custom: true })
				}) as never,
				form: {
					panel: true,
					candidates: [candidate],
					search: { q: 'The Big Bang theory', year: 2007 },
					imagePanel: true,
					imageKind: 'poster',
					imageOptions
				}
			}
		}).body;

		expect(body).toContain('https://images.example/candidate.jpg');
		expect(body).toContain('The Big Bang Theory');
		expect(body).toContain('Es esta');
		expect(body).toContain('name="tmdb_id" value="1418"');
		expect(body).toContain('action="?/match"');
		expect(body).toContain('personalizada');
		expect(body).toContain('https://images.example/poster-small.jpg');
		expect(body).toContain('action="?/imageList"');
		expect(body).toContain('action="?/imageSet"');
		expect(body).toContain('action="?/imageReset"');
	});
});
