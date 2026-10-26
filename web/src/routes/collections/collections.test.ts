import { render } from 'svelte/server';
import { describe, expect, it, vi } from 'vitest';
import type { Collection, CollectionDetail, MediaItem } from '$lib/api';
import CollectionPage from './[id]/+page.svelte';
import { actions as detailActions, load as loadDetail } from './[id]/+page.server';
import CollectionsPage from './+page.svelte';
import { actions as listActions, load as loadList } from './+page.server';

function jsonResponse(body: unknown, status = 200): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

function emptyResponse(status = 204): Response {
	return new Response(null, { status });
}

function movie(id: number, title: string, watched = false): MediaItem {
	return {
		id,
		library_id: 1,
		path: `/media/${id}.mkv`,
		title,
		file_title: title,
		year: 2000 + id,
		kind: 'movie',
		season: null,
		episode: null,
		tmdb_id: null,
		overview: null,
		poster_url: `https://images.example/${id}.jpg`,
		poster_custom: false,
		original_title: null,
		imdb_id: null,
		tagline: null,
		genres: [],
		runtime_minutes: null,
		rating: null,
		directors: [],
		cast: [],
		backdrop_url: null,
		backdrop_custom: false,
		overview_language: null,
		match_status: null,
		match_score: null,
		watched,
		progress: null,
		collections: [{ id: 5, name: 'La saga' }],
		series_id: null,
		series_title: null,
		episode_title: null,
		still_url: null
	};
}

const collection: Collection = {
	id: 5,
	name: 'La saga',
	overview: 'Una saga de prueba.',
	poster_url: 'https://images.example/collection.jpg',
	backdrop_url: null,
	item_count: 2,
	source: 'tmdb',
	edited: false
};

const detail: CollectionDetail = {
	...collection,
	items: [movie(1, 'Primera', true), movie(2, 'Segunda', false)]
};

function listLoadEvent(fetchImpl: typeof fetch, path = '/collections', isAdmin = false) {
	return {
		locals: { sessionToken: 'token-123' },
		fetch: fetchImpl,
		url: new URL(`http://localhost${path}`),
		parent: async () => ({ isAdmin })
	};
}

function detailLoadEvent(fetchImpl: typeof fetch, path = '/collections/5', isAdmin = true) {
	return {
		params: { id: '5' },
		locals: { sessionToken: 'token-123' },
		fetch: fetchImpl,
		url: new URL(`http://localhost${path}`),
		parent: async () => ({ isAdmin })
	};
}

function actionEvent(
	fetchImpl: typeof fetch,
	action: string,
	fields: Record<string, string> = {},
	id = '5',
	token: string | null = 'token-123'
) {
	const path = id ? `/collections/${id}` : '/collections';
	return {
		request: new Request(`http://localhost${path}?/${action}`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
			body: new URLSearchParams(fields)
		}),
		locals: { sessionToken: token },
		fetch: fetchImpl,
		params: id ? { id } : {}
	};
}

function listData(overrides: Record<string, unknown> = {}) {
	return {
		collections: [collection],
		total: 1,
		page: 1,
		last: 1,
		pageSize: 48,
		filters: { q: '', sort: 'name', page: 1 },
		currentUrl: '/collections',
		isAdmin: false,
		errorMessage: null,
		...overrides
	};
}

function detailData(overrides: Record<string, unknown> = {}) {
	return {
		collection: detail,
		isAdmin: false,
		searchQuery: '',
		searchResults: [],
		searchError: null,
		manage: false,
		...overrides
	};
}

describe('carga del listado de colecciones', () => {
	it('traduce filtros y página a la API', async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			jsonResponse({ collections: [collection], total: 100, limit: 48, offset: 48 })
		);
		const result = (await loadList(
			listLoadEvent(fetchMock, '/collections?q=saga&sort=size&page=2') as never
		)) as {
			page: number;
			last: number;
			collections: Collection[];
		};

		const url = new URL(String(fetchMock.mock.calls[0][0]));
		expect(url.pathname).toBe('/api/v1/collections');
		expect(url.searchParams.get('limit')).toBe('48');
		expect(url.searchParams.get('offset')).toBe('48');
		expect(url.searchParams.get('q')).toBe('saga');
		expect(url.searchParams.get('sort')).toBe('size');
		expect(result.page).toBe(2);
		expect(result.last).toBe(3);
		expect(result.collections).toHaveLength(1);
	});

	it('redirige a la última página si se pasa de rango', async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			jsonResponse({ collections: [], total: 100, limit: 48, offset: 0 })
		);
		await expect(
			loadList(listLoadEvent(fetchMock, '/collections?q=saga&sort=name&page=99') as never)
		).rejects.toMatchObject({
			status: 303,
			location: '/collections?q=saga&sort=name&page=3'
		});
	});

	it('redirige al login con 401', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ error: 'sin sesión' }, 401));
		await expect(loadList(listLoadEvent(fetchMock) as never)).rejects.toMatchObject({
			status: 303,
			location: '/login'
		});
	});

	it('devuelve un estado vacío con mensaje si la API falla', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ error: 'fallo' }, 500));
		const result = (await loadList(listLoadEvent(fetchMock) as never)) as {
			collections: Collection[];
			total: number;
			errorMessage: string | null;
		};
		expect(result.collections).toEqual([]);
		expect(result.total).toBe(0);
		expect(result.errorMessage).toBeTruthy();
	});
});

describe('acciones del listado', () => {
	it('valida y crea una colección, redirigiendo a su ficha', async () => {
		const invalidFetch = vi.fn();
		const invalid = (await listActions.create!(
			actionEvent(invalidFetch, 'create', { name: '   ' }, '') as never
		)) as { status: number; data: { message: string } };
		expect(invalid.status).toBe(400);
		expect(invalidFetch).not.toHaveBeenCalled();

		const fetchMock = vi.fn().mockResolvedValue(jsonResponse(collection, 201));
		await expect(
			listActions.create!(
				actionEvent(fetchMock, 'create', { name: 'La saga', overview: 'Sinopsis' }, '') as never
			)
		).rejects.toMatchObject({ status: 303, location: '/collections/5' });
		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe('http://api.test/api/v1/collections');
		expect(init.method).toBe('POST');
		expect(JSON.parse(init.body)).toEqual({ name: 'La saga', overview: 'Sinopsis' });
	});

	it('traduce los errores de creación', async () => {
		const forbidden = vi.fn().mockResolvedValue(jsonResponse({ error: 'no admin' }, 403));
		const result = (await listActions.create!(
			actionEvent(forbidden, 'create', { name: 'La saga' }, '') as never
		)) as { status: number; data: { message: string } };
		expect(result.status).toBe(403);
		expect(result.data.message).toContain('administradores');
	});
});

describe('carga del detalle', () => {
	it('carga la colección y la búsqueda de películas', async () => {
		const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
			const url = new URL(String(input));
			if (url.pathname === '/api/v1/collections/5') {
				return jsonResponse(detail);
			}
			if (url.pathname === '/api/v1/items') {
				return jsonResponse({ items: [movie(3, 'Tercera')], total: 1, limit: 12, offset: 0 });
			}
			throw new Error(`URL inesperada: ${url}`);
		});

		const result = (await loadDetail(
			detailLoadEvent(fetchMock, '/collections/5?buscar=matrix&gestionar=1') as never
		)) as {
			collection: CollectionDetail;
			isAdmin: boolean;
			searchQuery: string;
			searchResults: MediaItem[];
			manage: boolean;
		};

		expect(result.collection.name).toBe('La saga');
		expect(result.isAdmin).toBe(true);
		expect(result.searchQuery).toBe('matrix');
		expect(result.searchResults).toHaveLength(1);
		expect(result.manage).toBe(true);
		const searchURL = new URL(String(fetchMock.mock.calls[1][0]));
		expect(searchURL.pathname).toBe('/api/v1/items');
		expect(searchURL.searchParams.get('kind')).toBe('movie');
		expect(searchURL.searchParams.get('q')).toBe('matrix');
		expect(searchURL.searchParams.get('limit')).toBe('12');
	});

	it('devuelve 404 si la colección no existe', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ error: 'no existe' }, 404));
		await expect(loadDetail(detailLoadEvent(fetchMock) as never)).rejects.toMatchObject({
			status: 404
		});
	});

	it('no busca películas para un usuario que no es administrador', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse(detail));
		const result = (await loadDetail(
			detailLoadEvent(fetchMock, '/collections/5?buscar=matrix', false) as never
		)) as { searchQuery: string; searchResults: MediaItem[]; manage: boolean };
		expect(fetchMock).toHaveBeenCalledTimes(1);
		expect(result.searchQuery).toBe('');
		expect(result.searchResults).toEqual([]);
		expect(result.manage).toBe(false);
	});
});

describe('acciones del detalle', () => {
	it('actualiza nombre y descripción', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ ...collection, name: 'Nueva' }));
		const result = (await detailActions.update!(
			actionEvent(fetchMock, 'update', { name: 'Nueva', overview: 'Otra' }) as never
		)) as { success?: boolean; manage?: boolean; message?: string };
		expect(result.success).toBe(true);
		expect(result.manage).toBe(true);
		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe('http://api.test/api/v1/collections/5');
		expect(init.method).toBe('PATCH');
		expect(JSON.parse(init.body)).toEqual({ name: 'Nueva', overview: 'Otra' });
	});

	it('cambia cartel y fondo, incluido vaciarlos', async () => {
		for (const [kind, key] of [
			['poster', 'poster_url'],
			['backdrop', 'backdrop_url']
		] as const) {
			const fetchMock = vi.fn().mockResolvedValue(jsonResponse(collection));
			await detailActions.poster!(
				actionEvent(fetchMock, 'poster', { kind, url: 'https://images.example/x.jpg' }) as never
			);
			const body = JSON.parse(fetchMock.mock.calls[0][1].body);
			expect(body).toEqual({ [key]: 'https://images.example/x.jpg' });

			const cleared = vi.fn().mockResolvedValue(jsonResponse(collection));
			await detailActions.poster!(
				actionEvent(cleared, 'poster', { kind, url: '' }) as never
			);
			expect(JSON.parse(cleared.mock.calls[0][1].body)).toEqual({ [key]: '' });
		}
	});

	it('añade y quita películas', async () => {
		const addFetch = vi.fn().mockResolvedValue(emptyResponse());
		await detailActions.addItem!(
			actionEvent(addFetch, 'addItem', { media_item_id: '3' }) as never
		);
		expect(String(addFetch.mock.calls[0][0])).toBe('http://api.test/api/v1/collections/5/items');
		expect(addFetch.mock.calls[0][1].method).toBe('POST');
		expect(JSON.parse(addFetch.mock.calls[0][1].body)).toEqual({ media_item_id: 3 });

		const removeFetch = vi.fn().mockResolvedValue(emptyResponse());
		await detailActions.removeItem!(
			actionEvent(removeFetch, 'removeItem', { media_item_id: '1' }) as never
		);
		expect(String(removeFetch.mock.calls[0][0])).toBe(
			'http://api.test/api/v1/collections/5/items/1'
		);
		expect(removeFetch.mock.calls[0][1].method).toBe('DELETE');
	});

	it('sube una película calculando la lista completa', async () => {
		const fetchMock = vi
			.fn()
			.mockResolvedValueOnce(jsonResponse(detail))
			.mockResolvedValueOnce(emptyResponse());
		await detailActions.move!(
			actionEvent(fetchMock, 'move', { media_item_id: '2', direction: 'up' }) as never
		);
		expect(fetchMock).toHaveBeenCalledTimes(2);
		expect(String(fetchMock.mock.calls[0][0])).toBe('http://api.test/api/v1/collections/5');
		expect(String(fetchMock.mock.calls[1][0])).toBe(
			'http://api.test/api/v1/collections/5/order'
		);
		expect(fetchMock.mock.calls[1][1].method).toBe('PUT');
		expect(JSON.parse(fetchMock.mock.calls[1][1].body)).toEqual({ media_item_ids: [2, 1] });
	});

	it('borra la colección y redirige', async () => {
		const fetchMock = vi.fn().mockResolvedValue(emptyResponse());
		await expect(
			detailActions.delete!(actionEvent(fetchMock, 'delete') as never)
		).rejects.toMatchObject({ status: 303, location: '/collections' });
		expect(String(fetchMock.mock.calls[0][0])).toBe('http://api.test/api/v1/collections/5');
		expect(fetchMock.mock.calls[0][1].method).toBe('DELETE');
	});

	it('traduce 400, 403, 404, 502 y 401', async () => {
		const cases = [
			[400, 'nombre no válido', 'nombre no válido'],
			[403, 'no admin', 'administradores'],
			[404, 'no existe', 'Colección no encontrada.'],
			[500, 'fallo', 'No se pudo completar la operación.']
		] as const;

		for (const [status, apiMessage, expected] of cases) {
			const fetchMock = vi
				.fn()
				.mockResolvedValue(jsonResponse({ error: apiMessage }, status));
			const result = (await detailActions.update!(
				actionEvent(fetchMock, 'update', { name: 'Saga', overview: '' }) as never
			)) as { status: number; data: { message: string } };
			expect(result.status).toBe(status === 500 ? 502 : status);
			expect(result.data.message).toContain(expected);
		}

		await expect(
			detailActions.update!(
				actionEvent(vi.fn(), 'update', { name: 'Saga' }, '5', null) as never
			)
		).rejects.toMatchObject({ status: 303, location: '/login' });
	});
});

describe('render del listado', () => {
	it('muestra las colecciones con su recuento', () => {
		const { body } = render(CollectionsPage, { props: { data: listData() as never } });
		expect(body).toContain('Colecciones');
		expect(body).toContain('1 colección');
		expect(body).toContain('La saga');
		expect(body).toContain('2 películas');
		expect(body).toContain('/collections/5');
		expect(body).not.toContain('Nueva colección');
	});

	it('muestra el panel de creación solo al administrador', () => {
		const { body } = render(CollectionsPage, {
			props: { data: listData({ isAdmin: true }) as never }
		});
		expect(body).toContain('Nueva colección');
	});

	it('distingue el vacío con y sin filtros', () => {
		const empty = render(CollectionsPage, {
			props: { data: listData({ collections: [], total: 0 }) as never }
		}).body;
		expect(empty).toContain('Todavía no hay colecciones');

		const filtered = render(CollectionsPage, {
			props: {
				data: listData({
					collections: [],
					total: 0,
					filters: { q: 'saga', sort: 'name', page: 1 }
				}) as never
			}
		}).body;
		expect(filtered).toContain('Ninguna colección coincide');
		expect(filtered).toContain('Limpiar filtros');
	});
});

describe('render del detalle', () => {
	it('muestra cabecera, películas y primera sin ver', () => {
		const { body } = render(CollectionPage, {
			props: { data: detailData() as never }
		});
		expect(body).toContain('La saga');
		expect(body).toContain('Una saga de prueba.');
		expect(body).toContain('2 películas');
		expect(body).toContain('1 de 2 vistas');
		expect(body).toContain('Reproducir la primera sin ver');
		expect(body).toContain('/items/2/play');
		expect(body).not.toContain('Gestionar colección');
	});

	it('muestra la gestión solo al administrador', () => {
		const { body } = render(CollectionPage, {
			props: {
				data: detailData({ isAdmin: true, manage: true }) as never
			}
		});
		expect(body).toContain('Gestionar colección');
		expect(body).toContain('Colección automática');
		expect(body).toContain('Sí, borrar esta colección');
	});
});
