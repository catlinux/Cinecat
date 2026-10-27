import { describe, expect, it, vi } from 'vitest';
import { POST } from './+server';

function formRequest(fields: Record<string, string>, extraHeaders: Record<string, string> = {}): Request {
	const body = new URLSearchParams(fields);
	return new Request('http://localhost/lists', {
		method: 'POST',
		headers: { 'Content-Type': 'application/x-www-form-urlencoded', ...extraHeaders },
		body
	});
}

function jsonRequest(fields: Record<string, unknown>, extraHeaders: Record<string, string> = {}): Request {
	return new Request('http://localhost/lists', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json', Accept: 'application/json', ...extraHeaders },
		body: JSON.stringify(fields)
	});
}

function event(request: Request, fetchImpl: typeof fetch, token: string | null = 'token-123') {
	return { request, fetch: fetchImpl, locals: { sessionToken: token } } as never;
}

const valid = { kind: 'item', id: '7', list: 'favorite', on: '1', redirectTo: '/items/7' };

describe('POST /lists', () => {
	it('marca con formulario y redirige a redirectTo', async () => {
		const fetchMock = vi.fn(async (_input: RequestInfo | URL, _init?: RequestInit) => new Response(null, { status: 204 }));
		const response = await POST(event(formRequest(valid), fetchMock as never));

		expect(response.status).toBe(303);
		expect(response.headers.get('location')).toBe('/items/7');
		expect(fetchMock.mock.calls[0][0]).toBe('http://api.test/api/v1/items/7/lists/favorite');
		expect(fetchMock.mock.calls[0]?.[1]?.method).toBe('PUT');
	});

	it('desmarca con DELETE y con JSON devuelve 204', async () => {
		const fetchMock = vi.fn(async (_input: RequestInfo | URL, _init?: RequestInit) => new Response(null, { status: 204 }));
		const response = await POST(
			event(jsonRequest({ ...valid, on: false, list: 'watchlist' }), fetchMock as never)
		);

		expect(response.status).toBe(204);
		expect(fetchMock.mock.calls[0]?.[1]?.method).toBe('DELETE');
		expect(response.headers.get('Cache-Control')).toBe('no-store');
	});

	it('sanea redirectTo malicioso y vuelve a la portada', async () => {
		const fetchMock = vi.fn(async (_input: RequestInfo | URL, _init?: RequestInit) => new Response(null, { status: 204 }));
		for (const redirectTo of ['//evil.com', 'https://evil.com', 'javascript:alert(1)', '/\\evil']) {
			const response = await POST(event(formRequest({ ...valid, redirectTo }), fetchMock as never));
			expect(response.headers.get('location')).toBe('/');
		}
	});

	it('sin sesión redirige los formularios y devuelve 401 a JSON', async () => {
		const fetchMock = vi.fn();
		const form = await POST(event(formRequest(valid), fetchMock as never, null));
		expect(form.status).toBe(303);
		expect(form.headers.get('location')).toBe('/login');
		const json = await POST(event(jsonRequest(valid), fetchMock as never, null));
		expect(json.status).toBe(401);
		expect(fetchMock).not.toHaveBeenCalled();
	});

	it('valida kind, id, list y on sin llamar a la API', async () => {
		const fetchMock = vi.fn();
		for (const fields of [
			{ ...valid, kind: 'otro' },
			{ ...valid, id: '0' },
			{ ...valid, list: 'otra' },
			{ ...valid, on: 'quizá' }
		]) {
			const json = await POST(event(jsonRequest(fields), fetchMock as never));
			expect(json.status).toBe(400);
			const body = (await json.json()) as { error: string };
			expect(body.error).toBeTruthy();
		}
		expect(fetchMock).not.toHaveBeenCalled();
	});

	it('traduce 404, 401 y 502 de la API', async () => {
		const notFound = vi.fn(async () => new Response(JSON.stringify({ error: 'no' }), { status: 404 }));
		const form404 = await POST(event(formRequest(valid), notFound as never));
		expect(form404.status).toBe(303);
		expect(form404.headers.get('location')).toContain('error=lista');
		const json404 = await POST(event(jsonRequest(valid), notFound as never));
		expect(json404.status).toBe(404);

		const unauthorized = vi.fn(async () => new Response(JSON.stringify({ error: 'no' }), { status: 401 }));
		const form401 = await POST(event(formRequest(valid), unauthorized as never));
		expect(form401.headers.get('location')).toBe('/login');
		const json401 = await POST(event(jsonRequest(valid), unauthorized as never));
		expect(json401.status).toBe(401);

		const broken = vi.fn(async () => new Response(JSON.stringify({ error: 'no' }), { status: 500 }));
		const json502 = await POST(event(jsonRequest(valid), broken as never));
		expect(json502.status).toBe(502);
	});

	it('responde como acción de SvelteKit cuando use:enhance lo pide', async () => {
		const fetchMock = vi.fn(async (_input: RequestInfo | URL, _init?: RequestInit) => new Response(null, { status: 204 }));
		const response = await POST(
			event(formRequest(valid, { 'x-sveltekit-action': 'true' }), fetchMock as never)
		);
		const body = (await response.json()) as { type: string; status: number };
		expect(body).toMatchObject({ type: 'success', status: 204 });
	});
});
