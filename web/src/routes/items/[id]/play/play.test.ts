import { render } from 'svelte/server';
import { describe, expect, it, vi } from 'vitest';
import type { MediaItem, PlayResponse } from '$lib/api';
import PlayPage from './+page.svelte';
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
	overview: null,
	poster_url: null,
	poster_custom: false,
	original_title: null,
	imdb_id: null,
	tagline: null,
	genres: [],
	runtime_minutes: 175,
	rating: null,
	directors: [],
	cast: [],
	backdrop_url: null,
	backdrop_custom: false,
	overview_language: null,
	match_status: 'auto',
	match_score: 1,
	watched: false,
	progress: {
		position_seconds: 4320,
		duration_seconds: 7200,
		updated_at: '2026-10-26T10:00:00Z'
	},
	collections: [],
	series_id: null,
	series_title: null,
	episode_title: null,
	still_url: null
};

const play: PlayResponse = {
	url: '/api/v1/stream/7?exp=1&sig=abc',
	mode: 'direct',
	expires_at: '2026-10-26T12:00:00Z'
};

function loadEvent(fetchImpl: typeof fetch, path = '/items/7/play') {
	return {
		params: { id: '7' },
		locals: { sessionToken: 'token-123' },
		fetch: fetchImpl,
		url: new URL(`http://localhost${path}`)
	};
}

function fetchForItem() {
	return vi.fn(async (input: RequestInfo | URL) => {
		const url = new URL(String(input));
		if (url.pathname === '/api/v1/items/7') {
			return jsonResponse(item);
		}
		if (url.pathname === '/api/v1/items/7/play') {
			return jsonResponse(play);
		}
		throw new Error(`URL inesperada: ${url}`);
	});
}

function renderPlay(startAt: number, withProgress = true): string {
	return render(PlayPage, {
		props: {
			data: {
				item: withProgress ? item : { ...item, progress: null },
				play,
				playUrl: 'http://api.test/api/v1/stream/7?exp=1&sig=abc',
				playError: null,
				startAt
			}
		} as never
	}).body;
}

describe('carga de reproducción', () => {
	it('usa la posición guardada', async () => {
		const result = (await load(loadEvent(fetchForItem()) as never)) as { startAt: number };
		expect(result.startAt).toBe(4320);
	});

	it('permite sobrescribir con ?t=', async () => {
		const result = (await load(loadEvent(fetchForItem(), '/items/7/play?t=30') as never)) as {
			startAt: number;
		};
		expect(result.startAt).toBe(30);
	});

	it('vuelve al principio con ?t=0', async () => {
		const result = (await load(loadEvent(fetchForItem(), '/items/7/play?t=0') as never)) as {
			startAt: number;
		};
		expect(result.startAt).toBe(0);
	});

	it('ignora un ?t= no válido', async () => {
		const result = (await load(loadEvent(fetchForItem(), '/items/7/play?t=abc') as never)) as {
			startAt: number;
		};
		expect(result.startAt).toBe(4320);
	});
});

describe('opciones de reproducción en la carga', () => {
	it('pasa a la API los parámetros válidos de audio, calidad y subtítulo', async () => {
		let playURL = '';
		const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
			const url = new URL(String(input));
			if (url.pathname === '/api/v1/items/7') {
				return jsonResponse(item);
			}
			playURL = url.toString();
			return jsonResponse(play);
		});
		const result = (await load(
			loadEvent(fetchMock, '/items/7/play?audio=1&quality=720&subtitle=e0&t=30') as never
		)) as { startAt: number; options: PlayResponse };
		expect(result.startAt).toBe(30);
		const parsed = new URL(playURL);
		expect(parsed.searchParams.get('audio')).toBe('1');
		expect(parsed.searchParams.get('quality')).toBe('720');
		expect(parsed.searchParams.get('subtitle')).toBe('e0');
		expect(result.options).toEqual(play);
	});

	it('ignora los parámetros no válidos y reintenta sin opciones tras un 400', async () => {
		const calls: string[] = [];
		const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
			const url = new URL(String(input));
			if (url.pathname === '/api/v1/items/7') {
				return jsonResponse(item);
			}
			calls.push(url.toString());
			if (url.searchParams.get('audio') === '1') {
				return jsonResponse({ error: 'pista no válida' }, 400);
			}
			return jsonResponse(play);
		});
		const result = (await load(
			loadEvent(fetchMock, '/items/7/play?audio=1&quality=4k&subtitle=bad') as never
		)) as { play: PlayResponse | null };
		expect(result.play).toEqual(play);
		expect(calls).toHaveLength(2);
		expect(new URL(calls[0]).searchParams.get('audio')).toBe('1');
		expect(new URL(calls[1]).searchParams.has('audio')).toBe(false);
	});

	it('calcula el siguiente episodio', async () => {
		const episode = {
			...item,
			id: 7,
			kind: 'episode' as const,
			series_id: 5,
			season: 1,
			episode: 1
		};
		const next = { ...episode, id: 8, episode: 2 };
		const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
			const url = new URL(String(input));
			if (url.pathname === '/api/v1/items/7') {
				return jsonResponse(episode);
			}
			if (url.pathname === '/api/v1/items/7/play') {
				return jsonResponse(play);
			}
			if (url.pathname === '/api/v1/series/5') {
				return jsonResponse({ seasons: [{ season: 1 }] });
			}
			if (url.pathname === '/api/v1/series/5/seasons/1') {
				return jsonResponse({ items: [episode, next] });
			}
			throw new Error(`URL inesperada: ${url}`);
		});
		const result = (await load(loadEvent(fetchMock) as never)) as {
			nextEpisodeUrl: string | null;
		};
		expect(result.nextEpisodeUrl).toBe('/items/8/play');
	});

	it('sigue sin botón si falla la serie', async () => {
		const episode = {
			...item,
			id: 7,
			kind: 'episode' as const,
			series_id: 5,
			season: 1,
			episode: 1
		};
		const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
			const url = new URL(String(input));
			if (url.pathname === '/api/v1/items/7') {
				return jsonResponse(episode);
			}
			if (url.pathname === '/api/v1/items/7/play') {
				return jsonResponse(play);
			}
			return jsonResponse({ error: 'fallo' }, 502);
		});
		const result = (await load(loadEvent(fetchMock) as never)) as {
			nextEpisodeUrl: string | null;
			play: PlayResponse | null;
		};
		expect(result.play).toEqual(play);
		expect(result.nextEpisodeUrl).toBeNull();
	});
});

describe('página de reproducción', () => {
	it('muestra Continúas por y el enlace al principio con progreso', () => {
		const body = renderPlay(4320);

		expect(body).toContain('Continúas por 1 h 12 min');
		expect(body).toContain('Empezar desde el principio');
		expect(body).toContain('/items/7/play?t=0');
		expect(body).toContain('<video');
	});

	it('no muestra el aviso si se empieza desde el principio', () => {
		const body = renderPlay(0);
		expect(body).not.toContain('Continúas por');
	});
});
