import { render } from 'svelte/server';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import ScanProgress from '$lib/components/ScanProgress.svelte';
import type { Library, ScanStatus } from '$lib/api';
import AdminPage from './+page.svelte';
import { actions, load } from './+page.server';
import { GET as getScanStatus } from './status/+server';

const create = actions.create!;
const rename = actions.rename!;
const addPath = actions.addPath!;
const removePath = actions.removePath!;
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
	fields: Record<string, string | string[]>,
	token: string | null = 'token-123'
) {
	const body = new URLSearchParams();
	for (const [key, value] of Object.entries(fields)) {
		if (Array.isArray(value)) {
			for (const item of value) {
				body.append(key, item);
			}
		} else {
			body.set(key, value);
		}
	}
	return {
		request: new Request('http://localhost/admin/libraries', {
			method: 'POST',
			headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
			body
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
	{
		id: 1,
		name: 'Peliculas',
		kind: 'video',
		path: '/media/peliculas',
		paths: [
			{ id: 10, path: '/media/peliculas' },
			{ id: 11, path: '/mnt/otro-disco/peliculas' }
		],
		item_count: 120
	},
	{
		id: 2,
		name: 'Series',
		kind: 'video',
		path: '/media/series',
		paths: [{ id: 12, path: '/media/series' }],
		item_count: 30
	}
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
		expect(result.libraries[0].paths).toHaveLength(2);
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

	it('crear una biblioteca con una carpeta llama a la API', async () => {
		const created: Library = {
			id: 3,
			name: 'Documentales',
			kind: 'video',
			path: '/media/docs',
			paths: [{ id: 20, path: '/media/docs' }]
		};
		const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
			const url = String(input);
			if (url.endsWith('/api/v1/libraries') && init?.method === 'POST') {
				return jsonResponse(created, 201);
			}
			throw new Error(`URL inesperada: ${url}`);
		});

		const result = (await create(
			actionEvent(fetchMock, {
				name: 'Documentales',
				kind: 'video',
				paths: '/media/docs'
			}) as never
		)) as { success?: boolean; message?: string };

		expect(result.success).toBe(true);
		expect(result.message).toContain('Biblioteca creada');
		expect(JSON.parse(fetchMock.mock.calls[0][1]?.body as string)).toEqual({
			name: 'Documentales',
			kind: 'video',
			paths: ['/media/docs']
		});
		expect(fetchMock.mock.calls[0][1]?.headers).toMatchObject({
			Authorization: 'Bearer token-123'
		});
	});

	it('crear con varias carpetas manda todas y las conserva si falla', async () => {
		const fetchMock = vi
			.fn()
			.mockResolvedValue(jsonResponse({ error: 'la carpeta está solapada' }, 409));

		const result = (await create(
			actionEvent(fetchMock, {
				name: 'Cine',
				kind: 'video',
				paths: ['/media/a', '/media/b']
			}) as never
		)) as {
			status: number;
			data: { message: string; action: string; name: string; paths: string[] };
		};

		expect(JSON.parse(fetchMock.mock.calls[0][1]?.body as string)).toEqual({
			name: 'Cine',
			kind: 'video',
			paths: ['/media/a', '/media/b']
		});
		expect(result.status).toBe(409);
		expect(result.data.message).toContain('solapada');
		expect(result.data.action).toBe('create');
		expect(result.data.name).toBe('Cine');
		expect(result.data.paths).toEqual(['/media/a', '/media/b']);
	});

	it('crear con una ruta inválida muestra el mensaje de la API', async () => {
		const fetchMock = vi
			.fn()
			.mockResolvedValue(jsonResponse({ error: 'la ruta no existe o no es accesible' }, 400));

		const result = (await create(
			actionEvent(fetchMock, {
				name: 'Rota',
				kind: 'video',
				paths: '/no/existe'
			}) as never
		)) as {
			status: number;
			data: { message: string; name?: string; paths?: string[] };
		};

		expect(result.status).toBe(400);
		expect(result.data.message).toContain('la ruta no existe');
		expect(result.data.name).toBe('Rota');
		expect(result.data.paths).toEqual(['/no/existe']);
	});

	it('validaciones: sin nombre, nombre largo, tipo no permitido y sin carpetas', async () => {
		const fetchMock = vi.fn();

		const noName = (await create(
			actionEvent(fetchMock, { name: '', kind: 'video', paths: '/media/a' }) as never
		)) as { status: number; data: { message: string } };
		const longName = (await create(
			actionEvent(fetchMock, { name: 'x'.repeat(81), kind: 'video', paths: '/media/a' }) as never
		)) as { status: number; data: { message: string } };
		const badKind = (await create(
			actionEvent(fetchMock, { name: 'Cine', kind: 'otro', paths: '/media/a' }) as never
		)) as { status: number; data: { message: string } };
		const noPaths = (await create(
			actionEvent(fetchMock, { name: 'Cine', kind: 'video', paths: ['', '  '] }) as never
		)) as { status: number; data: { message: string } };

		expect(noName.status).toBe(400);
		expect(noName.data.message).toContain('nombre');
		expect(longName.status).toBe(400);
		expect(longName.data.message).toContain('80');
		expect(badKind.status).toBe(400);
		expect(badKind.data.message).toContain('Tipo');
		expect(noPaths.status).toBe(400);
		expect(noPaths.data.message).toContain('carpeta');
		expect(fetchMock).not.toHaveBeenCalled();
	});

	it('crear con 403 devuelve el mensaje de administradores y con 401 redirige', async () => {
		const forbidden = vi
			.fn()
			.mockResolvedValue(jsonResponse({ error: 'se requiere rol de administrador' }, 403));
		const result = (await create(
			actionEvent(forbidden, { name: 'Cine', kind: 'video', paths: '/media/a' }) as never
		)) as { status: number; data: { message: string } };

		expect(result.status).toBe(403);
		expect(result.data.message).toContain('administradores');

		const unauthorized = vi.fn().mockResolvedValue(jsonResponse({ error: 'sesión' }, 401));
		await expect(
			create(actionEvent(unauthorized, { name: 'Cine', kind: 'video', paths: '/media/a' }) as never)
		).rejects.toMatchObject({ status: 303, location: '/login' });
	});

	it('renombrar llama a PATCH y conserva el nombre si falla', async () => {
		const fetchMock = vi
			.fn()
			.mockResolvedValue(jsonResponse({ error: 'la biblioteca no existe' }, 404));

		const result = (await rename(
			actionEvent(fetchMock, { id: '4', name: 'Cine nuevo' }) as never
		)) as { status: number; data: { message: string; action: string; renameName: string } };

		expect(fetchMock.mock.calls[0][0]).toBe('http://api.test/api/v1/libraries/4');
		expect(fetchMock.mock.calls[0][1]?.method).toBe('PATCH');
		expect(JSON.parse(String(fetchMock.mock.calls[0][1]?.body))).toEqual({ name: 'Cine nuevo' });
		expect(result.status).toBe(404);
		expect(result.data.action).toBe('rename');
		expect(result.data.renameName).toBe('Cine nuevo');
	});

	it('renombrar valida id y nombre sin llamar a la API', async () => {
		const fetchMock = vi.fn();

		const badID = (await rename(
			actionEvent(fetchMock, { id: '0', name: 'Cine' }) as never
		)) as { status: number };
		const badName = (await rename(
			actionEvent(fetchMock, { id: '4', name: '' }) as never
		)) as { status: number; data: { message: string } };

		expect(badID.status).toBe(400);
		expect(badName.status).toBe(400);
		expect(badName.data.message).toContain('nombre');
		expect(fetchMock).not.toHaveBeenCalled();
	});

	it('añadir carpeta llama a la API y conserva la ruta si falla', async () => {
		const fetchMock = vi
			.fn()
			.mockResolvedValue(jsonResponse({ error: 'la carpeta ya está en otra biblioteca' }, 409));

		const result = (await addPath(
			actionEvent(fetchMock, { id: '4', path: '/media/nueva' }) as never
		)) as { status: number; data: { message: string; path: string } };

		expect(fetchMock.mock.calls[0][0]).toBe('http://api.test/api/v1/libraries/4/paths');
		expect(fetchMock.mock.calls[0][1]?.method).toBe('POST');
		expect(JSON.parse(String(fetchMock.mock.calls[0][1]?.body))).toEqual({ path: '/media/nueva' });
		expect(result.status).toBe(409);
		expect(result.data.message).toContain('otra biblioteca');
		expect(result.data.path).toBe('/media/nueva');
	});

	it('quitar carpeta llama a DELETE con los dos identificadores', async () => {
		const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));

		const result = (await removePath(
			actionEvent(fetchMock, { id: '4', pathId: '11' }) as never
		)) as { success?: boolean; message?: string };

		expect(result.success).toBe(true);
		expect(result.message).toContain('Carpeta quitada');
		expect(fetchMock.mock.calls[0][0]).toBe('http://api.test/api/v1/libraries/4/paths/11');
		expect(fetchMock.mock.calls[0][1]?.method).toBe('DELETE');
	});

	it('quitar la única carpeta devuelve el error de la API', async () => {
		const fetchMock = vi
			.fn()
			.mockResolvedValue(
				jsonResponse({ error: 'una biblioteca debe tener al menos una carpeta' }, 400)
			);

		const result = (await removePath(
			actionEvent(fetchMock, { id: '4', pathId: '11' }) as never
		)) as { status: number; data: { message: string } };

		expect(result.status).toBe(400);
		expect(result.data.message).toContain('al menos una carpeta');
	});

	it('borrar exige escribir el nombre y luego llama a DELETE', async () => {
		const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));

		const mismatch = (await remove(
			actionEvent(fetchMock, { id: '2', name: 'Series', confirm: 'Otra' }) as never
		)) as { status: number; data: { message: string } };

		expect(mismatch.status).toBe(400);
		expect(mismatch.data.message).toContain('Escribe «Series»');
		expect(fetchMock).not.toHaveBeenCalled();

		const result = (await remove(
			actionEvent(fetchMock, { id: '2', name: 'Series', confirm: 'Series' }) as never
		)) as { success?: boolean; message?: string };

		expect(result.success).toBe(true);
		expect(result.message).toContain('Biblioteca borrada');
		expect(fetchMock.mock.calls[0][0]).toBe('http://api.test/api/v1/libraries/2');
		expect(fetchMock.mock.calls[0][1]?.method).toBe('DELETE');
	});

	it('borrar con 404 muestra el mensaje de la API', async () => {
		const fetchMock = vi
			.fn()
			.mockResolvedValue(jsonResponse({ error: 'la biblioteca no existe' }, 404));

		const result = (await remove(
			actionEvent(fetchMock, { id: '9', name: 'No', confirm: 'No' }) as never
		)) as { status: number; data: { message: string } };

		expect(result.status).toBe(404);
		expect(result.data.message).toContain('no existe');
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
	it('muestra bibliotecas con varias carpetas, tipos y formularios sin JavaScript', () => {
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
		expect(body).toContain('/mnt/otro-disco/peliculas');
		expect(body).toContain('Películas y series');
		expect(body).toContain('120 fichas');
		expect(body).toContain('Vídeos · Próximamente');
		expect(body).toContain('Música · Próximamente');
		expect(body).toContain('disabled');
		expect(body).toContain('Nueva biblioteca');
		expect(body).toContain('Añadir otra carpeta');
		expect(body).toContain('Elegir…');
		expect(body).toContain('action="?/create"');
		expect(body).toContain('action="?/rename"');
		expect(body).toContain('action="?/addPath"');
		expect(body).toContain('action="?/removePath"');
		expect(body).toContain('action="?/delete"');
		expect(body).toContain('action="?/scan"');
		expect(body).toContain('Escanear todas las bibliotecas');
		expect(body).toContain('Escaneo iniciado.');
		expect(body).toContain('Escaneo en curso…');
	});

	it('conserva nombre, tipo y varias carpetas tras un error de creación', () => {
		const { body } = render(AdminPage, {
			props: {
				data: { libraries },
				form: {
					success: false,
					action: 'create',
					message: 'la carpeta está solapada',
					name: 'Cine de prueba',
					kind: 'music',
					paths: ['/media/a', '/media/b']
				}
			}
		});

		expect(body).toContain('value="Cine de prueba"');
		expect(body).toContain('value="/media/a"');
		expect(body).toContain('value="/media/b"');
		expect(body).toMatch(/value="music"[^>]*selected/);
		expect(body).toContain('la carpeta está solapada');
		expect(body).toContain('name="paths"');
		expect(body).toContain('name="paths"');
	});

	it('muestra el aviso de que Vídeos y Música aún no se escanean', () => {
		const { body } = render(AdminPage, { props: { data: { libraries } } });

		expect(body).toContain('Los tipos Vídeos y Música aún no se escanean.');
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
