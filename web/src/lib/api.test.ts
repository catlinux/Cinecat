import { describe, expect, it, vi } from 'vitest';
import { ApiClient, ApiError } from './api';

function jsonResponse(body: unknown, status = 200): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

describe('ApiClient', () => {
	it('hace login y devuelve el token tipado', async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			jsonResponse({ token: 'token-123', expires_at: '2026-10-26T12:00:00Z' })
		);
		const client = new ApiClient({ baseUrl: 'http://api.test/', fetch: fetchMock });

		const session = await client.login({ username: 'ana', password: 'secreta' });

		expect(session).toEqual({ token: 'token-123', expires_at: '2026-10-26T12:00:00Z' });
		expect(fetchMock).toHaveBeenCalledTimes(1);
		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe('http://api.test/api/v1/auth/login');
		expect(init.method).toBe('POST');
		expect(init.body).toBe(JSON.stringify({ username: 'ana', password: 'secreta' }));
	});

	it('convierte un 401 en ApiError con estado', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ error: 'credenciales' }, 401));
		const client = new ApiClient({ baseUrl: 'http://api.test', fetch: fetchMock });

		await expect(client.login({ username: 'ana', password: 'mala' })).rejects.toMatchObject({
			name: 'ApiError',
			status: 401
		});
	});

	it('convierte un 429 en ApiError', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ error: 'bloqueado' }, 429));
		const client = new ApiClient({ baseUrl: 'http://api.test', fetch: fetchMock });

		await expect(client.login({ username: 'ana', password: 'mala' })).rejects.toBeInstanceOf(ApiError);
		await expect(client.login({ username: 'ana', password: 'mala' })).rejects.toMatchObject({
			status: 429
		});
	});

	it('manda el token Bearer en logout y acepta 204', async () => {
		const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
		const client = new ApiClient({ baseUrl: 'http://api.test', fetch: fetchMock });

		await expect(client.logout('token-123')).resolves.toBeUndefined();

		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe('http://api.test/api/v1/auth/logout');
		expect(init.headers.Authorization).toBe('Bearer token-123');
	});

	it('manda el token Bearer en llamadas autenticadas', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ libraries: [] }));
		const client = new ApiClient({ baseUrl: 'http://api.test', fetch: fetchMock });

		const data = await client.get<{ libraries: unknown[] }>('/api/v1/libraries', 'token-abc');

		expect(data.libraries).toEqual([]);
		expect(fetchMock.mock.calls[0][1].headers.Authorization).toBe('Bearer token-abc');
	});

	it('convierte un fallo de red en ApiError con estado 0', async () => {
		const fetchMock = vi.fn().mockRejectedValue(new Error('sin conexión'));
		const client = new ApiClient({ baseUrl: 'http://api.test', fetch: fetchMock });

		await expect(client.get('/api/v1/items')).rejects.toMatchObject({ status: 0 });
	});

	it('consulta, fuerza y solicita actualizaciones del sistema', async () => {
		const info = {
			current: '0.15.0',
			latest: 'v0.16.0',
			available: true,
			checked_at: '',
			notes: '',
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
		const fetchMock = vi.fn(async (_input: RequestInfo | URL, _init?: RequestInit) =>
			jsonResponse(info)
		);
		const client = new ApiClient({ baseUrl: 'http://api.test', fetch: fetchMock });

		await client.getUpdate('token-123');
		await client.checkUpdate('token-123');
		await client.applyUpdate('v0.16.0', 'token-123');

		expect(fetchMock.mock.calls.map(([url]) => String(url))).toEqual([
			'http://api.test/api/v1/system/update',
			'http://api.test/api/v1/system/update/check',
			'http://api.test/api/v1/system/update/apply'
		]);
		expect(fetchMock.mock.calls[1][1]?.method).toBe('POST');
		expect(JSON.parse(String(fetchMock.mock.calls[2][1]?.body))).toEqual({ tag: 'v0.16.0' });
		for (const [, init] of fetchMock.mock.calls) {
			expect((init?.headers as Record<string, string>).Authorization).toBe('Bearer token-123');
		}
	});
});
