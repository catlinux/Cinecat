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
