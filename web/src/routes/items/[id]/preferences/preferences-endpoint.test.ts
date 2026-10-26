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
		request: new Request('http://localhost/items/7/preferences', {
			method: 'POST',
			headers: { 'Content-Type': contentType },
			body
		}),
		locals: { sessionToken: token },
		fetch: fetchImpl
	};
}

describe('endpoint web de preferencias', () => {
	it('reenvía el JSON a la API y responde 204', async () => {
		const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
		const response = await POST(
			endpointEvent(
				fetchMock,
				JSON.stringify({ audio_index: 1, subtitle_id: 'e0', quality: '720' })
			) as never
		);

		expect(response.status).toBe(204);
		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe('http://api.test/api/v1/items/7/preferences');
		expect(init.method).toBe('PUT');
		expect(init.headers.Authorization).toBe('Bearer token-123');
		expect(JSON.parse(init.body)).toEqual({ audio_index: 1, subtitle_id: 'e0', quality: '720' });
	});

	it('acepta el cuerpo text/plain de sendBeacon', async () => {
		const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
		const response = await POST(
			endpointEvent(fetchMock, JSON.stringify({ subtitle_id: '' }), 'text/plain') as never
		);

		expect(response.status).toBe(204);
		expect(fetchMock).toHaveBeenCalledTimes(1);
	});

	it('acepta null para borrar un campo', async () => {
		const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
		const response = await POST(
			endpointEvent(fetchMock, JSON.stringify({ audio_index: null, quality: null })) as never
		);

		expect(response.status).toBe(204);
		expect(JSON.parse(fetchMock.mock.calls[0][1].body)).toEqual({
			audio_index: null,
			quality: null
		});
	});

	it('rechaza cuerpos o valores no válidos sin llamar a la API', async () => {
		const bodies = [
			'no es json',
			JSON.stringify({ audio_index: -1 }),
			JSON.stringify({ quality: '4k' }),
			JSON.stringify({ subtitle_id: 'e0.srt' }),
			JSON.stringify({ desconocido: true })
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
		const response = await POST(endpointEvent(fetchMock, '{}', 'application/json', null) as never);
		expect(response.status).toBe(401);
		expect(fetchMock).not.toHaveBeenCalled();
	});

	it('propaga 404 y convierte el resto en 502', async () => {
		const missing = vi.fn().mockResolvedValue(jsonResponse({ error: 'no existe' }, 404));
		const missingResponse = await POST(
			endpointEvent(missing, JSON.stringify({ audio_index: 0 })) as never
		);
		expect(missingResponse.status).toBe(404);

		const broken = vi.fn().mockResolvedValue(jsonResponse({ error: 'fallo' }, 500));
		const brokenResponse = await POST(
			endpointEvent(broken, JSON.stringify({ audio_index: 0 })) as never
		);
		expect(brokenResponse.status).toBe(502);
	});
});
