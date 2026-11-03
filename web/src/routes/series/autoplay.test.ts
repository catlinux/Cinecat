import { render } from 'svelte/server';
import { describe, expect, it, vi } from 'vitest';
import type { SeriesDetail } from '$lib/api';
import SeriesPage from './[id]/+page.svelte';
import { actions } from './[id]/+page.server';
import { POST } from './[id]/autoplay/+server';

function jsonResponse(body: unknown, status = 200): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

function detail(autoplayNext: boolean): SeriesDetail {
	return {
		id: 5,
		title: 'The Big Bang Theory',
		file_title: 'The Big Bang theory',
		folder: 'The Big Bang theory',
		original_title: null,
		year: 2007,
		tagline: null,
		overview: null,
		overview_language: null,
		poster_url: null,
		backdrop_url: null,
		genres: [],
		rating: null,
		directors: [],
		cast: [],
		credits: { cast: [], crew: [] },
		tmdb_id: null,
		match_status: null,
		match_score: null,
		poster_custom: false,
		backdrop_custom: false,
		season_count: 0,
		episode_count: 0,
		watched_count: 0,
		favorite: false,
		watchlist: false,
		seasons: [],
		next_episode: null,
		autoplay_next: autoplayNext
	};
}

function detailData(autoplayNext: boolean) {
	return {
		series: detail(autoplayNext),
		season: null,
		selectedSeason: null,
		firstEpisode: null,
		isAdmin: false,
		corregir: false,
		imagenes: false,
		similarSeries: []
	};
}

function actionEvent(fetchImpl: typeof fetch, fields: Record<string, string>, token: string | null = 'token-123') {
	return {
		request: new Request('http://localhost/series/5?/autoplay', {
			method: 'POST',
			headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
			body: new URLSearchParams(fields)
		}),
		locals: { sessionToken: token },
		fetch: fetchImpl,
		params: { id: '5' }
	};
}

function endpointEvent(fetchImpl: typeof fetch, body: string, token: string | null = 'token-123') {
	return {
		params: { id: '5' },
		request: new Request('http://localhost/series/5/autoplay', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body
		}),
		locals: { sessionToken: token },
		fetch: fetchImpl
	};
}

describe('interruptor de reproducción automática en la ficha', () => {
	it('muestra el estado activado y un formulario que funciona sin JavaScript', () => {
		const body = render(SeriesPage, {
			props: { data: detailData(true) as never }
		}).body;

		expect(body).toContain('Reproducir el siguiente episodio automáticamente');
		expect(body).toContain('role="switch"');
		expect(body).toContain('aria-checked="true"');
		expect(body).toContain('Activado');
		expect(body).toContain('name="enabled" value="0"');
	});

	it('muestra el estado desactivado', () => {
		const body = render(SeriesPage, {
			props: { data: detailData(false) as never }
		}).body;

		expect(body).toContain('Desactivado');
		expect(body).toContain('aria-checked="false"');
		expect(body).toContain('name="enabled" value="1"');
	});

	it('guarda el valor con la acción de formulario', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ enabled: false }));
		const result = (await actions.autoplay!(
			actionEvent(fetchMock, { enabled: '0' }) as never
		)) as { success?: boolean; autoplay?: boolean; message?: string };

		expect(result.success).toBe(true);
		expect(result.autoplay).toBe(false);
		expect(result.message).toBe('Guardado');
		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe('http://api.test/api/v1/series/5/autoplay');
		expect(init.method).toBe('PUT');
		expect(init.headers.Authorization).toBe('Bearer token-123');
		expect(JSON.parse(init.body)).toEqual({ enabled: false });
	});

	it('traduce los errores del servidor y exige sesión', async () => {
		const failure = vi.fn().mockResolvedValue(jsonResponse({ error: 'fallo' }, 500));
		const result = (await actions.autoplay!(
			actionEvent(failure, { enabled: '1' }) as never
		)) as { status: number; data: { message: string } };
		expect(result.status).toBe(502);
		expect(result.data.message).toContain('No se pudo guardar');

		await expect(
			actions.autoplay!(actionEvent(vi.fn(), { enabled: '1' }, null) as never)
		).rejects.toMatchObject({ status: 303, location: '/login' });
	});
});

describe('endpoint web de reproducción automática', () => {
	it('reenvía el PUT con la sesión y devuelve el estado', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ enabled: true }));
		const response = await POST(
			endpointEvent(fetchMock, JSON.stringify({ enabled: true })) as never
		);
		expect(response.status).toBe(200);
		expect(await response.json()).toEqual({ enabled: true });

		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe('http://api.test/api/v1/series/5/autoplay');
		expect(init.method).toBe('PUT');
		expect(init.headers.Authorization).toBe('Bearer token-123');
		expect(JSON.parse(init.body)).toEqual({ enabled: true });
	});

	it('rechaza cuerpos no válidos y peticiones sin sesión', async () => {
		for (const body of ['no es json', '{}', '{"enabled":"sí"}', '[]']) {
			const fetchMock = vi.fn();
			const response = await POST(endpointEvent(fetchMock, body) as never);
			expect(response.status).toBe(400);
			expect(fetchMock).not.toHaveBeenCalled();
		}

		const noSession = await POST(endpointEvent(vi.fn(), '{"enabled":true}', null) as never);
		expect(noSession.status).toBe(401);
	});

	it('propaga los errores de la API', async () => {
		const missing = vi.fn().mockResolvedValue(jsonResponse({ error: 'no existe' }, 404));
		expect((await POST(endpointEvent(missing, '{"enabled":false}') as never)).status).toBe(404);

		const failed = vi.fn().mockResolvedValue(jsonResponse({ error: 'fallo' }, 500));
		expect((await POST(endpointEvent(failed, '{"enabled":false}') as never)).status).toBe(502);
	});
});
