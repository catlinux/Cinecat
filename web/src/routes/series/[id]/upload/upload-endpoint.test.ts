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
	options: { body?: string; contentLength?: string | null } = {}
) {
	const url = new URL('http://localhost/series/7/upload');
	url.searchParams.set('season', '2');
	url.searchParams.set('episode', '5');
	url.searchParams.set('name', 'Capitulo.es.srt');
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
		params: { id: '7' },
		route: { id: '/series/[id]/upload' },
		setHeaders: vi.fn(),
		getClientAddress: () => '127.0.0.1',
		isDataRequest: false,
		isSubRequest: false,
		platform: undefined
	};
}

describe('PUT /series/[id]/upload', () => {
	it('reenvía el cuerpo, temporada, episodio y nombre en streaming', async () => {
		let forwardedBody = '';
		let forwarded: { url: string; init: RequestInit & { duplex?: string } } | null = null;
		const fetchMock = vi.fn(
			async (input: RequestInfo | URL, init?: RequestInit & { duplex?: string }) => {
				forwarded = { url: String(input), init: init ?? {} };
				if (init?.body) {
					forwardedBody = await new Response(init.body as BodyInit).text();
				}
				return jsonResponse(
					{ path: '/media/series/Serie S02E05.es.srt', size: 6, season: 2, episode: 5 },
					201
				);
			}
		);

		const response = await PUT(event(fetchMock as never, 'token-123', { body: 'abcdef' }) as never);
		const body = (await response.json()) as {
			path: string;
			size: number;
			season: number;
			episode: number;
		};

		expect(response.status).toBe(201);
		expect(response.headers.get('Cache-Control')).toBe('no-store');
		expect(body.season).toBe(2);
		expect(forwarded).not.toBeNull();
		expect(forwarded!.url).toBe(
			'http://api.test/api/v1/series/7/files?season=2&episode=5&name=Capitulo.es.srt'
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

	it('exige sesión y Content-Length', async () => {
		const unauthorized = await PUT(event(vi.fn() as never, null) as never);
		expect(unauthorized.status).toBe(401);

		const noLength = await PUT(
			event(vi.fn() as never, 'token-123', { contentLength: null }) as never
		);
		expect(noLength.status).toBe(411);
	});

	it('reenvía los estados de error y su JSON', async () => {
		for (const [status, message] of [
			[400, 'temporada no válida'],
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
