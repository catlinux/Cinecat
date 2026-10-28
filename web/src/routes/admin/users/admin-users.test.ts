import { render } from 'svelte/server';
import { describe, expect, it, vi } from 'vitest';
import type { Library, ManagedUser } from '$lib/api';
import { DEFAULT_PUBLIC_UI } from '$lib/settings';
import { webVersion } from '$lib/version';
import UsersPage from './+page.svelte';
import { actions, load } from './+page.server';

const createAction = actions.create!;
const updateAction = actions.update!;
const librariesAction = actions.libraries!;
const passwordAction = actions.password!;
const deleteAction = actions.delete!;

function jsonResponse(body: unknown, status = 200): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

const libraries: Library[] = [
	{ id: 1, name: 'Películas', kind: 'video', item_count: 10 },
	{ id: 2, name: 'Vídeos', kind: 'clips', item_count: 3 },
	{ id: 3, name: 'Música', kind: 'music', item_count: 8 }
];

const users: ManagedUser[] = [
	{
		id: 1,
		username: 'ana',
		role: 'admin',
		disabled: false,
		all_libraries: true,
		library_ids: [],
		created_at: '2026-10-01T10:00:00Z',
		active_sessions: 1
	},
	{
		id: 4,
		username: 'bea',
		role: 'user',
		disabled: false,
		all_libraries: false,
		library_ids: [1],
		created_at: '2026-10-02T10:00:00Z',
		active_sessions: 2
	}
];

function loadEvent(fetchImpl: typeof fetch, token: string | null = 'token-123') {
	return {
		locals: { sessionToken: token },
		fetch: fetchImpl,
		url: new URL('http://localhost/admin/users'),
		params: {},
		cookies: {},
		route: { id: '/admin/users' },
		request: new Request('http://localhost/admin/users'),
		setHeaders: vi.fn(),
		getClientAddress: () => '127.0.0.1',
		isDataRequest: false,
		isSubRequest: false,
		platform: undefined
	};
}

function actionEvent(
	fetchImpl: typeof fetch,
	name: string,
	fields: Record<string, string | string[]>,
	token: string | null = 'token-123'
) {
	const form = new FormData();
	for (const [key, value] of Object.entries(fields)) {
		for (const item of Array.isArray(value) ? value : [value]) {
			form.append(key, item);
		}
	}
	return {
		request: new Request(`http://localhost/admin/users?/${name}`, { method: 'POST', body: form }),
		locals: { sessionToken: token },
		fetch: fetchImpl,
		params: {},
		url: new URL(`http://localhost/admin/users?/${name}`),
		cookies: {},
		route: { id: '/admin/users' },
		setHeaders: vi.fn(),
		getClientAddress: () => '127.0.0.1',
		isDataRequest: false,
		isSubRequest: false,
		platform: undefined
	};
}

function adminFetch(overrides: (url: URL, init?: RequestInit) => Response | undefined = () => undefined) {
	return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
		const url = new URL(String(input));
		const override = overrides(url, init);
		if (override) {
			return override;
		}
		if (url.pathname === '/api/v1/auth/me') {
			return jsonResponse({ username: 'ana', role: 'admin', all_libraries: true, library_ids: [] });
		}
		if (url.pathname === '/api/v1/users' && (init?.method ?? 'GET') === 'GET') {
			return jsonResponse({ users });
		}
		if (url.pathname === '/api/v1/libraries') {
			return jsonResponse({ libraries });
		}
		if (url.pathname === '/api/v1/users' && init?.method === 'POST') {
			return jsonResponse(users[1], 201);
		}
		if (url.pathname.startsWith('/api/v1/users/') && init?.method === 'PATCH') {
			return jsonResponse(users[1]);
		}
		if (url.pathname.endsWith('/libraries') && init?.method === 'PUT') {
			return jsonResponse(users[1]);
		}
		if (url.pathname.endsWith('/password')) {
			return new Response(null, { status: 204 });
		}
		if (url.pathname.startsWith('/api/v1/users/') && init?.method === 'DELETE') {
			return new Response(null, { status: 204 });
		}
		throw new Error(`URL inesperada: ${url} ${init?.method ?? 'GET'}`);
	});
}

describe('carga de usuarios', () => {
	it('redirige sin sesión', async () => {
		await expect(load(loadEvent(vi.fn(), null) as never)).rejects.toMatchObject({
			status: 303,
			location: '/login'
		});
	});

	it('rechaza a un usuario no administrador', async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			jsonResponse({ username: 'bea', role: 'user', all_libraries: false, library_ids: [1] })
		);
		await expect(load(loadEvent(fetchMock) as never)).rejects.toMatchObject({ status: 403 });
	});

	it('carga usuarios y bibliotecas para el administrador', async () => {
		const result = (await load(loadEvent(adminFetch()) as never)) as {
			users: ManagedUser[];
			libraries: Library[];
			currentUser: { username: string };
		};
		expect(result.users).toHaveLength(2);
		expect(result.libraries).toHaveLength(3);
		expect(result.currentUser.username).toBe('ana');
	});

	it('devuelve 502 si falla la lista de usuarios', async () => {
		const fetchMock = adminFetch((url) =>
			url.pathname === '/api/v1/users' ? jsonResponse({ error: 'fallo' }, 500) : undefined
		);
		await expect(load(loadEvent(fetchMock) as never)).rejects.toMatchObject({ status: 502 });
	});
});

describe('acciones de usuarios', () => {
	it('crea un usuario y envía sus datos', async () => {
		const fetchMock = adminFetch();
		const result = (await createAction(
			actionEvent(fetchMock, 'create', {
				username: 'carla',
				password: 'Clave-segura-1',
				repeat_password: 'Clave-segura-1',
				role: 'user',
				library_ids: ['1', '2']
			}) as never
		)) as { success?: boolean; message?: string };
		expect(result.success).toBe(true);
		expect(result.message).toBe('Usuario creado.');
		const post = fetchMock.mock.calls.find(
			([url, init]) => String(url).endsWith('/api/v1/users') && init?.method === 'POST'
		);
		expect(JSON.parse(String(post?.[1]?.body))).toEqual({
			username: 'carla',
			password: 'Clave-segura-1',
			role: 'user',
			all_libraries: false,
			library_ids: [1, 2]
		});
	});

	it('exige biblioteca o Todas a un usuario normal sin llamar a la API', async () => {
		const fetchMock = adminFetch();
		const result = (await createAction(
			actionEvent(fetchMock, 'create', {
				username: 'carla',
				password: 'Clave-segura-1',
				repeat_password: 'Clave-segura-1',
				role: 'user'
			}) as never
		)) as { status: number; data: { message: string } };
		expect(result.status).toBe(400);
		expect(result.data.message).toContain('biblioteca');
		expect(fetchMock).not.toHaveBeenCalled();
	});

	it('propaga el 409 de nombre duplicado', async () => {
		const fetchMock = adminFetch((url, init) =>
			url.pathname === '/api/v1/users' && init?.method === 'POST'
				? jsonResponse({ error: 'ya existe un usuario con ese nombre' }, 409)
				: undefined
		);
		const result = (await createAction(
			actionEvent(fetchMock, 'create', {
				username: 'bea',
				password: 'Clave-segura-1',
				repeat_password: 'Clave-segura-1',
				role: 'user',
				all_libraries: 'true'
			}) as never
		)) as { status: number; data: { message: string } };
		expect(result.status).toBe(409);
		expect(result.data.message).toContain('ya existe');
	});

	it('actualiza el rol y propaga el 409 del último administrador', async () => {
		const fetchMock = adminFetch();
		const result = (await updateAction(
			actionEvent(fetchMock, 'update', { id: '4', role: 'admin' }) as never
		)) as { success?: boolean };
		expect(result.success).toBe(true);
		const patch = fetchMock.mock.calls.find(([, init]) => init?.method === 'PATCH');
		expect(JSON.parse(String(patch?.[1]?.body))).toEqual({ role: 'admin' });

		const conflict = adminFetch((url, init) =>
			url.pathname === '/api/v1/users/4' && init?.method === 'PATCH'
				? jsonResponse({ error: 'debe quedar al menos un administrador activo' }, 409)
				: undefined
		);
		const failed = (await updateAction(
			actionEvent(conflict, 'update', { id: '4', role: 'user' }) as never
		)) as { status: number; data: { message: string } };
		expect(failed.status).toBe(409);
		expect(failed.data.message).toContain('administrador');
	});

	it('propaga el 403 al intentar modificar la cuenta propia', async () => {
		const fetchMock = adminFetch((url, init) =>
			url.pathname === '/api/v1/users/1' && init?.method === 'PATCH'
				? jsonResponse({ error: 'no puedes desactivar tu propia cuenta' }, 403)
				: undefined
		);
		const result = (await updateAction(
			actionEvent(fetchMock, 'update', { id: '1', disabled: 'true' }) as never
		)) as { status: number; data: { message: string } };
		expect(result.status).toBe(403);
		expect(result.data.message).toContain('propia cuenta');
	});

	it('guarda bibliotecas y el acceso a todas', async () => {
		const fetchMock = adminFetch();
		const result = (await librariesAction(
			actionEvent(fetchMock, 'libraries', {
				id: '4',
				library_ids: ['2', '3'],
				all_libraries_present: '1',
				all_libraries: 'true'
			}) as never
		)) as { success?: boolean };
		expect(result.success).toBe(true);
		const put = fetchMock.mock.calls.find(([, init]) => init?.method === 'PUT');
		expect(JSON.parse(String(put?.[1]?.body))).toEqual({ library_ids: [2, 3] });
		const patches = fetchMock.mock.calls.filter(([, init]) => init?.method === 'PATCH');
		expect(JSON.parse(String(patches[0][1]?.body))).toEqual({ all_libraries: true });
	});

	it('restablece una contraseña y valida la confirmación', async () => {
		const fetchMock = adminFetch();
		const result = (await passwordAction(
			actionEvent(fetchMock, 'password', {
				id: '4',
				password: 'Clave-nueva-1',
				repeat_password: 'Clave-nueva-1'
			}) as never
		)) as { success?: boolean; message?: string };
		expect(result.success).toBe(true);
		expect(result.message).toContain('sesiones');

		const invalid = adminFetch();
		const failed = (await passwordAction(
			actionEvent(invalid, 'password', {
				id: '4',
				password: 'Clave-nueva-1',
				repeat_password: 'otra'
			}) as never
		)) as { status: number };
		expect(failed.status).toBe(400);
		expect(invalid).not.toHaveBeenCalled();
	});

	it('exige escribir el nombre antes de borrar y permite borrar después', async () => {
		const fetchMock = adminFetch();
		const mismatch = (await deleteAction(
			actionEvent(fetchMock, 'delete', { id: '4', confirmation: 'otra' }) as never
		)) as { status: number; data: { message: string } };
		expect(mismatch.status).toBe(400);
		expect(mismatch.data.message).toContain('nombre');
		expect(fetchMock.mock.calls.some(([, init]) => init?.method === 'DELETE')).toBe(false);

		const deleted = (await deleteAction(
			actionEvent(fetchMock, 'delete', { id: '4', confirmation: 'bea' }) as never
		)) as { success?: boolean };
		expect(deleted.success).toBe(true);
		expect(fetchMock.mock.calls.some(([, init]) => init?.method === 'DELETE')).toBe(true);
	});

	it('redirige al login si la acción recibe 401', async () => {
		const fetchMock = adminFetch((url, init) =>
			url.pathname === '/api/v1/users' && init?.method === 'POST'
				? jsonResponse({ error: 'sesión' }, 401)
				: undefined
		);
		await expect(
			createAction(
				actionEvent(fetchMock, 'create', {
					username: 'carla',
					password: 'Clave-segura-1',
					repeat_password: 'Clave-segura-1',
					role: 'user',
					all_libraries: 'true'
				}) as never
			)
		).rejects.toMatchObject({ status: 303, location: '/login' });
	});
});

describe('página de usuarios', () => {
	it('marca al usuario actual y no ofrece acciones destructivas sobre él', () => {
		const body = render(UsersPage, {
			props: {
				data: {
					users,
					libraries,
					currentUser: { username: 'ana', role: 'admin', all_libraries: true, library_ids: [] },
					ui: DEFAULT_PUBLIC_UI,
					hasVideo: true,
					hasAnyLibraryAccess: true,
					hasSession: true,
					isAdmin: true,
					hasClips: true,
					hasMusic: true,
					isHome: false,
					activeCategory: null,
					serverVersion: null,
					serverCommit: null,
					webVersion
				} as never
			}
		}).body;
		expect(body).toContain('Nuevo usuario');
		expect(body).toContain('bea');
		expect(body).toContain('Tú');
		expect(body).toContain('Restablecer contraseña');
		expect(body).toContain('user-delete-4');
		expect(body).not.toContain('user-delete-1');
	});
});
