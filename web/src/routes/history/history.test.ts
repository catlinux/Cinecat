import { render } from 'svelte/server';
import { describe, expect, it, vi } from 'vitest';
import type { HistoryEntry, HistoryResponse } from '$lib/api';
import HistoryPage from './+page.svelte';
import { actions, load } from './+page.server';

const deleteAction = actions.delete!;

function jsonResponse(body: unknown, status = 200): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

const movie: HistoryEntry = {
	item_id: 7,
	title: 'El Padrino',
	kind: 'movie',
	series_title: null,
	season: null,
	episode: null,
	poster_url: 'padrino.jpg',
	position_seconds: 600,
	duration_seconds: 7200,
	watched: false,
	updated_at: '2026-10-31T10:00:00Z'
};

const episode: HistoryEntry = {
	item_id: 8,
	title: 'El episodio',
	kind: 'episode',
	series_title: 'The Big Bang Theory',
	season: 2,
	episode: 5,
	poster_url: null,
	position_seconds: 0,
	duration_seconds: 0,
	watched: true,
	updated_at: '2026-10-30T10:00:00Z'
};

function loadEvent(fetchImpl: typeof fetch, token: string | null = 'token-123') {
	return {
		locals: { sessionToken: token },
		fetch: fetchImpl,
		url: new URL('http://localhost/history')
	};
}

function actionEvent(fetchImpl: typeof fetch, itemID: string, token: string | null = 'token-123') {
	return {
		request: new Request('http://localhost/history?/delete', {
			method: 'POST',
			headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
			body: new URLSearchParams({ item_id: itemID })
		}),
		locals: { sessionToken: token },
		fetch: fetchImpl,
		url: new URL('http://localhost/history')
	};
}

describe('carga del historial', () => {
	it('carga las entradas del usuario', async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			jsonResponse({ entries: [movie, episode] } satisfies HistoryResponse)
		);
		const result = (await load(loadEvent(fetchMock) as never)) as {
			entries: HistoryEntry[];
		};
		expect(result.entries).toHaveLength(2);
		expect(fetchMock.mock.calls[0][0]).toBe('http://api.test/api/v1/history');
	});

	it('redirige al login si la sesión ha caducado', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ error: 'sesión' }, 401));
		await expect(load(loadEvent(fetchMock) as never)).rejects.toMatchObject({
			status: 303,
			location: '/login'
		});
	});

	it('devuelve un error de carga sin romper la página', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ error: 'fallo' }, 500));
		await expect(load(loadEvent(fetchMock) as never)).rejects.toMatchObject({ status: 502 });
	});
});

describe('página del historial', () => {
	const data = {
		entries: [movie, episode],
		ui: { server_name: 'Cinecat' }
	};

	it('muestra el estado vacío', () => {
		const body = render(HistoryPage, {
			props: { data: { entries: [], ui: { server_name: 'Cinecat' } } as never, form: null }
		}).body;
		expect(body).toContain('Todavía no has visto nada');
	});

	it('lista películas y episodios con progreso o visto y quitar', () => {
		const body = render(HistoryPage, {
			props: { data: data as never, form: null }
		}).body;
		expect(body).toContain('Historial');
		expect(body).toContain('El Padrino');
		expect(body).toContain('The Big Bang Theory');
		expect(body).toContain('T2 E5');
		expect(body).toContain('Visto');
		expect(body).toContain('role="progressbar"');
		expect(body).toContain('Quitar del historial');
		expect(body).toContain('action="?/delete"');
		expect(body).toContain('value="7"');
		expect(body).toContain('href="/items/7"');
		expect(body).toContain('src="padrino.jpg"');
	});
});

describe('quitar una entrada', () => {
	it('llama al borrado y confirma', async () => {
		const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
		const result = (await deleteAction(actionEvent(fetchMock, '7') as never)) as {
			success: boolean;
			message: string;
		};
		expect(result.success).toBe(true);
		expect(result.message).toContain('historial');
		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe('http://api.test/api/v1/history/7');
		expect(init.method).toBe('DELETE');
		expect(init.headers.Authorization).toBe('Bearer token-123');
	});

	it('rechaza un identificador no válido sin llamar a la API', async () => {
		const fetchMock = vi.fn();
		const result = (await deleteAction(actionEvent(fetchMock, '0') as never)) as {
			status: number;
		};
		expect(result.status).toBe(400);
		expect(fetchMock).not.toHaveBeenCalled();
	});

	it('avisa si la entrada ya no está disponible', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ error: 'no existe' }, 404));
		const result = (await deleteAction(actionEvent(fetchMock, '7') as never)) as {
			status: number;
			data: { message: string };
		};
		expect(result.status).toBe(404);
		expect(result.data.message).toContain('ya no está disponible');
	});
});
