import { describe, expect, it, vi } from 'vitest';
import { GET } from './+server';

function jsonResponse(body: unknown, status = 200): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

function event(fetchImpl: typeof fetch, token: string | null = 'token-123') {
	const url = new URL('http://localhost/series/upload-targets');
	return {
		url,
		locals: { sessionToken: token },
		fetch: fetchImpl,
		cookies: {},
		route: { id: '/series/upload-targets' },
		setHeaders: vi.fn(),
		getClientAddress: () => '127.0.0.1',
		isDataRequest: false,
		isSubRequest: false,
		platform: undefined
	};
}

describe('GET /series/upload-targets', () => {
	it('reenvía la sesión y devuelve los destinos', async () => {
		const body = {
			allowed: true,
			roots: [
				{
					library_id: 1,
					library_name: 'Series',
					library_kind: 'video',
					path: '/media/series',
					movie_count: 0,
					series_count: 4
				}
			]
		};
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse(body));

		const response = await GET(event(fetchMock as never) as never);
		expect(response.status).toBe(200);
		expect(response.headers.get('Cache-Control')).toBe('no-store');
		expect(await response.json()).toEqual(body);
		const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit];
		expect(url).toBe('http://api.test/api/v1/uploads/targets');
		expect(init.headers).toMatchObject({
			Authorization: 'Bearer token-123',
			Accept: 'application/json'
		});
	});

	it('exige sesión', async () => {
		const fetchMock = vi.fn();
		const response = await GET(event(fetchMock as never, null) as never);
		expect(response.status).toBe(401);
		expect(fetchMock).not.toHaveBeenCalled();
	});

	it('reenvía errores y devuelve 502 si la API no responde', async () => {
		const forbidden = vi.fn().mockResolvedValue(jsonResponse({ error: 'no admin' }, 403));
		const forbiddenResponse = await GET(event(forbidden as never) as never);
		expect(forbiddenResponse.status).toBe(403);
		expect(await forbiddenResponse.json()).toEqual({ error: 'no admin' });

		const broken = vi.fn().mockRejectedValue(new Error('sin conexión'));
		const brokenResponse = await GET(event(broken as never) as never);
		expect(brokenResponse.status).toBe(502);
		const payload = (await brokenResponse.json()) as { error: string };
		expect(payload.error).toContain('No se pudieron consultar');
	});
});
