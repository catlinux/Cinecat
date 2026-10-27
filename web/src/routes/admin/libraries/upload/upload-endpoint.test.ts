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
		fetch: vi.fn(),
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

beforeEach(() => {
	proxyUpload.mockReset();
});

describe('PUT /admin/libraries/upload', () => {
	it('reenvía el cuerpo, los parámetros y la autorización en streaming', async () => {
		proxyUpload.mockResolvedValue(
			jsonResponse({ path: '/media/pelis/Peli.mkv', size: 6 }, 201)
		);
		const uploadEvent = event('token-123', {
			body: 'abcdef',
			contentLength: '6',
			query: { subdir: 'Temporada 1' }
		});

		const response = await PUT(uploadEvent as never);
		const body = (await response.json()) as { path: string; size: number };

		expect(response.status).toBe(201);
		expect(response.headers.get('Cache-Control')).toBe('no-store');
		expect(body).toEqual({ path: '/media/pelis/Peli.mkv', size: 6 });
		expect(proxyUpload).toHaveBeenCalledTimes(1);
		expect(proxyUpload).toHaveBeenCalledWith(
			uploadEvent.request,
			'/api/v1/libraries/4/files?dir=%2Fmedia%2Fpelis&name=Peli.mkv&subdir=Temporada+1',
			{
				token: 'token-123',
				contentLength: '6',
				errorMessage: 'No se pudo contactar con el servidor para subir el archivo.',
				origin: 'http://localhost',
				publicBase: 'http://api.test'
			}
		);
	});

	it('exige sesión', async () => {
		const response = await PUT(event(null) as never);
		expect(response.status).toBe(401);
		expect(proxyUpload).not.toHaveBeenCalled();
	});

	it('exige Content-Length', async () => {
		const response = await PUT(event('token-123', { contentLength: null }) as never);
		expect(response.status).toBe(411);
		expect(proxyUpload).not.toHaveBeenCalled();
	});

	it('reenvía los estados de error y su JSON', async () => {
		for (const [status, message] of [
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
