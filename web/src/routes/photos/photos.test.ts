import { render } from 'svelte/server';
import { describe, expect, it, vi } from 'vitest';
import type { Library, PhotoItem } from '$lib/api';
import PhotoViewer from '$lib/components/PhotoViewer.svelte';
import { DEFAULT_PUBLIC_UI } from '$lib/settings';
import PhotoLibraryPage from './[libraryId]/+page.svelte';
import { load as loadLibrary } from './[libraryId]/+page.server';
import { GET as OriginalProxy } from './original/[id]/+server';
import PhotosPage from './+page.svelte';
import { load as loadPhotos } from './+page.server';
import { GET as ThumbProxy } from './thumb/[id]/+server';

function jsonResponse(body: unknown, status = 200, headers: Record<string, string> = {}): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json', ...headers }
	});
}

function photoLibrary(overrides: Partial<Library> = {}): Library {
	return {
		id: 4,
		name: 'Fotos de casa',
		kind: 'photos',
		path: '/media/fotos',
		paths: [{ id: 9, path: '/media/fotos' }],
		item_count: 2,
		...overrides
	};
}

function photoItem(overrides: Partial<PhotoItem> = {}): PhotoItem {
	return {
		id: 31,
		name: 'IMG_0031.jpg',
		width: 4000,
		height: 3000,
		taken_at: 1600000000,
		size: 123456,
		...overrides
	};
}

const photos = {
	folders: ['2024'],
	items: [photoItem(), photoItem({ id: 32, name: 'IMG_0032.png', width: 3000, height: 4000 })],
	total: 2
};

function loadEvent(fetchImpl: typeof fetch, url = 'http://localhost/photos') {
	const parsed = new URL(url);
	return {
		locals: { sessionToken: 'token-123' },
		fetch: fetchImpl,
		url: parsed,
		params: { libraryId: parsed.pathname.split('/').filter(Boolean).at(-1) ?? '' },
		parent: async () => ({ isAdmin: false, ui: DEFAULT_PUBLIC_UI })
	};
}

function photosFetch(overrides: Record<string, () => Response> = {}) {
	return vi.fn(async (input: RequestInfo | URL, _init?: RequestInit) => {
		const url = new URL(String(input));
		for (const [suffix, response] of Object.entries(overrides)) {
			if (url.pathname.endsWith(suffix) || url.search.includes(suffix)) {
				return response();
			}
		}
		if (url.pathname === '/api/v1/libraries') {
			return jsonResponse({ libraries: [photoLibrary()] });
		}
		if (url.pathname === '/api/v1/photos') {
			return jsonResponse(photos);
		}
		throw new Error(`URL inesperada: ${url}`);
	});
}

function endpointEvent(
	fetchImpl: typeof fetch,
	path: string,
	token: string | null = 'token-123',
	headers: Record<string, string> = {}
) {
	return {
		params: { id: path.split('/').at(-1) ?? '' },
		locals: { sessionToken: token },
		fetch: fetchImpl,
		url: new URL(`http://localhost${path}`),
		request: new Request(`http://localhost${path}`, { headers })
	};
}

describe('carga de /photos', () => {
	it('redirige si solo hay una biblioteca de fotos', async () => {
		const fetchMock = photosFetch();
		await expect(loadPhotos(loadEvent(fetchMock) as never)).rejects.toMatchObject({
			status: 303,
			location: '/photos/4'
		});
	});

	it('devuelve las tarjetas si hay varias bibliotecas', async () => {
		const fetchMock = vi.fn(async () =>
			jsonResponse({
				libraries: [photoLibrary(), photoLibrary({ id: 5, name: 'Viajes' })]
			})
		);
		const result = (await loadPhotos(loadEvent(fetchMock) as never)) as any;
		expect(result.libraries).toHaveLength(2);
		expect(result.errorMessage).toBeNull();
	});

	it('traduce 401 y los fallos de la API', async () => {
		await expect(
			loadPhotos(loadEvent(vi.fn(async () => jsonResponse({ error: 'sesión' }, 401)) as never) as never)
		).rejects.toMatchObject({ status: 303, location: '/login' });

		const broken = (await loadPhotos(
			loadEvent(vi.fn(async () => jsonResponse({ error: 'fallo' }, 500)) as never) as never
		)) as any;
		expect(broken.errorMessage).toBeTruthy();
	});
});

describe('carga de /photos/[libraryId]', () => {
	it('carga la carpeta, el orden y la paginación', async () => {
		const fetchMock = photosFetch({
			'/api/v1/photos': () => jsonResponse({ ...photos, total: 100, items: [photoItem()] })
		});
		const result = (await loadLibrary(
			loadEvent(fetchMock, 'http://localhost/photos/4?folder=2024&sort=name&page=2') as never
		)) as any;

		expect(result.libraryName).toBe('Fotos de casa');
		expect(result.folder).toBe('2024');
		expect(result.sort).toBe('name');
		expect(result.crumbs).toEqual([{ name: '2024', path: '2024' }]);
		expect(result.items).toHaveLength(1);
		expect(result.page).toBe(2);
		expect(result.last).toBe(3);
		const requestURL = new URL(String(fetchMock.mock.calls[1][0]));
		expect(requestURL.searchParams.get('library_id')).toBe('4');
		expect(requestURL.searchParams.get('folder')).toBe('2024');
		expect(requestURL.searchParams.get('sort')).toBe('name');
		expect(requestURL.searchParams.get('limit')).toBe('48');
		expect(requestURL.searchParams.get('offset')).toBe('48');
	});

	it('rechaza una biblioteca que no sea de fotos o una carpeta inválida', async () => {
		const videoLibrary = vi.fn(async () =>
			jsonResponse({ libraries: [photoLibrary({ kind: 'video' })] })
		);
		await expect(
			loadLibrary(loadEvent(videoLibrary as never, 'http://localhost/photos/4') as never)
		).rejects.toMatchObject({ status: 404 });

		const traversal = photosFetch();
		await expect(
			loadLibrary(
				loadEvent(traversal as never, 'http://localhost/photos/4?folder=../fuera') as never
			)
		).rejects.toMatchObject({ status: 404 });
		expect(traversal).not.toHaveBeenCalled();
	});

	it('redirige a la última página, al login y traduce 404/502', async () => {
		const fetchMock = photosFetch({
			'/api/v1/photos': () => jsonResponse({ ...photos, total: 100 })
		});
		await expect(
			loadLibrary(loadEvent(fetchMock as never, 'http://localhost/photos/4?page=99') as never)
		).rejects.toMatchObject({ status: 303, location: '/photos/4?page=3' });

		const unauthorized = vi.fn(async (input: RequestInfo | URL) => {
			if (String(input).endsWith('/api/v1/libraries')) {
				return jsonResponse({ libraries: [photoLibrary()] });
			}
			return jsonResponse({ error: 'sesión' }, 401);
		});
		await expect(
			loadLibrary(loadEvent(unauthorized as never, 'http://localhost/photos/4') as never)
		).rejects.toMatchObject({ status: 303, location: '/login' });

		const missing = vi.fn(async (input: RequestInfo | URL) => {
			if (String(input).endsWith('/api/v1/libraries')) {
				return jsonResponse({ libraries: [photoLibrary()] });
			}
			return jsonResponse({ error: 'no' }, 404);
		});
		await expect(
			loadLibrary(loadEvent(missing as never, 'http://localhost/photos/4') as never)
		).rejects.toMatchObject({ status: 404 });

		const broken = vi.fn(async (input: RequestInfo | URL) => {
			if (String(input).endsWith('/api/v1/libraries')) {
				return jsonResponse({ libraries: [photoLibrary()] });
			}
			return jsonResponse({ error: 'fallo' }, 500);
		});
		await expect(
			loadLibrary(loadEvent(broken as never, 'http://localhost/photos/4') as never)
		).rejects.toMatchObject({ status: 502 });
	});
});

describe('proxies de miniatura y original', () => {
	it('reenvía el token, el cuerpo y las cabeceras de la miniatura', async () => {
		const fetchMock = vi.fn(async (_input: RequestInfo | URL, _init?: RequestInit) =>
			new Response(new Uint8Array([1, 2, 3]), {
				status: 200,
				headers: {
					'Content-Type': 'image/jpeg',
					'Cache-Control': 'private, max-age=86400',
					ETag: '"miniatura"'
				}
			})
		);
		const response = await ThumbProxy(
			endpointEvent(fetchMock as never, '/photos/thumb/31') as never
		);
		expect(response.status).toBe(200);
		expect(response.headers.get('Content-Type')).toBe('image/jpeg');
		expect(response.headers.get('Cache-Control')).toBe('private, max-age=86400');
		expect(response.headers.get('ETag')).toBe('"miniatura"');
		expect((await response.arrayBuffer()).byteLength).toBe(3);
		expect(
			(fetchMock.mock.calls[0][1]?.headers as Record<string, string>).Authorization
		).toBe('Bearer token-123');
	});

	it('propaga 404 y 401 y reenvía Range en el original', async () => {
		const missing = vi.fn(async () => jsonResponse({ error: 'no' }, 404));
		expect(
			(await ThumbProxy(endpointEvent(missing as never, '/photos/thumb/31') as never)).status
		).toBe(404);
		expect(
			(await ThumbProxy(
				endpointEvent(vi.fn() as never, '/photos/thumb/31', null) as never
			)).status
		).toBe(401);

		const ranged = vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
			expect((init?.headers as Record<string, string>).Range).toBe('bytes=0-3');
			return new Response(new Uint8Array([1, 2, 3, 4]), {
				status: 206,
				headers: {
					'Content-Type': 'image/jpeg',
					'Content-Range': 'bytes 0-3/10',
					'Accept-Ranges': 'bytes'
				}
			});
		});
		const response = await OriginalProxy(
			endpointEvent(ranged as never, '/photos/original/31', 'token-123', {
				Range: 'bytes=0-3'
			}) as never
		);
		expect(response.status).toBe(206);
		expect(response.headers.get('Content-Range')).toBe('bytes 0-3/10');
		expect(response.headers.get('Accept-Ranges')).toBe('bytes');
		expect((await response.arrayBuffer()).byteLength).toBe(4);
	});
});

describe('páginas y visor de fotos', () => {
	it('renderiza la lista de bibliotecas y la cuadrícula', () => {
		const list = render(PhotosPage, {
			props: {
				data: {
					libraries: [photoLibrary()],
					isAdmin: false,
					errorMessage: null,
					ui: DEFAULT_PUBLIC_UI
				} as never
			}
		}).body;
		expect(list).toContain('Fotos de casa');
		expect(list).toContain('href="/photos/4"');

		const grid = render(PhotoLibraryPage, {
			props: {
				data: {
					libraryID: 4,
					libraryName: 'Fotos de casa',
					folder: '2024',
					crumbs: [{ name: '2024', path: '2024' }],
					items: [photoItem()],
					folders: ['Verano'],
					total: 1,
					sort: 'date',
					page: 1,
					last: 1,
					pageSize: 48,
					currentUrl: '/photos/4?folder=2024',
					errorMessage: null,
					ui: DEFAULT_PUBLIC_UI
				} as never
			}
		}).body;
		expect(grid).toContain('/photos/thumb/31');
		expect(grid).toContain('loading="lazy"');
		expect(grid).toContain('alt="IMG_0031.jpg"');
		expect(grid).toContain('/photos/4?folder=2024%2FVerano');
	});

	it('el visor es un diálogo con contador y enlace al original', () => {
		const { body } = render(PhotoViewer, {
			props: {
				photos: [photoItem()],
				startIndex: 0,
				onclose: vi.fn()
			}
		});
		expect(body).toContain('role="dialog"');
		expect(body).toContain('aria-modal="true"');
		expect(body).toContain('1 de 1');
		expect(body).toContain('/photos/original/31');
		expect(body).toContain('Presentación automática');
		expect(body).toContain('← Anterior');
		expect(body).toContain('Siguiente →');
	});
});
