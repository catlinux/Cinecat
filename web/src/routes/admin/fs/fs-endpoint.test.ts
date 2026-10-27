import { describe, expect, it, vi } from 'vitest';
import { GET } from './+server';

function jsonResponse(body: unknown, status = 200): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

function event(fetchImpl: typeof fetch, path = '', token: string | null = 'token-123') {
	const url = new URL('http://localhost/admin/fs');
	if (path) {
		url.searchParams.set('path', path);
	}
	return {
		url,
		locals: { sessionToken: token },
		fetch: fetchImpl,
		cookies: {},
		params: {},
		route: { id: '/admin/fs' },
		request: new Request(url),
		setHeaders: vi.fn(),
		getClientAddress: () => '127.0.0.1',
		isDataRequest: false,
		isSubRequest: false,
		platform: undefined
	};
}

describe('endpoint /admin/fs', () => {
	it('devuelve las raíces con no-store', async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			jsonResponse({ roots: [{ name: 'media', path: '/media' }] })
		);

		const response = await GET(event(fetchMock) as never);
		const body = (await response.json()) as { roots: unknown[] };

		expect(response.status).toBe(200);
		expect(response.headers.get('Cache-Control')).toBe('no-store');
		expect(body.roots).toEqual([{ name: 'media', path: '/media' }]);
		expect(fetchMock.mock.calls[0][0]).toBe('http://api.test/api/v1/fs');
		expect((fetchMock.mock.calls[0][1]?.headers as Record<string, string>).Authorization).toBe(
			'Bearer token-123'
		);
	});

	it('reenvía la ruta codificada a la API', async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			jsonResponse({
				path: '/media/pelis con espacio',
				parent: '/media',
				entries: [],
				truncated: false
			})
		);

		const response = await GET(event(fetchMock, '/media/pelis con espacio') as never);

		expect(response.status).toBe(200);
		expect(fetchMock.mock.calls[0][0]).toBe(
			'http://api.test/api/v1/fs?path=%2Fmedia%2Fpelis%20con%20espacio'
		);
	});

	it('sin sesión devuelve 401 sin llamar a la API', async () => {
		const fetchMock = vi.fn();

		const response = await GET(event(fetchMock, '', null) as never);

		expect(response.status).toBe(401);
		expect(response.headers.get('Cache-Control')).toBe('no-store');
		expect(fetchMock).not.toHaveBeenCalled();
	});

	it('reenvía 403 con su mensaje', async () => {
		const fetchMock = vi
			.fn()
			.mockResolvedValue(jsonResponse({ error: 'la ruta está fuera de las raíces permitidas' }, 403));

		const response = await GET(event(fetchMock, '/etc') as never);
		const body = (await response.json()) as { error: string };

		expect(response.status).toBe(403);
		expect(body.error).toContain('fuera de las raíces');
	});

	it('reenvía 404 con su mensaje', async () => {
		const fetchMock = vi
			.fn()
			.mockResolvedValue(jsonResponse({ error: 'la carpeta no existe' }, 404));

		const response = await GET(event(fetchMock, '/media/no') as never);
		const body = (await response.json()) as { error: string };

		expect(response.status).toBe(404);
		expect(body.error).toContain('no existe');
	});

	it('devuelve 502 para el resto de errores', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ error: 'fallo' }, 500));

		const response = await GET(event(fetchMock) as never);
		const body = (await response.json()) as { error: string };

		expect(response.status).toBe(502);
		expect(body.error).not.toBe('');
	});
});
