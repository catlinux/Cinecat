import { describe, expect, it, vi } from 'vitest';
import type { UpdateInfo } from '$lib/api';
import { GET } from './+server';

function jsonResponse(body: unknown, status = 200): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

function endpointEvent(fetchImpl: typeof fetch, token: string | null = 'token-123') {
	return {
		locals: { sessionToken: token },
		fetch: fetchImpl
	};
}

const updateInfo: UpdateInfo = {
	current: '0.15.0',
	latest: 'v0.16.0',
	available: true,
	checked_at: '2026-10-27T10:00:00Z',
	notes: 'Novedades',
	apply_supported: true,
	check_enabled: true,
	status: {
		phase: '',
		tag: '',
		message: '',
		started_at: '',
		finished_at: '',
		result: ''
	}
};

describe('endpoint web del estado de actualización', () => {
	it('devuelve el estado sin caché', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse(updateInfo));
		const response = await GET(endpointEvent(fetchMock) as never);

		expect(response.status).toBe(200);
		expect(response.headers.get('Cache-Control')).toBe('no-store');
		await expect(response.json()).resolves.toEqual(updateInfo);
		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe('http://api.test/api/v1/system/update');
		expect(init.headers.Authorization).toBe('Bearer token-123');
	});

	it('responde 401 sin sesión sin llamar a la API', async () => {
		const fetchMock = vi.fn();
		const response = await GET(endpointEvent(fetchMock, null) as never);

		expect(response.status).toBe(401);
		expect(response.headers.get('Cache-Control')).toBe('no-store');
		expect(fetchMock).not.toHaveBeenCalled();
	});

	it('conserva los códigos de sesión y de recurso', async () => {
		for (const status of [401, 403, 404]) {
			const fetchMock = vi
				.fn()
				.mockResolvedValue(jsonResponse({ error: `error ${status}` }, status));
			const response = await GET(endpointEvent(fetchMock) as never);
			expect(response.status).toBe(status);
			expect(response.headers.get('Cache-Control')).toBe('no-store');
		}
	});

	it('convierte los fallos de la API en 502 con un mensaje claro', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ error: 'fallo interno' }, 500));
		const response = await GET(endpointEvent(fetchMock) as never);

		expect(response.status).toBe(502);
		expect(response.headers.get('Cache-Control')).toBe('no-store');
		await expect(response.json()).resolves.toEqual({
			error: 'No se pudo consultar el estado de la actualización.'
		});
	});
});
