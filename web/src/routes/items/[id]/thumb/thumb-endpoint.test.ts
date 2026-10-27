import { describe, expect, it, vi } from 'vitest';
import { GET } from './+server';

function event(
	fetchImpl: typeof fetch,
	token: string | null = 'token-123',
	headers: Record<string, string> = {}
) {
	return {
		params: { id: '7' },
		locals: { sessionToken: token },
		fetch: fetchImpl,
		request: new Request('http://localhost/items/7/thumb', { headers }),
		url: new URL('http://localhost/items/7/thumb'),
		cookies: {},
		route: { id: '/items/[id]/thumb' },
		setHeaders: vi.fn(),
		getClientAddress: () => '127.0.0.1',
		isDataRequest: false,
		isSubRequest: false,
		platform: undefined
	};
}

describe('endpoint /items/[id]/thumb', () => {
	it('devuelve la imagen y conserva ETag y caché', async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			new Response(new Uint8Array([0xff, 0xd8, 0xff]), {
				status: 200,
				headers: {
					'Content-Type': 'image/jpeg',
					ETag: '"thumb-1"',
					'Cache-Control': 'private, max-age=86400'
				}
			})
		);

		const response = await GET(event(fetchMock) as never);

		expect(response.status).toBe(200);
		expect(response.headers.get('Content-Type')).toBe('image/jpeg');
		expect(response.headers.get('Cache-Control')).toBe('private, max-age=86400');
		expect(response.headers.get('ETag')).toBe('"thumb-1"');
		expect(new Uint8Array(await response.arrayBuffer())).toEqual(
			new Uint8Array([0xff, 0xd8, 0xff])
		);
		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe('http://api.test/api/v1/items/7/thumbnail');
		expect((init?.headers as Record<string, string>).Authorization).toBe('Bearer token-123');
	});

	it('reenvía If-None-Match y devuelve 304', async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			new Response(null, { status: 304, headers: { ETag: '"thumb-1"' } })
		);

		const response = await GET(
			event(fetchMock, 'token-123', { 'if-none-match': '"thumb-1"' }) as never
		);

		expect(response.status).toBe(304);
		expect(response.headers.get('ETag')).toBe('"thumb-1"');
		expect(
			(fetchMock.mock.calls[0][1]?.headers as Record<string, string>)['If-None-Match']
		).toBe('"thumb-1"');
	});

	it('sin sesión devuelve 401 sin llamar a la API', async () => {
		const fetchMock = vi.fn();

		const response = await GET(event(fetchMock, null) as never);

		expect(response.status).toBe(401);
		expect(fetchMock).not.toHaveBeenCalled();
	});

	it('ante 404 o 503 devuelve un PNG transparente cacheado', async () => {
		for (const status of [401, 404, 503, 500]) {
			const fetchMock = vi.fn().mockResolvedValue(
				new Response(JSON.stringify({ error: 'sin miniatura' }), {
					status,
					headers: { 'Content-Type': 'application/json' }
				})
			);

			const response = await GET(event(fetchMock) as never);
			const bytes = new Uint8Array(await response.arrayBuffer());

			expect(response.status).toBe(200);
			expect(response.headers.get('Content-Type')).toBe('image/png');
			expect(response.headers.get('Cache-Control')).toBe('private, max-age=60');
			expect(Array.from(bytes.slice(0, 4))).toEqual([0x89, 0x50, 0x4e, 0x47]);
		}
	});

	it('ante un fallo de red también devuelve el PNG transparente', async () => {
		const fetchMock = vi.fn().mockRejectedValue(new Error('sin conexión'));

		const response = await GET(event(fetchMock) as never);

		expect(response.status).toBe(200);
		expect(response.headers.get('Content-Type')).toBe('image/png');
	});
});
