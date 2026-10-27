import { render } from 'svelte/server';
import { describe, expect, it, vi } from 'vitest';
import type { TrashItem } from '$lib/api';
import AdminNav from '$lib/components/AdminNav.svelte';
import { DEFAULT_PUBLIC_UI } from '$lib/settings';
import TrashPage from './+page.svelte';
import { actions, load } from './+page.server';

function jsonResponse(body: unknown, status = 200): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

function trashItem(overrides: Partial<TrashItem> = {}): TrashItem {
	return {
		id: 9,
		library_id: 1,
		title: 'El Padrino',
		year: 1972,
		kind: 'movie',
		original_path: '/media/peliculas/El Padrino (1972).mkv',
		moved_kind: 'file',
		deleted_by: 'ana',
		deleted_at: '2026-10-27T10:00:00Z',
		expires_at: '2099-11-10T10:00:00Z',
		...overrides
	};
}

function loadEvent(fetchImpl: typeof fetch, path = '/admin/trash', token: string | null = 'token-123') {
	return {
		locals: { sessionToken: token },
		fetch: fetchImpl,
		url: new URL(`http://localhost${path}`),
		cookies: {},
		params: {},
		route: { id: '/admin/trash' },
		request: new Request(`http://localhost${path}`),
		setHeaders: vi.fn(),
		getClientAddress: () => '127.0.0.1',
		isDataRequest: false,
		isSubRequest: false,
		platform: undefined
	};
}

function actionEvent(fetchImpl: typeof fetch, action: string, fields: Record<string, string> = {}) {
	const body = new URLSearchParams(fields);
	return {
		request: new Request(`http://localhost/admin/trash?/${action}`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
			body
		}),
		locals: { sessionToken: 'token-123' },
		fetch: fetchImpl,
		cookies: {},
		url: new URL('http://localhost/admin/trash'),
		params: {},
		route: { id: '/admin/trash' },
		setHeaders: vi.fn(),
		getClientAddress: () => '127.0.0.1',
		isDataRequest: false,
		isSubRequest: false,
		platform: undefined
	};
}

function adminFetch(items: TrashItem[] = [trashItem()]) {
	return vi.fn(async (input: RequestInfo | URL) => {
		const url = String(input);
		if (url.endsWith('/api/v1/auth/me')) {
			return jsonResponse({ username: 'ana', role: 'admin' });
		}
		if (url.endsWith('/api/v1/trash')) {
			return jsonResponse({ items });
		}
		throw new Error(`URL inesperada: ${url}`);
	});
}

describe('carga de /admin/trash', () => {
	it('carga la papelera y el aviso de elemento movido', async () => {
		const result = (await load(
			loadEvent(adminFetch(), '/admin/trash?movido=9') as never
		)) as { items: TrashItem[]; moved: string | null };

		expect(result.items).toHaveLength(1);
		expect(result.items[0].title).toBe('El Padrino');
		expect(result.moved).toBe('9');
	});

	it('rechaza a un usuario normal y redirige sin sesión', async () => {
		const normal = vi.fn(async (input: RequestInfo | URL) => {
			if (String(input).endsWith('/api/v1/auth/me')) {
				return jsonResponse({ username: 'pepe', role: 'user' });
			}
			throw new Error('no debe listar');
		});
		await expect(load(loadEvent(normal) as never)).rejects.toMatchObject({ status: 403 });

		await expect(load(loadEvent(vi.fn(), '/admin/trash', null) as never)).rejects.toMatchObject({
			status: 303,
			location: '/login'
		});
	});

	it('devuelve 502 si falla el listado', async () => {
		const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
			if (String(input).endsWith('/api/v1/auth/me')) {
				return jsonResponse({ username: 'ana', role: 'admin' });
			}
			return jsonResponse({ error: 'fallo' }, 500);
		});
		await expect(load(loadEvent(fetchMock) as never)).rejects.toMatchObject({ status: 502 });
	});
});

describe('acciones de /admin/trash', () => {
	it('restaura un elemento', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ item_id: 4 }));
		const result = (await actions.restore!(
			actionEvent(fetchMock, 'restore', { id: '9' }) as never
		)) as { success?: boolean; message?: string };

		expect(result.success).toBe(true);
		expect(result.message).toContain('restaurado');
		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe('http://api.test/api/v1/trash/9/restore');
		expect(init.method).toBe('POST');
	});

	it('explica el ajuste apagado al restaurar con 403', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ error: 'apagado' }, 403));
		const result = (await actions.restore!(
			actionEvent(fetchMock, 'restore', { id: '9' }) as never
		)) as { status: number; data: { message: string } };

		expect(result.status).toBe(403);
		expect(result.data.message).toContain('Permitir cambiar archivos');
		expect(result.data.message).toContain('Administración → Biblioteca');
	});

	it('muestra el conflicto de ruta al restaurar', async () => {
		const fetchMock = vi
			.fn()
			.mockResolvedValue(jsonResponse({ error: 'la ruta original ya existe' }, 409));
		const result = (await actions.restore!(
			actionEvent(fetchMock, 'restore', { id: '9' }) as never
		)) as { status: number; data: { message: string } };

		expect(result.status).toBe(409);
		expect(result.data.message).toContain('ruta original');
	});

	it('elimina definitivamente y purga', async () => {
		const deleted = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
		const deletedResult = (await actions.delete!(
			actionEvent(deleted, 'delete', { id: '9' }) as never
		)) as { success?: boolean; message?: string };
		expect(deletedResult.success).toBe(true);
		expect(deletedResult.message).toContain('definitivamente');
		expect(deleted.mock.calls[0][0]).toBe('http://api.test/api/v1/trash/9');

		const purged = vi.fn().mockResolvedValue(jsonResponse({ purged: 2 }));
		const purgedResult = (await actions.purge!(
			actionEvent(purged, 'purge') as never
		)) as { success?: boolean; message?: string };
		expect(purgedResult.success).toBe(true);
		expect(purgedResult.message).toContain('2');
	});
});

describe('página /admin/trash', () => {
	it('muestra datos, acciones y confirmación de borrado', () => {
		const body = render(TrashPage, {
			props: {
				data: {
					items: [trashItem()],
					moved: '9',
					ui: DEFAULT_PUBLIC_UI
				} as never,
				form: null
			}
		}).body;

		expect(body).toContain('Papelera');
		expect(body).toContain('Elemento movido a la papelera');
		expect(body).toContain('El Padrino (1972)');
		expect(body).toContain('/media/peliculas/El Padrino (1972).mkv');
		expect(body).toContain('ana');
		expect(body).toContain('Quedan');
		expect(body).toContain('Restaurar');
		expect(body).toContain('Vaciar lo caducado');
		expect(body).toContain('Eliminar definitivamente');
		expect(body).toContain('Se borrará del disco para siempre.');
		expect(body).toContain('action="?/delete"');
	});

	it('muestra el estado vacío y los mensajes de las acciones', () => {
		const empty = render(TrashPage, {
			props: {
				data: { items: [], moved: null, ui: DEFAULT_PUBLIC_UI } as never,
				form: { success: true, message: 'No había elementos caducados.' }
			}
		}).body;
		expect(empty).toContain('La papelera está vacía.');
		expect(empty).toContain('No había elementos caducados.');

		const error = render(TrashPage, {
			props: {
				data: { items: [], moved: null, ui: DEFAULT_PUBLIC_UI } as never,
				form: { success: false, message: 'No se pudo restaurar.' }
			}
		}).body;
		expect(error).toContain('No se pudo restaurar.');
		expect(error).toContain('role="alert"');
	});
});

describe('navegación de administración', () => {
	it('incluye Papelera y Biblioteca en las posiciones previstas', () => {
		const body = render(AdminNav, { props: { pathname: '/admin/trash' } }).body;
		expect(body.indexOf('href="/admin/libraries"')).toBeLessThan(
			body.indexOf('href="/admin/trash"')
		);
		expect(body.indexOf('href="/admin/trash"')).toBeLessThan(
			body.indexOf('href="/admin/settings/scan"')
		);
		expect(body.indexOf('href="/admin/settings/playback"')).toBeLessThan(
			body.indexOf('href="/admin/settings/library"')
		);
		expect(body.indexOf('href="/admin/settings/library"')).toBeLessThan(
			body.indexOf('href="/admin/settings/system"')
		);
		expect(body).toContain('href="/admin/trash"');
		expect(body).toContain('href="/admin/settings/library"');
	});
});
