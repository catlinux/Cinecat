import { render } from 'svelte/server';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import AccountPage from './+page.svelte';
import { actions, load } from './+page.server';

const changePassword = actions.default!;
const closeSession = actions.closeSession!;

function formRequest(fields: Record<string, string>): Request {
	return new Request('http://localhost/account', {
		method: 'POST',
		headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
		body: new URLSearchParams(fields)
	});
}

function fakeEvent(fetchImpl: typeof fetch, fields: Record<string, string>, token = 'token-123') {
	return {
		request: formRequest(fields),
		locals: { sessionToken: token },
		fetch: fetchImpl,
		cookies: { get: vi.fn(), set: vi.fn(), delete: vi.fn() },
		url: new URL('http://localhost/account'),
		params: {},
		route: { id: '/account' },
		setHeaders: vi.fn(),
		getClientAddress: () => '127.0.0.1',
		isDataRequest: false,
		isSubRequest: false,
		platform: undefined
	};
}

function loadEvent(fetchImpl: typeof fetch, token = 'token-123') {
	return {
		locals: { sessionToken: token },
		fetch: fetchImpl,
		url: new URL('http://localhost/account'),
		params: {},
		cookies: { get: vi.fn(), set: vi.fn(), delete: vi.fn() },
		route: { id: '/account' },
		setHeaders: vi.fn(),
		getClientAddress: () => '127.0.0.1',
		isDataRequest: false,
		isSubRequest: false,
		platform: undefined
	};
}

const validFields = {
	current_password: 'Clave-vieja-1',
	new_password: 'Clave-nueva-1',
	repeat_password: 'Clave-nueva-1'
};

describe('acción de cambio de contraseña', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	it('cambia la contraseña y devuelve un mensaje de éxito', async () => {
		const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
		const event = fakeEvent(fetchMock, validFields);

		const result = (await changePassword(event as never)) as {
			success?: boolean;
			message?: string;
		};

		expect(result.success).toBe(true);
		expect(result.message).toBe(
			'Contraseña actualizada. Se han cerrado tus sesiones en otros dispositivos; la de este sigue abierta.'
		);

		expect(fetchMock).toHaveBeenCalledTimes(1);
		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe('http://api.test/api/v1/auth/password');
		expect(init.method).toBe('POST');
		expect(init.headers.Authorization).toBe('Bearer token-123');
		expect(JSON.parse(init.body)).toEqual({
			current_password: 'Clave-vieja-1',
			new_password: 'Clave-nueva-1'
		});
	});

	it('no llama a la API si las contraseñas nuevas no coinciden', async () => {
		const fetchMock = vi.fn();
		const event = fakeEvent(fetchMock, {
			...validFields,
			repeat_password: 'Otra-clave-1'
		});

		const result = (await changePassword(event as never)) as {
			status: number;
			data: { message: string };
		};

		expect(result.status).toBe(400);
		expect(result.data.message).toContain('no coinciden');
		expect(fetchMock).not.toHaveBeenCalled();
	});

	it('no llama a la API si la contraseña nueva es demasiado corta', async () => {
		const fetchMock = vi.fn();
		const event = fakeEvent(fetchMock, {
			current_password: 'Clave-vieja-1',
			new_password: 'corta',
			repeat_password: 'corta'
		});

		const result = (await changePassword(event as never)) as {
			status: number;
			data: { message: string };
		};

		expect(result.status).toBe(400);
		expect(result.data.message).toContain('al menos 8 caracteres');
		expect(fetchMock).not.toHaveBeenCalled();
	});

	it('traduce el 401 de la API a un mensaje claro', async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			new Response(JSON.stringify({ error: 'la contraseña actual no es correcta' }), {
				status: 401,
				headers: { 'Content-Type': 'application/json' }
			})
		);
		const event = fakeEvent(fetchMock, validFields);

		const result = (await changePassword(event as never)) as {
			status: number;
			data: { message: string };
		};

		expect(result.status).toBe(401);
		expect(result.data.message).toContain('contraseña actual');
	});
});

describe('carga de sesiones', () => {
	it('devuelve las sesiones propias', async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			new Response(
				JSON.stringify({
					sessions: [
						{
							id: 7,
							device_name: 'Portátil',
							client: 'web',
							created_at: '2026-10-27T09:00:00Z',
							last_seen_at: '2026-10-27T10:00:00Z',
							expires_at: '2026-10-28T10:00:00Z',
							current: true
						}
					]
				}),
				{ status: 200, headers: { 'Content-Type': 'application/json' } }
			)
		);
		const result = (await load(loadEvent(fetchMock) as never)) as {
			sessions: Array<{ id: number }> | null;
		};
		expect(result.sessions).toHaveLength(1);
		expect(result.sessions?.[0].id).toBe(7);
	});

	it('no rompe la página si falla la lista', async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			new Response(JSON.stringify({ error: 'fallo' }), {
				status: 500,
				headers: { 'Content-Type': 'application/json' }
			})
		);
		const result = (await load(loadEvent(fetchMock) as never)) as { sessions: null };
		expect(result.sessions).toBeNull();
	});
});

describe('acción de cerrar sesión', () => {
	it('cierra una sesión que no es la actual', async () => {
		const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
		const event = fakeEvent(fetchMock, { id: '9', _current: '0' });

		const result = (await closeSession(event as never)) as {
			success?: boolean;
			message?: string;
		};

		expect(result.success).toBe(true);
		expect(result.message).toBe('Sesión cerrada.');
		expect(fetchMock.mock.calls[0][0]).toBe('http://api.test/api/v1/auth/sessions/9');
		expect(fetchMock.mock.calls[0][1].method).toBe('DELETE');
		expect(event.cookies.delete).not.toHaveBeenCalled();
	});

	it('traduce el 404 de una sesión inexistente', async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			new Response(JSON.stringify({ error: 'sesión no encontrada' }), {
				status: 404,
				headers: { 'Content-Type': 'application/json' }
			})
		);
		const event = fakeEvent(fetchMock, { id: '9', _current: '0' });
		const result = (await closeSession(event as never)) as {
			status: number;
			data: { message: string };
		};
		expect(result.status).toBe(404);
		expect(result.data.message).toContain('sesión no encontrada');
	});

	it('redirige al login y borra la cookie al cerrar la sesión actual', async () => {
		const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
			const url = new URL(String(input));
			if (url.pathname === '/api/v1/auth/sessions' && (init?.method ?? 'GET') === 'GET') {
				return new Response(
					JSON.stringify({
						sessions: [
							{
								id: 7,
								device_name: 'Portátil',
								client: 'web',
								created_at: '2026-10-27T09:00:00Z',
								last_seen_at: '2026-10-27T10:00:00Z',
								expires_at: '2026-10-28T10:00:00Z',
								current: true
							}
						]
					}),
					{ status: 200, headers: { 'Content-Type': 'application/json' } }
				);
			}
			if (url.pathname === '/api/v1/auth/sessions/7' && init?.method === 'DELETE') {
				return new Response(null, { status: 204 });
			}
			throw new Error(`URL inesperada: ${url} ${init?.method ?? 'GET'}`);
		});
		const event = fakeEvent(fetchMock, { id: '7' });

		await expect(closeSession(event as never)).rejects.toMatchObject({
			status: 303,
			location: '/login'
		});
		expect(event.cookies.delete).toHaveBeenCalledWith('cinecat_session', {
			path: '/',
			secure: false
		});
	});
});

describe('página de cuenta', () => {
	it('muestra el mensaje de éxito', () => {
		const { body } = render(AccountPage, {
			props: {
				form: {
					success: true,
					message:
						'Contraseña actualizada. Se han cerrado tus sesiones en otros dispositivos; la de este sigue abierta.'
				}
			}
		});

		expect(body).toContain('Se han cerrado tus sesiones en otros dispositivos');
		expect(body).toContain('action="/login?/logout"');
		expect(body).toContain('method="POST"');
		expect(body).toContain('Guardar contraseña');
		expect(body).toContain('Se cerrarán tus sesiones abiertas en otros dispositivos');
		expect(body).toContain('Cerrar sesión');
	});

	it('muestra las sesiones abiertas y la actual', () => {
		const { body } = render(AccountPage, {
			props: {
				data: {
					ui: { server_name: 'Cinecat' },
					sessions: [
						{
							id: 7,
							device_name: 'Portátil',
							client: 'web',
							created_at: '2026-10-27T09:00:00Z',
							last_seen_at: '2026-10-27T10:00:00Z',
							expires_at: '2026-10-28T10:00:00Z',
							current: true
						},
						{
							id: 8,
							device_name: '',
							client: 'android',
							created_at: '2026-10-27T09:00:00Z',
							last_seen_at: '2026-10-27T09:30:00Z',
							expires_at: '2027-01-25T09:00:00Z',
							current: false
						}
					]
				}
			}
		});
		expect(body).toContain('Sesiones abiertas');
		expect(body).toContain('Portátil');
		expect(body).toContain('Dispositivo sin nombre');
		expect(body).toContain('Este dispositivo');
		expect(body).toContain('Cerrar esta sesión');
		expect(body).toContain('action="?/closeSession"');
	});
});
