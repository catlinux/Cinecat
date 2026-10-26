import { render } from 'svelte/server';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import ScanProgress from '$lib/components/ScanProgress.svelte';
import type { Library, ScanStatus } from '$lib/api';
import AdminPage from './+page.svelte';
import { actions, load } from './+page.server';
import { GET as getScanStatus } from './status/+server';

const create = actions.create!;
const remove = actions.delete!;
const scan = actions.scan!;

function jsonResponse(body: unknown, status = 200): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

function loadEvent(fetchImpl: typeof fetch) {
	return {
		locals: { sessionToken: 'token-123' },
		fetch: fetchImpl,
		cookies: {},
		url: new URL('http://localhost/admin/libraries'),
		params: {},
		route: { id: '/admin/libraries' },
		request: new Request('http://localhost/admin/libraries'),
		setHeaders: vi.fn(),
		getClientAddress: () => '127.0.0.1',
		isDataRequest: false,
		isSubRequest: false,
		platform: undefined
	};
}

function actionEvent(
	fetchImpl: typeof fetch,
	fields: Record<string, string>,
	token: string | null = 'token-123'
) {
	return {
		request: new Request('http://localhost/admin/libraries', {
			method: 'POST',
			headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
			body: new URLSearchParams(fields)
		}),
		locals: { sessionToken: token },
		fetch: fetchImpl,
		cookies: {},
		url: new URL('http://localhost/admin/libraries'),
		params: {},
		route: { id: '/admin/libraries' },
		setHeaders: vi.fn(),
		getClientAddress: () => '127.0.0.1',
		isDataRequest: false,
		isSubRequest: false,
		platform: undefined
	};
}

const libraries: Library[] = [
	{ id: 1, name: 'Peliculas', path: '/media/peliculas' },
	{ id: 2, name: 'Series', path: '/media/series' }
];

function adminFetch(libs: Library[] = libraries) {
	return vi.fn(async (input: RequestInfo | URL) => {
		const url = String(input);
		if (url.endsWith('/api/v1/auth/me')) {
			return jsonResponse({ username: 'ana', role: 'admin' });
		}
		if (url.endsWith('/api/v1/libraries')) {
			return jsonResponse({ libraries: libs });
		}
		throw new Error(`URL inesperada: ${url}`);
	});
}

describe('carga de /admin/libraries', () => {
	it('con admin carga las bibliotecas', async () => {
		const fetchMock = adminFetch();

		const result = (await load(loadEvent(fetchMock) as never)) as { libraries: Library[] };

		expect(result.libraries).toHaveLength(2);
		expect(result.libraries[0].name).toBe('Peliculas');
		expect(fetchMock).toHaveBeenCalledTimes(2);
		expect(fetchMock.mock.calls[0][0]).toBe('http://api.test/api/v1/auth/me');
		expect(fetchMock.mock.calls[1][0]).toBe('http://api.test/api/v1/libraries');
	});

	it('con usuario no admin devuelve 403 y no pide la lista', async () => {
		const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
			const url = String(input);
			if (url.endsWith('/api/v1/auth/me')) {
				return jsonResponse({ username: 'pepe', role: 'user' });
			}
			throw new Error(`URL inesperada: ${url}`);
		});

		await expect(load(loadEvent(fetchMock) as never)).rejects.toMatchObject({
			status: 403
		});
		expect(fetchMock).toHaveBeenCalledTimes(1);
	});
});

describe('acciones de /admin/libraries', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	it('crear una biblioteca válida llama a la API y la lista se refresca', async () => {
		const created: Library = { id: 3, name: 'Documentales', path: '/media/docs' };
		const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
			const url = String(input);
			if (url.endsWith('/api/v1/libraries') && init?.method === 'POST') {
				return jsonResponse(created, 201);
			}
			throw new Error(`URL inesperada: ${url}`);
		});

		const result = (await create(
			actionEvent(fetchMock, { name: 'Documentales', path: '/media/docs' }) as never
		)) as { success?: boolean; message?: string };

		expect(result.success).toBe(true);
		expect(result.message).toContain('Biblioteca creada');
		expect(JSON.parse(fetchMock.mock.calls[0][1]?.body as string)).toEqual({
			name: 'Documentales',
			path: '/media/docs'
		});
		expect(fetchMock.mock.calls[0][1]?.headers).toMatchObject({
			Authorization: 'Bearer token-123'
		});

		// SvelteKit vuelve a ejecutar el load tras una acción correcta.
		const refreshed = (await load(
			loadEvent(adminFetch([...libraries, created])) as never
		)) as { libraries: Library[] };
		expect(refreshed.libraries.map((library) => library.name)).toContain('Documentales');
	});

	it('crear con una ruta inválida muestra el mensaje de la API', async () => {
		const fetchMock = vi
			.fn()
			.mockResolvedValue(jsonResponse({ error: 'la ruta no existe o no es accesible' }, 400));

		const result = (await create(
			actionEvent(fetchMock, { name: 'Rota', path: '/no/existe' }) as never
		)) as { status: number; data: { message: string; name?: string; path?: string } };

		expect(result.status).toBe(400);
		expect(result.data.message).toContain('la ruta no existe');
		expect(result.data.name).toBe('Rota');
		expect(result.data.path).toBe('/no/existe');
	});

	it('crear sin nombre ni ruta no llama a la API', async () => {
		const fetchMock = vi.fn();

		const result = (await create(
			actionEvent(fetchMock, { name: '', path: '' }) as never
		)) as { status: number; data: { message: string } };

		expect(result.status).toBe(400);
		expect(result.data.message).toContain('Escribe un nombre y una ruta');
		expect(fetchMock).not.toHaveBeenCalled();
	});

	it('borrar llama a deleteLibrary con el id', async () => {
		const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));

		const result = (await remove(actionEvent(fetchMock, { id: '2' }) as never)) as {
			success?: boolean;
			message?: string;
		};

		expect(result.success).toBe(true);
		expect(result.message).toContain('Biblioteca borrada');
		expect(fetchMock.mock.calls[0][0]).toBe('http://api.test/api/v1/libraries/2');
		expect(fetchMock.mock.calls[0][1]?.method).toBe('DELETE');
	});

	it('escanear inicia el trabajo y devuelve started', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ started: true }, 202));

		const result = (await scan(actionEvent(fetchMock, {}) as never)) as {
			success?: boolean;
			started?: boolean;
			message?: string;
		};

		expect(result.success).toBe(true);
		expect(result.started).toBe(true);
		expect(result.message).toBe('Escaneo iniciado.');
		expect(fetchMock.mock.calls[0][0]).toBe('http://api.test/api/v1/libraries/scan');
		expect(fetchMock.mock.calls[0][1]?.method).toBe('POST');
	});

	it('escanear con 409 lo trata como escaneo en curso', async () => {
		const fetchMock = vi
			.fn()
			.mockResolvedValue(jsonResponse({ error: 'ya hay un escaneo en curso' }, 409));

		const result = (await scan(actionEvent(fetchMock, {}) as never)) as {
			success?: boolean;
			started?: boolean;
			message?: string;
		};

		expect(result.success).toBe(true);
		expect(result.started).toBe(true);
		expect(result.message).toBe('Ya hay un escaneo en curso.');
	});

	it('escanear con 403 devuelve forbidden', async () => {
		const fetchMock = vi
			.fn()
			.mockResolvedValue(jsonResponse({ error: 'se requiere rol de administrador' }, 403));

		const result = (await scan(actionEvent(fetchMock, {}) as never)) as {
			status: number;
			data: { message: string };
		};

		expect(result.status).toBe(403);
		expect(result.data.message).toContain('administradores');
	});

	it('escanear con 401 redirige al login', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ error: 'sesión' }, 401));

		await expect(scan(actionEvent(fetchMock, {}) as never)).rejects.toMatchObject({
			status: 303,
			location: '/login'
		});
	});
});

describe('estado del escaneo', () => {
	const running: ScanStatus = {
		running: true,
		started_at: '2026-10-26T08:00:00Z',
		finished_at: null,
		libraries: 2,
		files: 12,
		upserted: 10,
		identified: 8,
		unidentified: 2,
		warnings: [],
		error: ''
	};

	it('GET devuelve el estado del escaneo', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse(running));

		const response = await getScanStatus(actionEvent(fetchMock, {}) as never);

		expect(response.status).toBe(200);
		expect(await response.json()).toEqual(running);
		expect(fetchMock.mock.calls[0][0]).toBe('http://api.test/api/v1/libraries/scan');
		expect(fetchMock.mock.calls[0][1]?.headers).toMatchObject({
			Authorization: 'Bearer token-123'
		});
	});

	it('GET propaga el 401 de la API', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ error: 'sesión' }, 401));

		const response = await getScanStatus(actionEvent(fetchMock, {}) as never);

		expect(response.status).toBe(401);
	});

	it('GET propaga el 403 de la API', async () => {
		const fetchMock = vi
			.fn()
			.mockResolvedValue(jsonResponse({ error: 'se requiere rol de administrador' }, 403));

		const response = await getScanStatus(actionEvent(fetchMock, {}) as never);

		expect(response.status).toBe(403);
	});

	it('GET devuelve 502 si falla la API', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ error: 'fallo' }, 500));

		const response = await getScanStatus(actionEvent(fetchMock, {}) as never);
		const body = (await response.json()) as { message: string };

		expect(response.status).toBe(502);
		expect(body.message).not.toBe('');
	});

	it('GET sin sesión devuelve 401 sin llamar a la API', async () => {
		const fetchMock = vi.fn();

		const response = await getScanStatus(actionEvent(fetchMock, {}, null) as never);

		expect(response.status).toBe(401);
		expect(fetchMock).not.toHaveBeenCalled();
	});
});

describe('página de administración', () => {
	it('muestra bibliotecas, formulario y controles del escaneo', () => {
		const { body } = render(AdminPage, {
			props: {
				data: { libraries },
				form: {
					success: true,
					started: true,
					message: 'Escaneo iniciado.'
				}
			}
		});

		expect(body).toContain('Peliculas');
		expect(body).toContain('/media/peliculas');
		expect(body).toContain('Nueva biblioteca');
		expect(body).toContain('Escanear todas las bibliotecas');
		expect(body).toContain('Escaneo iniciado.');
		expect(body).toContain('Escaneo en curso…');
		expect(body).toContain('disabled');
	});
});

describe('progreso del escaneo', () => {
	it('muestra el aviso de escaneo en curso', () => {
		const { body } = render(ScanProgress, {
			props: {
				status: {
					running: true,
					started_at: '2026-10-26T08:00:00Z',
					finished_at: null,
					libraries: 0,
					files: 0,
					upserted: 0,
					identified: 0,
					unidentified: 0,
					warnings: [],
					error: ''
				}
			}
		});

		expect(body).toContain('Escaneo en curso…');
		expect(body).toContain('role="status"');
	});

	it('muestra cifras, avisos plegables y el error', () => {
		const { body } = render(ScanProgress, {
			props: {
				status: {
					running: false,
					started_at: '2026-10-26T08:00:00Z',
					finished_at: '2026-10-26T08:05:00Z',
					libraries: 2,
					files: 12,
					upserted: 10,
					identified: 8,
					unidentified: 4,
					warnings: ['Ruta ilegible: /x'],
					error: 'El escaneo terminó con un error.'
				}
			}
		});

		expect(body).toContain('12');
		expect(body).toContain('archivos encontrados');
		expect(body).toContain('10');
		expect(body).toContain('entradas actualizadas');
		expect(body).toContain('8');
		expect(body).toContain('identificadas');
		expect(body).toContain('4');
		expect(body).toContain('sin identificar');
		expect(body).toContain('Avisos (1)');
		expect(body).toContain('Ruta ilegible: /x');
		expect(body).toContain('El escaneo terminó con un error.');
		expect(body).toContain('<details');
	});

	it('indica que no hay avisos', () => {
		const { body } = render(ScanProgress, {
			props: {
				status: {
					running: false,
					started_at: '2026-10-26T08:00:00Z',
					finished_at: '2026-10-26T08:05:00Z',
					libraries: 1,
					files: 0,
					upserted: 0,
					identified: 0,
					unidentified: 0,
					warnings: [],
					error: ''
				}
			}
		});

		expect(body).toContain('Avisos (0)');
		expect(body).toContain('Sin avisos.');
	});
});
