import { render } from 'svelte/server';
import { describe, expect, it, vi } from 'vitest';
import type { ActivityEntry, ManagedUser } from '$lib/api';
import AdminNav from '$lib/components/AdminNav.svelte';
import { DEFAULT_PUBLIC_UI } from '$lib/settings';
import ActivityPage from './+page.svelte';
import { load } from './+page.server';

function jsonResponse(body: unknown, status = 200): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

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
		id: 2,
		username: 'bea',
		role: 'user',
		disabled: false,
		all_libraries: true,
		library_ids: [],
		created_at: '2026-10-02T10:00:00Z',
		active_sessions: 0
	}
];

const entries: ActivityEntry[] = [
	{
		user_id: 2,
		username: 'bea',
		item_id: 8,
		title: 'El episodio',
		kind: 'episode',
		series_title: 'Una serie',
		season: 2,
		episode: 3,
		position_seconds: 120,
		duration_seconds: 2400,
		watched: false,
		updated_at: '2026-10-31T09:00:00Z'
	},
	{
		user_id: 1,
		username: 'ana',
		item_id: 4,
		title: 'El Padrino',
		kind: 'movie',
		series_title: null,
		season: null,
		episode: null,
		position_seconds: 600,
		duration_seconds: 7200,
		watched: true,
		updated_at: '2026-10-31T08:00:00Z'
	}
];

function loadEvent(
	fetchImpl: typeof fetch,
	path = '/admin/activity',
	token: string | null = 'token-123'
) {
	return {
		locals: { sessionToken: token },
		fetch: fetchImpl,
		url: new URL(`http://localhost${path}`),
		cookies: {},
		params: {},
		route: { id: '/admin/activity' },
		request: new Request(`http://localhost${path}`),
		setHeaders: vi.fn(),
		getClientAddress: () => '127.0.0.1',
		isDataRequest: false,
		isSubRequest: false,
		platform: undefined
	};
}

function adminFetch(activityEntries = entries) {
	return vi.fn(async (input: RequestInfo | URL) => {
		const url = new URL(String(input));
		if (url.pathname === '/api/v1/auth/me') {
			return jsonResponse({ username: 'ana', role: 'admin' });
		}
		if (url.pathname === '/api/v1/users') {
			return jsonResponse({ users });
		}
		if (url.pathname === '/api/v1/activity') {
			return jsonResponse({ entries: activityEntries });
		}
		throw new Error(`URL inesperada: ${url}`);
	});
}

describe('carga de /admin/activity', () => {
	it('carga la actividad y la lista de usuarios', async () => {
		const fetchMock = adminFetch();
		const result = (await load(loadEvent(fetchMock) as never)) as {
			entries: ActivityEntry[];
			users: ManagedUser[];
			selectedUserId: number | null;
			limit: number;
		};

		expect(result.entries).toHaveLength(2);
		expect(result.users).toHaveLength(2);
		expect(result.selectedUserId).toBeNull();
		expect(result.limit).toBe(100);
		const activityCall = fetchMock.mock.calls.find(([url]) =>
			String(url).includes('/api/v1/activity')
		);
		const activityURL = new URL(String(activityCall?.[0]));
		expect(activityURL.searchParams.get('limit')).toBe('100');
		expect(activityURL.searchParams.has('user_id')).toBe(false);
	});

	it('filtra por usuario y valida el parámetro', async () => {
		const fetchMock = adminFetch();
		const result = (await load(
			loadEvent(fetchMock, '/admin/activity?user_id=2') as never
		)) as { selectedUserId: number | null };

		expect(result.selectedUserId).toBe(2);
		const activityCall = fetchMock.mock.calls.find(([url]) =>
			String(url).includes('/api/v1/activity')
		);
		expect(new URL(String(activityCall?.[0])).searchParams.get('user_id')).toBe('2');

		const invalidFetch = adminFetch();
		await expect(
			load(loadEvent(invalidFetch, '/admin/activity?user_id=nope') as never)
		).rejects.toMatchObject({ status: 400 });
	});

	it('rechaza a un usuario normal y redirige sin sesión', async () => {
		const normal = vi.fn(async (input: RequestInfo | URL) => {
			if (String(input).endsWith('/api/v1/auth/me')) {
				return jsonResponse({ username: 'bea', role: 'user' });
			}
			throw new Error('no debe listar');
		});
		await expect(load(loadEvent(normal) as never)).rejects.toMatchObject({ status: 403 });

		await expect(
			load(loadEvent(vi.fn(), '/admin/activity', null) as never)
		).rejects.toMatchObject({
			status: 303,
			location: '/login'
		});
	});

	it('devuelve 404 para un usuario desconocido y 502 si falla la API', async () => {
		await expect(
			load(loadEvent(adminFetch(), '/admin/activity?user_id=999') as never)
		).rejects.toMatchObject({ status: 404 });

		const broken = vi.fn(async (input: RequestInfo | URL) => {
			const url = new URL(String(input));
			if (url.pathname === '/api/v1/auth/me') {
				return jsonResponse({ username: 'ana', role: 'admin' });
			}
			if (url.pathname === '/api/v1/users') {
				return jsonResponse({ users });
			}
			return jsonResponse({ error: 'fallo' }, 500);
		});
		await expect(load(loadEvent(broken) as never)).rejects.toMatchObject({ status: 502 });
	});
});

describe('página /admin/activity', () => {
	it('muestra la tabla con usuario, episodio, progreso y estado', () => {
		const body = render(ActivityPage, {
			props: {
				data: {
					entries,
					users,
					selectedUserId: null,
					limit: 100,
					ui: DEFAULT_PUBLIC_UI
				} as never
			}
		}).body;

		expect(body).toContain('Actividad');
		expect(body).toContain('Todos los usuarios');
		expect(body).toContain('bea');
		expect(body).toContain('El episodio');
		expect(body).toContain('Una serie');
		expect(body).toContain('T2 E3');
		expect(body).toContain('2 min de 40 min');
		expect(body).toContain('En curso');
		expect(body).toContain('Visto');
		expect(body).toContain('/items/8');
	});

	it('muestra el estado vacío y el filtro activo', () => {
		const body = render(ActivityPage, {
			props: {
				data: {
					entries: [],
					users,
					selectedUserId: 2,
					limit: 100,
					ui: DEFAULT_PUBLIC_UI
				} as never
			}
		}).body;
		expect(body).toContain('Sin actividad');
		expect(body).toContain('Este usuario todavía no tiene reproducciones');
		expect(body).toContain('Quitar filtro');
	});
});

describe('navegación de administración con actividad', () => {
	it('incluye el enlace Actividad', () => {
		const body = render(AdminNav, { props: { pathname: '/admin/activity' } }).body;
		expect(body).toContain('href="/admin/activity"');
		expect(body).toContain('Actividad');
		expect(body).toContain('aria-current="page"');
	});
});
