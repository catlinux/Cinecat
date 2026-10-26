import { render } from 'svelte/server';
import { describe, expect, it, vi } from 'vitest';
import type {
	FileInfo,
	MediaItem,
	SeriesDetail,
	SeriesSeason,
	SeriesSeasonDetail
} from '$lib/api';
import ItemPage from './+page.svelte';
import { load } from './+page.server';

function jsonResponse(body: unknown, status = 200): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

function episode(
	id: number,
	season: number,
	episodeNumber: number,
	title: string,
	overrides: Partial<MediaItem> = {}
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
		overview: `Sinopsis de ${title}.`,
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
		overview_language: 'es-ES',
		match_status: null,
		match_score: null,
		watched: false,
		progress: null,
		collections: [],
		series_id: 5,
		series_title: 'Una serie',
		episode_title: title,
		still_url: null,
		...overrides
	};
}

const first = episode(11, 1, 1, 'El principio');
const second = episode(12, 1, 2, 'El episodio', {
	still_url: 'https://images.example/still-12.jpg'
});
const third = episode(21, 2, 1, 'La continuación');
const fourth = episode(22, 2, 2, 'El final');

const season1: SeriesSeason = {
	season: 1,
	name: 'Temporada 1',
	overview: null,
	poster_url: null,
	episode_count: 2,
	watched_count: 0
};
const season2: SeriesSeason = {
	season: 2,
	name: 'Temporada 2',
	overview: null,
	poster_url: null,
	episode_count: 2,
	watched_count: 0
};

function detail(seasons: SeriesSeason[] = [season1, season2]): SeriesDetail {
	return {
		id: 5,
		title: 'Una serie',
		file_title: 'Una serie',
		folder: 'Una serie',
		original_title: null,
		year: 2010,
		tagline: null,
		overview: null,
		overview_language: null,
		poster_url: null,
		backdrop_url: null,
		genres: [],
		rating: null,
		directors: [],
		cast: [],
		tmdb_id: null,
		match_status: 'auto',
		match_score: 1,
		poster_custom: false,
		backdrop_custom: false,
		season_count: seasons.length,
		episode_count: 4,
		watched_count: 0,
		seasons,
		next_episode: null
	};
}

function seasonDetail(season: number, items: MediaItem[]): SeriesSeasonDetail {
	return {
		season,
		name: `Temporada ${season}`,
		overview: null,
		poster_url: null,
		items
	};
}

function fileInfo(): FileInfo {
	return {
		container: 'mkv',
		container_long_name: 'Matroska',
		size_bytes: 734003200,
		duration_seconds: 2520,
		bit_rate: 5000000,
		modified_at: '2026-10-26T10:00:00Z',
		file_name: '12.mkv',
		path: '/media/series/12.mkv',
		video: [],
		audio: [],
		subtitles: [],
		chapters: 0,
		quality: '1080p',
		playback: { mode: 'direct', description: '', supported: true }
	};
}

function loadEvent(fetchImpl: typeof fetch, id = '12', path = '/items/12', isAdmin = false) {
	return {
		params: { id },
		locals: { sessionToken: 'token-123' },
		fetch: fetchImpl,
		url: new URL(`http://localhost${path}`),
		parent: async () => ({ isAdmin })
	};
}

function renderPage(
	item: MediaItem,
	options: {
		isAdmin?: boolean;
		episodeNavigation?: {
			previous: MediaItem | null;
			next: MediaItem | null;
		} | null;
	} = {}
): string {
	return render(ItemPage, {
		props: {
			data: {
				item,
				isAdmin: options.isAdmin ?? false,
				corregir: false,
				imagenes: false,
				fileInfo: { data: fileInfo() },
				episodeNavigation: options.episodeNavigation ?? null
			}
		} as never
	}).body;
}

function apiMock(
	item: MediaItem,
	options: { series?: SeriesDetail | null; failure?: number } = {}
) {
	return vi.fn(async (input: RequestInfo | URL) => {
		const url = new URL(String(input));
		if (url.pathname === `/api/v1/items/${item.id}`) {
			return jsonResponse(item);
		}
		if (url.pathname === `/api/v1/items/${item.id}/file-info`) {
			return jsonResponse(fileInfo());
		}
		if (url.pathname === '/api/v1/series/5') {
			if (options.failure) {
				return jsonResponse({ error: 'fallo' }, options.failure);
			}
			return jsonResponse(options.series ?? detail());
		}
		if (url.pathname === '/api/v1/series/5/seasons/1') {
			return jsonResponse(seasonDetail(1, [first, second]));
		}
		if (url.pathname === '/api/v1/series/5/seasons/2') {
			return jsonResponse(seasonDetail(2, [third, fourth]));
		}
		throw new Error(`URL inesperada: ${url}`);
	});
}

describe('carga de la ficha de episodio', () => {
	it('calcula el anterior y el siguiente, incluso al cambiar de temporada', async () => {
		const fetchMock = apiMock(second);
		const result = (await load(loadEvent(fetchMock) as never)) as {
			item: MediaItem;
			episodeNavigation: { previous: MediaItem | null; next: MediaItem | null } | null;
		};

		expect(result.item.id).toBe(12);
		expect(result.episodeNavigation?.previous?.id).toBe(11);
		expect(result.episodeNavigation?.next?.id).toBe(21);
	});

	it('no busca fuera de la serie en el primer episodio', async () => {
		const fetchMock = apiMock(first);
		const result = (await load(loadEvent(fetchMock, '11', '/items/11') as never)) as {
			episodeNavigation: { previous: MediaItem | null; next: MediaItem | null } | null;
		};

		expect(result.episodeNavigation?.previous).toBeNull();
		expect(result.episodeNavigation?.next?.id).toBe(12);
		expect(
			fetchMock.mock.calls.some(([input]) =>
				String(input).includes('/api/v1/series/5/seasons/2')
			)
		).toBe(false);
	});

	it('no busca fuera de la serie en el último episodio', async () => {
		const fetchMock = apiMock(fourth);
		const result = (await load(loadEvent(fetchMock, '22', '/items/22') as never)) as {
			episodeNavigation: { previous: MediaItem | null; next: MediaItem | null } | null;
		};

		expect(result.episodeNavigation?.previous?.id).toBe(21);
		expect(result.episodeNavigation?.next).toBeNull();
	});

	it('se muestra sin navegación si la API de la serie falla', async () => {
		const fetchMock = apiMock(second, { failure: 502 });
		const result = (await load(loadEvent(fetchMock) as never)) as {
			item: MediaItem;
			episodeNavigation: unknown;
		};

		expect(result.item.id).toBe(12);
		expect(result.episodeNavigation).toBeNull();
	});

	it('no consulta la serie si el episodio no tiene series_id', async () => {
		const orphan = episode(30, 1, 1, 'Suelto', { series_id: null, series_title: null });
		const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
			const url = new URL(String(input));
			if (url.pathname === '/api/v1/items/30') {
				return jsonResponse(orphan);
			}
			if (url.pathname === '/api/v1/items/30/file-info') {
				return jsonResponse(fileInfo());
			}
			throw new Error(`URL inesperada: ${url}`);
		});

		const result = (await load(loadEvent(fetchMock, '30', '/items/30') as never)) as {
			episodeNavigation: unknown;
		};

		expect(result.episodeNavigation).toBeNull();
		expect(
			fetchMock.mock.calls.some(([input]) => String(input).includes('/api/v1/series/'))
		).toBe(false);
	});
});

describe('render de la ficha de episodio', () => {
	it('muestra migas, cabecera, sinopsis, duración y archivo', () => {
		const body = renderPage(second);

		expect(body).toContain('aria-label="Ruta"');
		expect(body).toContain('href="/series"');
		expect(body).toContain('href="/series/5"');
		expect(body).toContain('href="/series/5?season=1"');
		expect(body).toContain('Una serie');
		expect(body).toContain('E2 · El episodio');
		expect(body).toContain('T1E2');
		expect(body).toContain('Temporada 1, episodio 2');
		expect(body).toContain('Sinopsis de El episodio.');
		expect(body).toContain('https://images.example/still-12.jpg');
		expect(body).toContain('42 min');
		expect(body).toContain('Archivo');
		expect(body).toContain('aria-current="page"');
	});

	it('muestra el estado sin sinopsis', () => {
		const body = renderPage(episode(12, 1, 2, 'El episodio', { overview: null }));
		expect(body).toContain('Sin sinopsis');
	});

	it('enlaza el episodio anterior y el siguiente con su título', () => {
		const body = renderPage(second, {
			episodeNavigation: { previous: first, next: third }
		});

		expect(body).toContain('← Episodio anterior');
		expect(body).toContain('Siguiente episodio →');
		expect(body).toContain('href="/items/11"');
		expect(body).toContain('href="/items/21"');
		expect(body).toContain('T1 E1 · El principio');
		expect(body).toContain('T2 E1 · La continuación');
	});

	it('solo ofrece «Corregir la serie» al administrador', () => {
		const admin = renderPage(second, { isAdmin: true });
		expect(admin).toContain('Corregir la serie');
		expect(admin).toContain('href="/series/5?corregir=1"');
		expect(admin).not.toContain('Corregir ficha');
		expect(admin).not.toContain('Cambiar imagen');

		const user = renderPage(second);
		expect(user).not.toContain('Corregir la serie');
	});

	it('no enlaza la serie si el episodio no tiene series_id', () => {
		const orphan = episode(30, 1, 1, 'Suelto', { series_id: null, series_title: null });
		const body = renderPage(orphan, { isAdmin: true });

		expect(body).toContain('Suelto');
		expect(body).not.toContain('href="/series');
		expect(body).not.toContain('Corregir la serie');
	});
});
