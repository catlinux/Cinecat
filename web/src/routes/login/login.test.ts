import { render } from 'svelte/server';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import LoginPage from './+page.svelte';
import { actions } from './+page.server';

const login = actions.login!;
const logout = actions.logout!;

function formRequest(username?: string, password?: string): Request {
	const body = new URLSearchParams();
	if (username !== undefined) body.set('username', username);
	if (password !== undefined) body.set('password', password);
	return new Request('http://localhost/login?/login', {
		method: 'POST',
		headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
		body
	});
}

function fakeEvent(options: {
	fetchImpl: typeof fetch;
	url?: string;
	username?: string;
	password?: string;
	token?: string | null;
}) {
	const get = vi.fn(() => options.token ?? undefined);
	const set = vi.fn();
	const remove = vi.fn();
	const cookies = { get, set, delete: remove };

	const event = {
		request: formRequest(options.username, options.password),
		cookies,
		url: new URL(options.url ?? 'http://localhost/login'),
		fetch: options.fetchImpl,
		locals: {},
		params: {},
		route: { id: '/login' },
		setHeaders: vi.fn(),
		getClientAddress: () => '127.0.0.1',
		isDataRequest: false,
		isSubRequest: false,
		platform: undefined
	};

	return { event, cookies, set, remove };
}

function loginOk(body: unknown = { token: 'token-123', expires_at: '2026-10-26T12:00:00Z' }) {
	return vi.fn().mockResolvedValue(
		new Response(JSON.stringify(body), {
			status: 200,
			headers: { 'Content-Type': 'application/json' }
		})
	);
}

function apiError(status: number) {
	return vi.fn().mockResolvedValue(
		new Response(JSON.stringify({ error: 'error del servidor' }), {
			status,
			headers: { 'Content-Type': 'application/json' }
		})
	);
}

describe('acción de login', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	it('fija la cookie httpOnly y redirige a la portada con credenciales correctas', async () => {
		const fetchMock = loginOk();
		const { event, set } = fakeEvent({
			fetchImpl: fetchMock,
			username: 'ana',
			password: 'Clave-segura-1'
		});

		await expect(login(event as never)).rejects.toMatchObject({ status: 303, location: '/' });

		expect(fetchMock).toHaveBeenCalledTimes(1);
		expect(fetchMock.mock.calls[0][0]).toBe('http://api.test/api/v1/auth/login');
		expect(set).toHaveBeenCalledTimes(1);
		const [name, value, options] = set.mock.calls[0];
		expect(name).toBe('cinecat_session');
		expect(value).toBe('token-123');
		expect(options).toMatchObject({
			path: '/',
			httpOnly: true,
			sameSite: 'lax',
			secure: false
		});
	});

	it('marca Secure en la cookie si la petición llegó por HTTPS', async () => {
		const { event, set } = fakeEvent({
			fetchImpl: loginOk(),
			url: 'https://localhost/login',
			username: 'ana',
			password: 'Clave-segura-1'
		});

		await expect(login(event as never)).rejects.toMatchObject({ status: 303 });
		expect(set.mock.calls[0][2].secure).toBe(true);
	});

	it('no fija cookie y muestra el mensaje de credenciales incorrectas con 401', async () => {
		const { event, set } = fakeEvent({
			fetchImpl: apiError(401),
			username: 'ana',
			password: 'incorrecta'
		});

		const result = (await login(event as never)) as {
			status: number;
			data: { message: string; username: string };
		};

		expect(result.status).toBe(401);
		expect(result.data.message).toContain('Usuario o contraseña incorrectos');
		expect(result.data.username).toBe('ana');
		expect(set).not.toHaveBeenCalled();
	});

	it('explica el bloqueo temporal con 429 y no fija cookie', async () => {
		const { event, set } = fakeEvent({
			fetchImpl: apiError(429),
			username: 'ana',
			password: 'Clave-segura-1'
		});

		const result = (await login(event as never)) as {
			status: number;
			data: { message: string };
		};

		expect(result.status).toBe(429);
		expect(result.data.message).toContain('Demasiados intentos fallidos');
		expect(result.data.message).toContain('Espera unos minutos');
		expect(set).not.toHaveBeenCalled();
	});

	it('valida los campos vacíos sin llamar a la API', async () => {
		const fetchMock = vi.fn();
		const { event } = fakeEvent({ fetchImpl: fetchMock, username: '', password: '' });

		const result = (await login(event as never)) as {
			status: number;
			data: { message: string };
		};

		expect(result.status).toBe(400);
		expect(result.data.message).toContain('Escribe tu usuario y contraseña');
		expect(fetchMock).not.toHaveBeenCalled();
	});
});

describe('acción de logout', () => {
	it('invalida la sesión en la API, borra la cookie y redirige al login', async () => {
		const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
		const { event, remove } = fakeEvent({ fetchImpl: fetchMock, token: 'token-123' });

		await expect(logout(event as never)).rejects.toMatchObject({ status: 303, location: '/login' });

		expect(fetchMock).toHaveBeenCalledTimes(1);
		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe('http://api.test/api/v1/auth/logout');
		expect(init.method).toBe('POST');
		expect(init.headers.Authorization).toBe('Bearer token-123');
		expect(remove).toHaveBeenCalledWith('cinecat_session', { path: '/', secure: false });
	});
});

describe('página de login', () => {
	it('muestra el mensaje de error y conserva el usuario', () => {
		const { body } = render(LoginPage, {
			props: {
				data: { hasSession: false },
				form: { username: 'ana', message: 'Usuario o contraseña incorrectos.' }
			}
		});

		expect(body).toContain('Usuario o contraseña incorrectos.');
		expect(body).toContain('value="ana"');
		expect(body).toContain('Entrar en la sala');
	});
});
