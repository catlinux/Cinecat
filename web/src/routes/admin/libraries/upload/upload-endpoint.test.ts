import { describe, expect, it, vi } from 'vitest';
import { PUT } from './+server';

function jsonResponse(body: unknown, status = 200): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

function event(
	fetchImpl: typeof fetch,
	token: string | null = 'token-123',
	options: { body?: string; contentLength?: string | null; query?: Record<string, string> } = {}
) {
	const url = new URL('http://localhost/admin/libraries/upload');
	url.searchParams.set('library', '4');
	url.searchParams.set('dir', '/media/pelis');
	url.searchParams.set('name', 'Peli.mkv');
	if (options.query) {
		for (const [key, value] of Object.entries(options.query)) {
			url.searchParams.set(key, value);
		}
	}
	const headers = new Headers({ 'Content-Type': 'application/octet-stream' });
	if (options.contentLength !== null) {
		headers.set('Content-Length', options.contentLength ?? '6');
	}
	const request = new Request(url, {
		method: 'PUT',
		headers,
		body: options.body ?? null
	});
	return {
		request,
		url,
		locals: { sessionToken: token },
		fetch: fetchImpl,
		cookies: {},
		params: { id: '4' },
		route: { id: '/admin/libraries/upload' },
		setHeaders: vi.fn(),
		getClientAddress: () => '127.0.0.1',
		isDataRequest: false,
		isSubRequest: false,
		platform: undefined
	};
}

describe('PUT /admin/libraries/upload', () => {
	it('reenvía el cuerpo, los parámetros y la autorización en streaming', async () => {
		let forwardedBody = '';
		let forwarded: { url: string; init: RequestInit & { duplex?: string } } | null = null;
		const fetchMock = vi.fn(
			async (input: RequestInfo | URL, init?: RequestInit & { duplex?: string }) => {
				forwarded = { url: String(input), init: init ?? {} };
				if (init?.body) {
					forwardedBody = await new Response(init.body as BodyInit).text();
				}
				return jsonResponse({ path: '/media/pelis/Peli.mkv', size: 6 }, 201);
			}
		);

		const response = await PUT(
			event(fetchMock as never, 'token-123', {
				body: 'abcdef',
				contentLength: '6',
				query: { subdir: 'Temporada 1' }
			}) as never
		);
		const body = (await response.json()) as { path: string; size: number };

		expect(response.status).toBe(201);
		expect(response.headers.get('Cache-Control')).toBe('no-store');
		expect(body).toEqual({ path: '/media/pelis/Peli.mkv', size: 6 });
		expect(forwarded).not.toBeNull();
		expect(forwarded!.url).toBe(
			'http://api.test/api/v1/libraries/4/files?dir=%2Fmedia%2Fpelis&name=Peli.mkv&subdir=Temporada+1'
		);
		expect(forwarded!.init.method).toBe('PUT');
		expect(forwarded!.init.duplex).toBe('half');
		expect(forwarded!.init.headers).toMatchObject({
			Authorization: 'Bearer token-123',
			'Content-Length': '6',
			'Content-Type': 'application/octet-stream'
		});
		expect(forwardedBody).toBe('abcdef');
	});

	it('exige sesión', async () => {
		const fetchMock = vi.fn();
		const response = await PUT(event(fetchMock as never, null) as never);
		expect(response.status).toBe(401);
		expect(fetchMock).not.toHaveBeenCalled();
	});

	it('exige Content-Length', async () => {
		const fetchMock = vi.fn();
		const response = await PUT(
			event(fetchMock as never, 'token-123', { contentLength: null }) as never
		);
		expect(response.status).toBe(411);
		expect(fetchMock).not.toHaveBeenCalled();
	});

	it('reenvía los estados de error y su JSON', async () => {
		for (const [status, message] of [
			[403, 'activa el ajuste'],
			[409, 'ya existe'],
			[413, 'demasiado grande'],
			[507, 'sin espacio']
		] as const) {
			const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ error: message }, status));
			const response = await PUT(event(fetchMock as never) as never);
			const body = (await response.json()) as { error: string };
			expect(response.status).toBe(status);
			expect(body.error).toBe(message);
		}
	});

	it('devuelve 502 si no se puede contactar con la API', async () => {
		const fetchMock = vi.fn().mockRejectedValue(new Error('sin conexión'));
		const response = await PUT(event(fetchMock as never) as never);
		const body = (await response.json()) as { error: string };
		expect(response.status).toBe(502);
		expect(body.error).toContain('No se pudo contactar');
	});
});
