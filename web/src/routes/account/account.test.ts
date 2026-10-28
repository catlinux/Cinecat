import { render } from 'svelte/server';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import AccountPage from './+page.svelte';
import { actions } from './+page.server';

const changePassword = actions.default!;

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
		cookies: {},
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
});
