import { render } from 'svelte/server';
import { describe, expect, it, vi } from 'vitest';
import type { MediaItem } from '$lib/api';
import ItemPage from './+page.svelte';
import { actions } from './+page.server';

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
		overview: 'Una familia muy unida.',
		poster_url: null,
		poster_custom: false,
		original_title: null,
		imdb_id: null,
		tagline: null,
		genres: [],
		runtime_minutes: 112,
		rating: null,
		directors: [],
		cast: [],
		backdrop_url: null,
		backdrop_custom: false,
		overview_language: null,
		match_status: 'auto',
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

function actionEvent(
	fetchImpl: typeof fetch,
	action: string,
	fields: Record<string, string> = {},
	id = '7'
) {
	return {
		request: new Request(`http://localhost/items/${id}?/${action}`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
			body: new URLSearchParams(fields)
		}),
		locals: { sessionToken: 'token-123' },
		fetch: fetchImpl,
		params: { id },
		url: new URL(`http://localhost/items/${id}`)
	};
}

function renderPage(item: MediaItem, isAdmin: boolean, segments: unknown): string {
	return render(ItemPage, {
		props: {
			data: {
				item,
				isAdmin,
				segments,
				corregir: false,
				similarItems: null,
				episodeNavigation: null
			},
			form: null
		} as never
	}).body;
}

describe('editor de segmentos', () => {
	it('muestra la sección solo a administradores y precarga los manuales', () => {
		const segments = {
			manual: [{ kind: 'intro', start: 10, end: 25 }],
			chapters: [{ kind: 'credits', start: 3500, end: 3560 }]
		};
		const admin = renderPage(makeItem(), true, segments);
		expect(admin).toContain('Intro y créditos');
		expect(admin).toContain('name="intro_start"');
		expect(admin).toContain('value="00:10"');
		expect(admin).toContain('value="00:25"');
		expect(admin).toContain('Detectado en capítulos: 58:20');
		expect(admin).toContain('action="?/segmentsSave"');
		expect(admin).not.toContain('Aplicar a toda la temporada');

		const user = renderPage(makeItem(), false, segments);
		expect(user).not.toContain('Intro y créditos');
		expect(user).not.toContain('action="?/segmentsSave"');

		const episode = renderPage(
			makeItem({
				kind: 'episode',
				series_id: 4,
				season: 1,
				episode: 2,
				title: 'Episodio'
			}),
			true,
			segments
		);
		expect(episode).toContain('action="?/segmentsApplySeason"');
		expect(episode).toContain('Aplicar a toda la temporada');
	});

	it('guarda los segmentos y envía solo las filas completas', async () => {
		const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
		const result = (await actions.segmentsSave!(
			actionEvent(fetchMock, 'segmentsSave', {
				intro_start: '00:10',
				intro_end: '00:25',
				credits_start: '',
				credits_end: ''
			}) as never
		)) as { success?: boolean; message?: string };

		expect(result.success).toBe(true);
		expect(result.message).toContain('guardados');
		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe('http://api.test/api/v1/items/7/segments');
		expect(init.method).toBe('PUT');
		expect(init.headers.Authorization).toBe('Bearer token-123');
		expect(JSON.parse(String(init.body))).toEqual({
			segments: [{ kind: 'intro', start: 10, end: 25 }]
		});
	});

	it('permite borrar todos los segmentos con filas vacías', async () => {
		const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
		const result = (await actions.segmentsSave!(
			actionEvent(fetchMock, 'segmentsSave', {
				intro_start: '',
				intro_end: '',
				credits_start: '',
				credits_end: ''
			}) as never
		)) as { success?: boolean };
		expect(result.success).toBe(true);
		const [, init] = fetchMock.mock.calls[0];
		expect(JSON.parse(String(init.body))).toEqual({ segments: [] });
	});

	it('rechaza tiempos inválidos antes de llamar a la API', async () => {
		const fetchMock = vi.fn();
		const result = (await actions.segmentsSave!(
			actionEvent(fetchMock, 'segmentsSave', {
				intro_start: '00:xx',
				intro_end: '00:25'
			}) as never
		)) as { status: number; data: { segmentsPanel: boolean; message: string } };

		expect(result.status).toBe(400);
		expect(result.data.segmentsPanel).toBe(true);
		expect(result.data.message).toContain('mm:ss');
		expect(fetchMock).not.toHaveBeenCalled();
	});

	it('traduce el error del servidor y el de permisos al guardar', async () => {
		const invalid = vi.fn().mockResolvedValue(jsonResponse({ error: 'kind no válido' }, 400));
		const invalidResult = (await actions.segmentsSave!(
			actionEvent(invalid, 'segmentsSave', {
				intro_start: '00:10',
				intro_end: '00:25'
			}) as never
		)) as { status: number; data: { segmentsPanel: boolean; message: string } };
		expect(invalidResult.status).toBe(400);
		expect(invalidResult.data.segmentsPanel).toBe(true);
		expect(invalidResult.data.message).toBe('kind no válido');

		const forbidden = vi.fn().mockResolvedValue(jsonResponse({ error: 'no admin' }, 403));
		const forbiddenResult = (await actions.segmentsSave!(
			actionEvent(forbidden, 'segmentsSave', {
				intro_start: '00:10',
				intro_end: '00:25'
			}) as never
		)) as { status: number; data: { message: string } };
		expect(forbiddenResult.status).toBe(403);
		expect(forbiddenResult.data.message).toContain('administradores');
	});

	it('aplica los segmentos a la temporada', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ updated: 2 }));
		const result = (await actions.segmentsApplySeason!(
			actionEvent(fetchMock, 'segmentsApplySeason') as never
		)) as { success?: boolean; message?: string };

		expect(result.success).toBe(true);
		expect(result.message).toContain('2 episodios');
		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe('http://api.test/api/v1/items/7/segments/apply-season');
		expect(init.method).toBe('POST');
		expect(init.headers.Authorization).toBe('Bearer token-123');
	});

	it('traduce el error al aplicar la temporada', async () => {
		const invalid = vi.fn().mockResolvedValue(jsonResponse({ error: 'no es episodio' }, 400));
		const result = (await actions.segmentsApplySeason!(
			actionEvent(invalid, 'segmentsApplySeason') as never
		)) as { status: number; data: { segmentsPanel: boolean; message: string } };
		expect(result.status).toBe(400);
		expect(result.data.segmentsPanel).toBe(true);
		expect(result.data.message).toBe('no es episodio');
	});
});
