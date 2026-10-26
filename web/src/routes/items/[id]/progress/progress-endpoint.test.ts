import { describe, expect, it, vi } from 'vitest';
import { POST } from './+server';

function jsonResponse(body: unknown, status = 200): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

function endpointEvent(
	fetchImpl: typeof fetch,
	body: string,
	contentType = 'application/json',
	token: string | null = 'token-123'
) {
	return {
		params: { id: '7' },
		request: new Request('http://localhost/items/7/progress', {
			method: 'POST',
			headers: { 'Content-Type': contentType },
			body
		}),
		locals: { sessionToken: token },
		fetch: fetchImpl
	};
}

describe('endpoint web de progreso', () => {
	it('reenvía el JSON a la API y responde 204', async () => {
		const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
		const response = await POST(
			endpointEvent(fetchMock, JSON.stringify({ position_seconds: 120, duration_seconds: 600 })) as never
		);

		expect(response.status).toBe(204);
		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe('http://api.test/api/v1/items/7/progress');
		expect(init.method).toBe('PUT');
		expect(init.headers.Authorization).toBe('Bearer token-123');
		expect(JSON.parse(init.body)).toEqual({ position_seconds: 120, duration_seconds: 600 });
	});

	it('acepta el cuerpo text/plain de sendBeacon', async () => {
		const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
		const response = await POST(
			endpointEvent(
				fetchMock,
				JSON.stringify({ position_seconds: 30, duration_seconds: 600 }),
				'text/plain'
			) as never
		);

		expect(response.status).toBe(204);
		expect(fetchMock).toHaveBeenCalledTimes(1);
	});

	it('rechaza cuerpos o valores no válidos sin llamar a la API', async () => {
		const bodies = [
			'no es json',
			JSON.stringify({ position_seconds: -1, duration_seconds: 10 }),
			JSON.stringify({ position_seconds: 10, duration_seconds: -1 }),
			JSON.stringify({ position_seconds: '10', duration_seconds: 10 }),
			JSON.stringify({ position_seconds: 10 })
		];

		for (const body of bodies) {
			const fetchMock = vi.fn();
			const response = await POST(endpointEvent(fetchMock, body) as never);
			expect(response.status).toBe(400);
			expect(fetchMock).not.toHaveBeenCalled();
		}
	});

	it('responde 401 sin sesión', async () => {
		const fetchMock = vi.fn();
		const response = await POST(
			endpointEvent(
				fetchMock,
				JSON.stringify({ position_seconds: 10, duration_seconds: 100 }),
				'application/json',
				null
			) as never
		);

		expect(response.status).toBe(401);
		expect(fetchMock).not.toHaveBeenCalled();
	});

	it('propaga el 404 de la API', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ error: 'no existe' }, 404));
		const response = await POST(
			endpointEvent(fetchMock, JSON.stringify({ position_seconds: 10, duration_seconds: 100 })) as never
		);

		expect(response.status).toBe(404);
	});

	it('convierte otros errores de la API en 502', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ error: 'fallo' }, 500));
		const response = await POST(
			endpointEvent(fetchMock, JSON.stringify({ position_seconds: 10, duration_seconds: 100 })) as never
		);

		expect(response.status).toBe(502);
	});
});
