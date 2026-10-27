import { beforeEach, describe, expect, it, vi } from 'vitest';

const proxyUpload = vi.hoisted(() => vi.fn());

vi.mock('$lib/server/upload-proxy', () => ({ proxyUpload }));

import { PUT } from './+server';

function jsonResponse(body: unknown, status = 200): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json', 'Cache-Control': 'no-store' }
	});
}

function event(
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
		fetch: vi.fn(),
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

beforeEach(() => {
	proxyUpload.mockReset();
});

describe('PUT /series/[id]/upload', () => {
	it('reenvía el cuerpo, temporada, episodio y nombre en streaming', async () => {
		proxyUpload.mockResolvedValue(
			jsonResponse(
				{ path: '/media/series/Serie S02E05.es.srt', size: 6, season: 2, episode: 5 },
				201
			)
		);
		const uploadEvent = event('token-123', { body: 'abcdef' });

		const response = await PUT(uploadEvent as never);
		const body = (await response.json()) as {
			path: string;
			size: number;
			season: number;
			episode: number;
		};

		expect(response.status).toBe(201);
		expect(response.headers.get('Cache-Control')).toBe('no-store');
		expect(body.season).toBe(2);
		expect(proxyUpload).toHaveBeenCalledTimes(1);
		expect(proxyUpload).toHaveBeenCalledWith(
			uploadEvent.request,
			'/api/v1/series/7/files?season=2&episode=5&name=Capitulo.es.srt',
			{
				token: 'token-123',
				contentLength: '6',
				errorMessage: 'No se pudo contactar con el servidor para subir el capítulo.',
				origin: 'http://localhost',
				publicBase: 'http://api.test'
			}
		);
	});

	it('exige sesión y Content-Length', async () => {
		const unauthorized = await PUT(event(null) as never);
		expect(unauthorized.status).toBe(401);

		const noLength = await PUT(event('token-123', { contentLength: null }) as never);
		expect(noLength.status).toBe(411);
		expect(proxyUpload).not.toHaveBeenCalled();
	});

	it('reenvía los estados de error y su JSON', async () => {
		for (const [status, message] of [
			[400, 'temporada no válida'],
			[403, 'activa el ajuste'],
			[409, 'ya existe'],
			[413, 'demasiado grande'],
			[507, 'sin espacio']
		] as const) {
			proxyUpload.mockResolvedValue(jsonResponse({ error: message }, status));
			const response = await PUT(event() as never);
			const body = (await response.json()) as { error: string };
			expect(response.status).toBe(status);
			expect(body.error).toBe(message);
		}
	});

	it('devuelve 502 si no se puede contactar con la API', async () => {
		proxyUpload.mockRejectedValue(new Error('sin conexión'));
		const response = await PUT(event() as never);
		const body = (await response.json()) as { error: string };
		expect(response.status).toBe(502);
		expect(body.error).toContain('No se pudo contactar');
	});
});
