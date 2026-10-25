import { render } from 'svelte/server';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { Library } from '$lib/api';
import AdminPage from './+page.svelte';
import { actions, load } from './+page.server';

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

function actionEvent(fetchImpl: typeof fetch, fields: Record<string, string>) {
	return {
		request: new Request('http://localhost/admin/libraries', {
			method: 'POST',
			headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
			body: new URLSearchParams(fields)
		}),
		locals: { sessionToken: 'token-123' },
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

	it('escanear devuelve el resultado', async () => {
		const scanResult = { libraries: 2, files: 12, upserted: 12, warnings: [] };
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse(scanResult));

		const result = (await scan(actionEvent(fetchMock, {}) as never)) as {
			success?: boolean;
			scan?: typeof scanResult;
		};

		expect(result.success).toBe(true);
		expect(result.scan).toEqual(scanResult);
		expect(fetchMock.mock.calls[0][0]).toBe('http://api.test/api/v1/libraries/scan');
		expect(fetchMock.mock.calls[0][1]?.method).toBe('POST');
	});
});

describe('página de administración', () => {
	it('muestra bibliotecas, formulario y resultado del escaneo', () => {
		const { body } = render(AdminPage, {
			props: {
				data: { libraries },
				form: {
					success: true,
					message: 'Escaneo completado.',
					scan: { libraries: 2, files: 5, upserted: 5, warnings: ['Ruta ilegible: /x'] }
				}
			}
		});

		expect(body).toContain('Peliculas');
		expect(body).toContain('/media/peliculas');
		expect(body).toContain('Nueva biblioteca');
		expect(body).toContain('Escanear todas las bibliotecas');
		expect(body).toContain('archivos encontrados');
		expect(body).toContain('Ruta ilegible: /x');
	});
});
