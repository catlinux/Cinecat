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

	it('reenvía la IP del cliente al hacer login si se indica', async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			jsonResponse({ token: 'token-123', expires_at: '2026-10-26T12:00:00Z' })
		);
		const client = new ApiClient({ baseUrl: 'http://api.test', fetch: fetchMock });

		await client.login({ username: 'ana', password: 'secreta' }, '203.0.113.7');

		expect(fetchMock.mock.calls[0][1].headers['X-Forwarded-For']).toBe('203.0.113.7');
	});

	it('no envía X-Forwarded-For si la IP está vacía', async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			jsonResponse({ token: 'token-123', expires_at: '2026-10-26T12:00:00Z' })
		);
		const client = new ApiClient({ baseUrl: 'http://api.test', fetch: fetchMock });

		await client.login({ username: 'ana', password: 'secreta' }, '  ');

		expect(fetchMock.mock.calls[0][1].headers['X-Forwarded-For']).toBeUndefined();
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

	it('consulta y crea copias de seguridad del sistema', async () => {
		const backup = {
			name: 'cinecat-20261027-100000.db',
			size_bytes: 1024,
			created_at: '2026-10-27T10:00:00Z'
		};
		const fetchMock = vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
			if (init?.method === 'POST') {
				return jsonResponse(backup, 201);
			}
			return jsonResponse({
				enabled: true,
				retention_days: 7,
				backups: [backup],
				last_error: ''
			});
		});
		const client = new ApiClient({ baseUrl: 'http://api.test', fetch: fetchMock });

		const list = await client.getBackups('token-123');
		const created = await client.createBackup('token-123');

		expect(list).toMatchObject({ enabled: true, retention_days: 7 });
		expect(list.backups[0]).toEqual(backup);
		expect(created).toEqual(backup);
		expect(fetchMock.mock.calls.map(([url]) => String(url))).toEqual([
			'http://api.test/api/v1/system/backups',
			'http://api.test/api/v1/system/backups'
		]);
		expect(fetchMock.mock.calls[0][1]?.method).toBe('GET');
		expect(fetchMock.mock.calls[1][1]?.method).toBe('POST');
		for (const [, init] of fetchMock.mock.calls) {
			expect((init?.headers as Record<string, string>).Authorization).toBe('Bearer token-123');
		}
	});

	it('crea, renombra, añade y quita carpetas de bibliotecas', async () => {
		const fetchMock = vi.fn(async (_input: RequestInfo | URL, _init?: RequestInit) =>
			jsonResponse({ id: 7, name: 'Pelis', kind: 'video', path: '/media/a', paths: [] })
		);
		const client = new ApiClient({ baseUrl: 'http://api.test/', fetch: fetchMock });

		await client.createLibrary(
			{ name: 'Pelis', kind: 'video', paths: ['/media/a', '/media/b'] },
			'token-123'
		);
		await client.renameLibrary(7, 'Películas', 'token-123');
		await client.addLibraryPath(7, '/media/c', 'token-123');
		await client.removeLibraryPath(7, 3, 'token-123');

		expect(fetchMock.mock.calls.map(([url]) => String(url))).toEqual([
			'http://api.test/api/v1/libraries',
			'http://api.test/api/v1/libraries/7',
			'http://api.test/api/v1/libraries/7/paths',
			'http://api.test/api/v1/libraries/7/paths/3'
		]);
		expect(fetchMock.mock.calls.map(([, init]) => init?.method)).toEqual([
			'POST',
			'PATCH',
			'POST',
			'DELETE'
		]);
		expect(JSON.parse(String(fetchMock.mock.calls[0][1]?.body))).toEqual({
			name: 'Pelis',
			kind: 'video',
			paths: ['/media/a', '/media/b']
		});
		expect(JSON.parse(String(fetchMock.mock.calls[1][1]?.body))).toEqual({ name: 'Películas' });
		expect(JSON.parse(String(fetchMock.mock.calls[2][1]?.body))).toEqual({ path: '/media/c' });
	});

	it('consulta el navegador de carpetas con y sin ruta', async () => {
		const fetchMock = vi.fn(async (_input: RequestInfo | URL, _init?: RequestInit) =>
			jsonResponse({ roots: [] })
		);
		const client = new ApiClient({ baseUrl: 'http://api.test', fetch: fetchMock });

		await client.browseFS(null, 'token-123');
		await client.browseFS('/media/pelis con espacio', 'token-123');

		expect(fetchMock.mock.calls.map(([url]) => String(url))).toEqual([
			'http://api.test/api/v1/fs',
			'http://api.test/api/v1/fs?path=%2Fmedia%2Fpelis%20con%20espacio'
		]);
		expect(fetchMock.mock.calls[0][1]?.headers).toMatchObject({
			Authorization: 'Bearer token-123'
		});
	});

	it('navega por las carpetas de una biblioteca de vídeos', async () => {
		const fetchMock = vi.fn(async (_input: RequestInfo | URL, _init?: RequestInit) =>
			jsonResponse({ roots: [] })
		);
		const client = new ApiClient({ baseUrl: 'http://api.test', fetch: fetchMock });

		await client.browseVideoLibrary(7, {}, 'token-123');
		await client.browseVideoLibrary(
			7,
			{ root: 3, path: 'Pelis/Sub', limit: 12, offset: 24 },
			'token-123'
		);

		expect(fetchMock.mock.calls.map(([url]) => String(url))).toEqual([
			'http://api.test/api/v1/libraries/7/browse',
			'http://api.test/api/v1/libraries/7/browse?root=3&path=Pelis%2FSub&limit=12&offset=24'
		]);
		expect(fetchMock.mock.calls[0][1]?.headers).toMatchObject({
			Authorization: 'Bearer token-123'
		});
	});
});

describe('ApiClient de música', () => {
	it('construye listados, detalles y reproducción de música', async () => {
		const fetchMock = vi.fn(async (_input: RequestInfo | URL, _init?: RequestInit) =>
			jsonResponse({ artists: [], albums: [], tracks: [], total: 0 })
		);
		const client = new ApiClient({ baseUrl: 'http://api.test/', fetch: fetchMock });

		await client.getMusicArtists({ q: 'beat', letter: 'B', limit: 10, offset: 20 }, 'token');
		await client.getMusicArtist(3, { limit: 5, offset: 0 }, 'token');
		await client.getMusicAlbums({ q: 'road', letter: 'A', sort: 'year', artist: 3 }, 'token');
		await client.getMusicAlbum(7, { limit: 200, offset: 0 }, 'token');
		await client.getMusicTracks({ q: 'come' }, 'token');
		fetchMock.mockResolvedValueOnce(
			jsonResponse({ url: '/stream', mime: 'audio/mpeg', expires_at: '', playable: true })
		);
		await client.getMusicPlay(11, 'token');

		expect(fetchMock.mock.calls.map(([url]) => String(url))).toEqual([
			'http://api.test/api/v1/music/artists?limit=10&offset=20&q=beat&letter=B',
			'http://api.test/api/v1/music/artists/3?limit=5&offset=0',
			'http://api.test/api/v1/music/albums?q=road&letter=A&sort=year&artist=3',
			'http://api.test/api/v1/music/albums/7?limit=200&offset=0',
			'http://api.test/api/v1/music/tracks?q=come',
			'http://api.test/api/v1/music/tracks/11/play'
		]);
		for (const [, init] of fetchMock.mock.calls) {
			expect((init?.headers as Record<string, string>).Authorization).toBe('Bearer token');
		}
	});
});

describe('ApiClient de listas', () => {
	it('consulta listas y marca ítems, series y colecciones', async () => {
		const fetchMock = vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
			if (init?.method === 'PUT' || init?.method === 'DELETE') {
				return new Response(null, { status: 204 });
			}
			return jsonResponse({
				items: [],
				series: [],
				collections: [],
				totals: { items: 0, series: 0, collections: 0 },
				limit: 10,
				offset: 5
			});
		});
		const client = new ApiClient({ baseUrl: 'http://api.test', fetch: fetchMock });

		await client.getLists('favorite', { kind: 'items', limit: 10, offset: 5 }, 'token');
		await client.setItemList(2, 'favorite', true, 'token');
		await client.setSeriesList(3, 'watchlist', false, 'token');
		await client.setCollectionList(4, 'favorite', true, 'token');

		expect(fetchMock.mock.calls.map(([url]) => String(url))).toEqual([
			'http://api.test/api/v1/lists/favorite?kind=items&limit=10&offset=5',
			'http://api.test/api/v1/items/2/lists/favorite',
			'http://api.test/api/v1/series/3/lists/watchlist',
			'http://api.test/api/v1/collections/4/lists/favorite'
		]);
		expect(fetchMock.mock.calls[1][1]?.method).toBe('PUT');
		expect(fetchMock.mock.calls[2][1]?.method).toBe('DELETE');
	});
});

describe('ApiClient de la papelera', () => {
	it('mueve, lista, restaura, elimina y purga elementos', async () => {
		const trash = {
			id: 9,
			library_id: 1,
			title: 'Peli',
			year: 2020,
			kind: 'movie',
			original_path: '/media/Peli.mkv',
			moved_kind: 'file',
			deleted_by: 'ana',
			deleted_at: '2026-10-27T10:00:00Z',
			expires_at: '2026-11-10T10:00:00Z'
		};
		const fetchMock = vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
			if (init?.method === 'DELETE' && String(_input).endsWith('/api/v1/trash/9')) {
				return new Response(null, { status: 204 });
			}
			if (init?.method === 'DELETE') {
				return jsonResponse(trash);
			}
			if (String(_input).endsWith('/restore')) {
				return jsonResponse({ item_id: 4 });
			}
			if (String(_input).endsWith('/purge')) {
				return jsonResponse({ purged: 2 });
			}
			return jsonResponse({ items: [trash] });
		});
		const client = new ApiClient({ baseUrl: 'http://api.test', fetch: fetchMock });

		await client.trashItem(4, 'token');
		await client.listTrash('token');
		await client.restoreTrash(9, 'token');
		await client.deleteTrash(9, 'token');
		await client.purgeTrash('token');

		expect(fetchMock.mock.calls.map(([url, init]) => [String(url), init?.method ?? 'GET'])).toEqual([
			['http://api.test/api/v1/items/4', 'DELETE'],
			['http://api.test/api/v1/trash', 'GET'],
			['http://api.test/api/v1/trash/9/restore', 'POST'],
			['http://api.test/api/v1/trash/9', 'DELETE'],
			['http://api.test/api/v1/trash/purge', 'POST']
		]);
	});
});

describe('ApiClient de usuarios', () => {
	it('consulta el usuario actual con acceso y gestiona cuentas', async () => {
		const user = {
			id: 4,
			username: 'bea',
			role: 'user',
			disabled: false,
			all_libraries: false,
			library_ids: [2],
			created_at: '2026-10-28T10:00:00Z',
			active_sessions: 1
		};
		const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
			const url = String(input);
			if (url.endsWith('/api/v1/auth/me')) {
				return jsonResponse({
					username: 'ana',
					role: 'admin',
					all_libraries: true,
					library_ids: []
				});
			}
			if (init?.method === 'DELETE' || url.endsWith('/password')) {
				return new Response(null, { status: 204 });
			}
			if (url.endsWith('/api/v1/users') && init?.method === 'POST') {
				return jsonResponse(user, 201);
			}
			if (url.endsWith('/api/v1/users')) {
				return jsonResponse({ users: [user] });
			}
			return jsonResponse(user);
		});
		const client = new ApiClient({ baseUrl: 'http://api.test', fetch: fetchMock });

		const current = await client.getCurrentUser('token');
		expect(current.all_libraries).toBe(true);
		const list = await client.listUsers('token');
		expect(list.users[0].username).toBe('bea');
		await client.createUser(
			{
				username: 'bea',
				password: 'Clave-segura-1',
				role: 'user',
				all_libraries: false,
				library_ids: [2]
			},
			'token'
		);
		await client.updateUser(4, { role: 'admin', disabled: true }, 'token');
		await client.setUserLibraries(4, [1, 2], 'token');
		await client.resetUserPassword(4, 'Clave-nueva-1', 'token');
		await client.deleteUser(4, 'token');

		expect(fetchMock.mock.calls.map(([url, init]) => [String(url), init?.method ?? 'GET'])).toEqual([
			['http://api.test/api/v1/auth/me', 'GET'],
			['http://api.test/api/v1/users', 'GET'],
			['http://api.test/api/v1/users', 'POST'],
			['http://api.test/api/v1/users/4', 'PATCH'],
			['http://api.test/api/v1/users/4/libraries', 'PUT'],
			['http://api.test/api/v1/users/4/password', 'POST'],
			['http://api.test/api/v1/users/4', 'DELETE']
		]);
		expect(JSON.parse(String(fetchMock.mock.calls[2][1]?.body))).toMatchObject({
			username: 'bea',
			role: 'user',
			library_ids: [2]
		});
		expect(JSON.parse(String(fetchMock.mock.calls[3][1]?.body))).toEqual({
			role: 'admin',
			disabled: true
		});
		expect(JSON.parse(String(fetchMock.mock.calls[4][1]?.body))).toEqual({
			library_ids: [1, 2]
		});
		expect(JSON.parse(String(fetchMock.mock.calls[5][1]?.body))).toEqual({
			password: 'Clave-nueva-1'
		});
	});
});
