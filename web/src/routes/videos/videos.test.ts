import { render } from 'svelte/server';
import { describe, expect, it, vi } from 'vitest';
import type { Library, MediaItem, VideoBrowseResponse, VideoRootsResponse } from '$lib/api';
import { DEFAULT_PUBLIC_UI } from '$lib/settings';
import VideosPage from './+page.svelte';
import VideoLibraryPage from './[id]/+page.svelte';
import { load as loadLibrary } from './[id]/+page.server';
import { load as loadVideos } from './+page.server';

function jsonResponse(body: unknown, status = 200): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

function pageData(overrides: Record<string, unknown> = {}): any {
	return {
		hasSession: true,
		isAdmin: false,
		hasClips: true,
		isHome: false,
		activeCategory: 'videos' as const,
		serverVersion: null,
		serverCommit: null,
		webVersion: '0.18.0',
		ui: DEFAULT_PUBLIC_UI,
		...overrides
	};
}

function clipLibrary(overrides: Partial<Library> = {}): Library {
	return {
		id: 4,
		name: 'Vídeos de casa',
		kind: 'clips',
		path: '/media/videos',
		paths: [{ id: 9, path: '/media/videos' }],
		item_count: 12,
		...overrides
	};
}

function videoItem(overrides: Partial<MediaItem> = {}): MediaItem {
	return {
		id: 21,
		library_id: 4,
		library_kind: 'clips',
		path: 'Conciertos/Clip.mp4',
		title: 'Clip de prueba',
		file_title: 'Clip de prueba',
		year: null,
		kind: 'movie',
		season: null,
		episode: null,
		tmdb_id: null,
		overview: null,
		poster_url: null,
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
		match_status: 'none',
		match_score: null,
		watched: false,
		progress: null,
		collections: [],
		series_id: null,
		series_title: null,
		episode_title: null,
		still_url: null,
		...overrides
	};
}

function rootsResponse(): VideoRootsResponse {
	return {
		roots: [
			{ id: 9, name: 'Conciertos', item_count: 7 },
			{ id: 10, name: 'Videoclips', item_count: 5 }
		]
	};
}

function browseResponse(overrides: Partial<VideoBrowseResponse> = {}): VideoBrowseResponse {
	return {
		library: { id: 4, name: 'Vídeos de casa' },
		root: { id: 9, name: 'Conciertos' },
		path: 'Año 2020',
		parent: '',
		crumbs: [
			{ name: 'Conciertos', path: '' },
			{ name: 'Año 2020', path: 'Año 2020' }
		],
		folders: [{ name: 'Verano', path: 'Año 2020/Verano', item_count: 2 }],
		items: [videoItem()],
		total: 1,
		limit: 48,
		offset: 0,
		...overrides
	};
}

function loadEvent(
	fetchImpl: typeof fetch,
	url = 'http://localhost/videos',
	isAdmin = false
) {
	const parsed = new URL(url);
	return {
		locals: { sessionToken: 'token-123' },
		fetch: fetchImpl,
		url: parsed,
		params: { id: parsed.pathname.split('/').filter(Boolean).at(-1) ?? '' },
		parent: async () => ({ isAdmin, ui: DEFAULT_PUBLIC_UI })
	};
}

describe('carga de /videos', () => {
	it('redirige si solo hay una biblioteca de clips con vídeos', async () => {
		const fetchMock = vi.fn(async () =>
			jsonResponse({ libraries: [clipLibrary()] })
		);

		await expect(loadVideos(loadEvent(fetchMock) as never)).rejects.toMatchObject({
			status: 303,
			location: '/videos/4'
		});
	});

	it('devuelve las tarjetas si hay varias bibliotecas', async () => {
		const fetchMock = vi.fn(async () =>
			jsonResponse({ libraries: [clipLibrary(), clipLibrary({ id: 5, name: 'Otros' })] })
		);

		const result = (await loadVideos(loadEvent(fetchMock) as never)) as {
			libraries: Library[];
			errorMessage: string | null;
		};

		expect(result.libraries).toHaveLength(2);
		expect(result.errorMessage).toBeNull();
	});

	it('ignora bibliotecas que no son clips o están vacías', async () => {
		const fetchMock = vi.fn(async () =>
			jsonResponse({
				libraries: [
					clipLibrary({ item_count: 0 }),
					clipLibrary({ id: 6, kind: 'video' }),
					clipLibrary({ id: 7, name: 'Con vídeos', item_count: 2 })
				]
			})
		);

		await expect(loadVideos(loadEvent(fetchMock) as never)).rejects.toMatchObject({
			status: 303,
			location: '/videos/7'
		});
	});

	it('traduce 401 y los fallos de la API', async () => {
		const unauthorized = vi.fn(async () => jsonResponse({ error: 'sesión' }, 401));
		await expect(loadVideos(loadEvent(unauthorized) as never)).rejects.toMatchObject({
			status: 303,
			location: '/login'
		});

		const broken = vi.fn(async () => jsonResponse({ error: 'fallo' }, 502));
		const result = (await loadVideos(loadEvent(broken) as never)) as {
			errorMessage: string | null;
		};
		expect(result.errorMessage).toBeTruthy();
	});
});

describe('carga de /videos/[id]', () => {
	it('carga las raíces de la biblioteca', async () => {
		const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
			const url = new URL(String(input));
			if (url.pathname === '/api/v1/libraries') {
				return jsonResponse({ libraries: [clipLibrary()] });
			}
			if (url.pathname === '/api/v1/libraries/4/browse') {
				return jsonResponse(rootsResponse());
			}
			throw new Error(`URL inesperada: ${url}`);
		});

		const result = (await loadLibrary(
			loadEvent(fetchMock, 'http://localhost/videos/4') as never
		)) as {
			mode: string;
			libraryName: string;
			roots: unknown[];
		};

		expect(result.mode).toBe('roots');
		expect(result.libraryName).toBe('Vídeos de casa');
		expect(result.roots).toHaveLength(2);
	});

	it('carga una carpeta y calcula las migas', async () => {
		const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
			const url = new URL(String(input));
			if (url.pathname === '/api/v1/libraries') {
				return jsonResponse({ libraries: [clipLibrary()] });
			}
			if (url.pathname === '/api/v1/libraries/4/browse') {
				expect(url.searchParams.get('root')).toBe('9');
				expect(url.searchParams.get('path')).toBe('Año 2020');
				expect(url.searchParams.get('limit')).toBe('48');
				return jsonResponse(browseResponse());
			}
			throw new Error(`URL inesperada: ${url}`);
		});

		const result = (await loadLibrary(
			loadEvent(fetchMock, 'http://localhost/videos/4?root=9&path=A%C3%B1o+2020') as never
		)) as {
			mode: string;
			path: string;
			items: MediaItem[];
			folders: unknown[];
			last: number;
		};

		expect(result.mode).toBe('browse');
		expect(result.path).toBe('Año 2020');
		expect(result.items).toHaveLength(1);
		expect(result.folders).toHaveLength(1);
		expect(result.last).toBe(1);
	});

	it('redirige a la última página si la pedida se pasa', async () => {
		const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
			const url = new URL(String(input));
			if (url.pathname === '/api/v1/libraries') {
				return jsonResponse({ libraries: [clipLibrary()] });
			}
			if (url.pathname === '/api/v1/libraries/4/browse') {
				return jsonResponse(browseResponse({ total: 100, limit: 48, offset: 4800 }));
			}
			throw new Error(`URL inesperada: ${url}`);
		});

		await expect(
			loadLibrary(loadEvent(fetchMock, 'http://localhost/videos/4?root=9&page=99') as never)
		).rejects.toMatchObject({
			status: 303,
			location: '/videos/4?root=9&page=3'
		});
	});

	it('rechaza identificadores, root y rutas inválidas', async () => {
		const fetchMock = vi.fn();
		for (const url of [
			'http://localhost/videos/0',
			'http://localhost/videos/4?root=0',
			'http://localhost/videos/4?root=9&path=../fuera',
			'http://localhost/videos/4?root=9&path=/absoluta',
			'http://localhost/videos/4?path=sin-root'
		]) {
			await expect(loadLibrary(loadEvent(fetchMock, url) as never)).rejects.toMatchObject({
				status: 404
			});
		}
		expect(fetchMock).not.toHaveBeenCalled();
	});

	it('traduce 401, 404 y otros errores de la API', async () => {
		const unauthorized = vi.fn(async (input: RequestInfo | URL) => {
			const path = new URL(String(input)).pathname;
			if (path === '/api/v1/libraries') {
				return jsonResponse({ libraries: [] });
			}
			return jsonResponse({ error: 'sesión' }, 401);
		});
		await expect(
			loadLibrary(loadEvent(unauthorized, 'http://localhost/videos/4') as never)
		).rejects.toMatchObject({ status: 303, location: '/login' });

		const notFound = vi.fn(async (input: RequestInfo | URL) => {
			const path = new URL(String(input)).pathname;
			if (path === '/api/v1/libraries') {
				return jsonResponse({ libraries: [clipLibrary()] });
			}
			return jsonResponse({ error: 'no existe' }, 404);
		});
		await expect(
			loadLibrary(loadEvent(notFound, 'http://localhost/videos/4') as never)
		).rejects.toMatchObject({
			status: 404
		});

		const broken = vi.fn(async (input: RequestInfo | URL) => {
			const path = new URL(String(input)).pathname;
			if (path === '/api/v1/libraries') {
				return jsonResponse({ libraries: [clipLibrary()] });
			}
			return jsonResponse({ error: 'fallo' }, 502);
		});
		await expect(
			loadLibrary(loadEvent(broken, 'http://localhost/videos/4') as never)
		).rejects.toMatchObject({
			status: 502
		});
	});
});

describe('páginas de vídeos', () => {
	it('muestra las bibliotecas y enlaza a cada una', () => {
		const { body } = render(VideosPage, {
			props: {
				data: pageData({
					libraries: [clipLibrary(), clipLibrary({ id: 5, name: 'Otros vídeos' })],
					isAdmin: false,
					errorMessage: null
				})
			}
		});

		expect(body).toContain('Vídeos');
		expect(body).toContain('Vídeos de casa');
		expect(body).toContain('Otros vídeos');
		expect(body).toContain('href="/videos/4"');
		expect(body).toContain('12 vídeos');
	});

	it('muestra el estado vacío con enlace de administración solo al admin', () => {
		const admin = render(VideosPage, {
			props: {
				data: pageData({ libraries: [], isAdmin: true, errorMessage: null })
			}
		}).body;
		expect(admin).toContain('Aún no hay vídeos');
		expect(admin).toContain('href="/admin/libraries"');

		const user = render(VideosPage, {
			props: {
				data: pageData({ libraries: [], isAdmin: false, errorMessage: null })
			}
		}).body;
		expect(user).toContain('Aún no hay vídeos');
		expect(user).not.toContain('href="/admin/libraries"');
	});

	it('pinta raíces, carpetas y vídeos con migas y enlaces sin JavaScript', () => {
		const roots = render(VideoLibraryPage, {
			props: {
				data: pageData({
					mode: 'roots',
					libraryID: 4,
					libraryName: 'Vídeos de casa',
					roots: rootsResponse().roots,
					page: 1,
					last: 1,
					pageSize: 48,
					currentUrl: '/videos/4',
					root: null,
					path: '',
					crumbs: [],
					folders: [],
					items: [],
					total: 0
				})
			}
		}).body;

		expect(roots).toContain('Vídeos de casa');
		expect(roots).toContain('Conciertos');
		expect(roots).toContain('href="/videos/4?root=9"');
		expect(roots).toContain('Videoclips');
		expect(roots).toContain('href="/videos"');

		const browse = render(VideoLibraryPage, {
			props: {
				data: pageData({
					mode: 'browse',
					libraryID: 4,
					libraryName: 'Vídeos de casa',
					root: { id: 9, name: 'Conciertos' },
					path: 'Año 2020',
					crumbs: browseResponse().crumbs,
					folders: browseResponse().folders,
					items: browseResponse().items,
					total: 1,
					page: 1,
					last: 1,
					pageSize: 48,
					currentUrl: '/videos/4?root=9&path=A%C3%B1o+2020'
				})
			}
		}).body;

		expect(browse).toContain('Vídeos');
		expect(browse).toContain('Año 2020');
		expect(browse).toContain('Verano');
		expect(browse).toContain('href="/videos/4?root=9&amp;path=A%C3%B1o+2020%2FVerano"');
		expect(browse).toContain('Clip de prueba');
		expect(browse).toContain('href="/items/21"');
	});

	it('muestra un estado vacío cuando la carpeta no tiene nada', () => {
		const { body } = render(VideoLibraryPage, {
			props: {
				data: pageData({
					mode: 'browse',
					libraryID: 4,
					libraryName: 'Vídeos de casa',
					root: { id: 9, name: 'Conciertos' },
					path: '',
					crumbs: [{ name: 'Conciertos', path: '' }],
					folders: [],
					items: [],
					total: 0,
					page: 1,
					last: 1,
					pageSize: 48,
					currentUrl: '/videos/4?root=9'
				})
			}
		});

		expect(body).toContain('No hay vídeos en esta carpeta');
	});
});
