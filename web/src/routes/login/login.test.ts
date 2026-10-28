import { render } from 'svelte/server';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import LoginPage from './+page.svelte';
import { actions, load as loadLogin } from './+page.server';

const login = actions.login!;
const logout = actions.logout!;

function formRequest(
	username?: string,
	password?: string,
	userAgent?: string,
	next?: string
): Request {
	const body = new URLSearchParams();
	if (username !== undefined) body.set('username', username);
	if (password !== undefined) body.set('password', password);
	if (next !== undefined) body.set('next', next);
	return new Request('http://localhost/login?/login', {
		method: 'POST',
		headers: {
			'Content-Type': 'application/x-www-form-urlencoded',
			...(userAgent ? { 'User-Agent': userAgent } : {})
		},
		body
	});
}

function fakeEvent(options: {
	fetchImpl: typeof fetch;
	url?: string;
	username?: string;
	password?: string;
	userAgent?: string;
	next?: string;
	token?: string | null;
	clientIP?: string | null;
}) {
	const get = vi.fn(() => options.token ?? undefined);
	const set = vi.fn();
	const remove = vi.fn();
	const cookies = { get, set, delete: remove };

	const event = {
		request: formRequest(options.username, options.password, options.userAgent, options.next),
		cookies,
		url: new URL(
			options.url ??
				(options.next
					? `http://localhost/login?next=${encodeURIComponent(options.next)}`
					: 'http://localhost/login')
		),
		fetch: options.fetchImpl,
		locals: {},
		params: {},
		route: { id: '/login' },
		setHeaders: vi.fn(),
		...(options.clientIP === null
			? {}
			: {
					getClientAddress: () => {
						if (options.clientIP === 'throw') {
							throw new Error('sin dirección');
						}
						return options.clientIP ?? '127.0.0.1';
					}
				}),
		isDataRequest: false,
		isSubRequest: false,
		platform: undefined
	};

	return { event, cookies, set, remove };
}

function loginOk(
	body: unknown = { token: 'token-123', expires_at: '2026-10-26T12:00:00Z', session_id: 1 }
) {
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
		expect(fetchMock.mock.calls[0][1].headers['X-Forwarded-For']).toBe('127.0.0.1');
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

	it('reenvía el User-Agent del navegador al servidor', async () => {
		const fetchMock = loginOk();
		const userAgent =
			'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/130.0 Safari/537.36';
		const { event } = fakeEvent({
			fetchImpl: fetchMock,
			username: 'ana',
			password: 'Clave-segura-1',
			userAgent
		});

		await expect(login(event as never)).rejects.toMatchObject({ status: 303, location: '/' });
		expect(fetchMock.mock.calls[0][1].headers['User-Agent']).toBe(userAgent);
	});

	it('vuelve a la ruta interna indicada en next', async () => {
		const { event } = fakeEvent({
			fetchImpl: loginOk(),
			username: 'ana',
			password: 'Clave-segura-1',
			next: '/vincular?codigo=AB'
		});

		await expect(login(event as never)).rejects.toMatchObject({
			status: 303,
			location: '/vincular?codigo=AB'
		});
	});

	it('usa next del campo oculto si la acción ya no lleva la query', async () => {
		const { event } = fakeEvent({
			fetchImpl: loginOk(),
			url: 'http://localhost/login',
			username: 'ana',
			password: 'Clave-segura-1',
			next: '/vincular?codigo=AB'
		});

		await expect(login(event as never)).rejects.toMatchObject({
			status: 303,
			location: '/vincular?codigo=AB'
		});
	});

	it('ignora destinos externos o inseguros', async () => {
		for (const next of ['//evil.com', '/\\evil.com', 'https://evil.com', 'javascript:alert(1)']) {
			const { event } = fakeEvent({
				fetchImpl: loginOk(),
				username: 'ana',
				password: 'Clave-segura-1',
				next
			});
			await expect(login(event as never)).rejects.toMatchObject({
				status: 303,
				location: '/'
			});
		}
	});

	it('mantiene en el mismo origen una ruta codificada con doble barra', async () => {
		const { event } = fakeEvent({
			fetchImpl: loginOk(),
			username: 'ana',
			password: 'Clave-segura-1',
			next: '/%2F%2Fevil.com'
		});
		await expect(login(event as never)).rejects.toMatchObject({
			status: 303,
			location: '/%2F%2Fevil.com'
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

	it('sigue funcionando si getClientAddress no está disponible', async () => {
		const fetchMock = apiError(401);
		const { event } = fakeEvent({
			fetchImpl: fetchMock,
			username: 'ana',
			password: 'incorrecta',
			clientIP: 'throw'
		});

		const result = (await login(event as never)) as { status: number };
		expect(result.status).toBe(401);
		expect(fetchMock.mock.calls[0][1].headers['X-Forwarded-For']).toBeUndefined();
	});

	it('el quinto fallo muestra el bloqueo y una IP distinta puede entrar', async () => {
		let attempts = 0;
		const fetchMock = vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
			const headers = init?.headers as Record<string, string>;
			if (headers['X-Forwarded-For'] === '203.0.113.8') {
				return new Response(JSON.stringify({ token: 'token-bea', expires_at: '2026-10-26T12:00:00Z' }), {
					status: 200,
					headers: { 'Content-Type': 'application/json' }
				});
			}
			attempts++;
			if (attempts >= 5) {
				return new Response(JSON.stringify({ error: 'bloqueado' }), {
					status: 429,
					headers: { 'Content-Type': 'application/json' }
				});
			}
			return new Response(JSON.stringify({ error: 'credenciales' }), {
				status: 401,
				headers: { 'Content-Type': 'application/json' }
			});
		});

		for (let index = 0; index < 4; index++) {
			const { event } = fakeEvent({
				fetchImpl: fetchMock as never,
				username: 'ana',
				password: 'mala',
				clientIP: '203.0.113.7'
			});
			const result = (await login(event as never)) as { status: number };
			expect(result.status).toBe(401);
		}
		const fifth = fakeEvent({
			fetchImpl: fetchMock as never,
			username: 'ana',
			password: 'mala',
			clientIP: '203.0.113.7'
		});
		const blocked = (await login(fifth.event as never)) as {
			status: number;
			data: { message: string };
		};
		expect(blocked.status).toBe(429);
		expect(blocked.data.message).toContain('Demasiados intentos');

		const other = fakeEvent({
			fetchImpl: fetchMock as never,
			username: 'bea',
			password: 'Clave-segura-2',
			clientIP: '203.0.113.8'
		});
		await expect(login(other.event as never)).rejects.toMatchObject({ status: 303, location: '/' });
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

	it('conserva next en el campo oculto y descarta destinos inseguros', async () => {
		const safe = (await loadLogin({
			locals: { sessionToken: null },
			url: new URL('http://localhost/login?next=%2Fvincular%3Fcodigo%3DAB')
		} as never)) as { next: string };
		expect(safe.next).toBe('/vincular?codigo=AB');

		const unsafe = (await loadLogin({
			locals: { sessionToken: null },
			url: new URL('http://localhost/login?next=%2F%2Fevil.com')
		} as never)) as { next: string };
		expect(unsafe.next).toBe('');

		const page = render(LoginPage, {
			props: { data: { hasSession: false, next: '/vincular?codigo=AB' } }
		}).body;
		expect(page).toContain('name="next"');
		expect(page).toContain('value="/vincular?codigo=AB"');
	});
});
