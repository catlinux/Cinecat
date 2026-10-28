import { render } from 'svelte/server';
import { createRawSnippet } from 'svelte';
import { describe, expect, it, vi } from 'vitest';
import { DEFAULT_PUBLIC_UI } from '$lib/settings';
import { webVersion } from '$lib/version';
import Layout from './+layout.svelte';
import { load } from './+layout.server';

const children = createRawSnippet(() => ({
	render: () => '<p>Contenido</p>'
}));

function jsonResponse(body: unknown, status = 200): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

function layoutData(overrides: Record<string, unknown> = {}) {
	return {
		hasSession: true,
		isAdmin: false,
		hasVideo: true,
		hasClips: false,
		hasMusic: false,
		hasAnyLibraryAccess: true,
		isHome: false,
		activeCategory: null,
		serverVersion: null,
		serverCommit: null,
		webVersion,
		ui: DEFAULT_PUBLIC_UI,
		...overrides
	};
}

function loadEvent(fetchImpl: typeof fetch, pathname = '/music') {
	return {
		locals: { sessionToken: 'token-123' },
		fetch: fetchImpl,
		url: new URL(`http://localhost${pathname}`)
	};
}

function layoutFetch(libraries: unknown[] = [], role: 'admin' | 'user' = 'user', allLibraries = false) {
	return vi.fn(async (input: RequestInfo | URL) => {
		const url = String(input);
		if (url.endsWith('/api/v1/system/version')) {
			return jsonResponse({ server: webVersion, commit: '', built_at: '' });
		}
		if (url.endsWith('/api/v1/settings/public')) {
			return jsonResponse({});
		}
		if (url.endsWith('/api/v1/auth/me')) {
			return jsonResponse({ username: 'ana', role, all_libraries: allLibraries, library_ids: [] });
		}
		if (url.endsWith('/api/v1/libraries')) {
			return jsonResponse({ libraries });
		}
		throw new Error(`URL inesperada: ${url}`);
	});
}

describe('menú de Música', () => {
	it('muestra Música en el menú lateral solo con hasMusic', () => {
		const withMusic = render(Layout, {
			props: {
				data: layoutData({ hasMusic: true, activeCategory: 'music' }),
				children
			}
		}).body;
		expect(withMusic).toContain('href="/music"');
		expect(withMusic).toContain('aria-current="page"');
		expect(withMusic).toContain('Música');

		const withoutMusic = render(Layout, {
			props: { data: layoutData(), children }
		}).body;
		expect(withoutMusic).not.toContain('href="/music"');
	});

	it('muestra Música en el menú superior solo con hasMusic', () => {
		const withMusic = render(Layout, {
			props: {
				data: layoutData({
					hasMusic: true,
					activeCategory: 'music',
					ui: { ...DEFAULT_PUBLIC_UI, nav_layout: 'top' }
				}),
				children
			}
		}).body;
		expect(withMusic).toContain('class="marquee"');
		expect(withMusic).toContain('href="/music"');
		expect(withMusic).toContain('aria-current="page"');

		const withoutMusic = render(Layout, {
			props: {
				data: layoutData({ ui: { ...DEFAULT_PUBLIC_UI, nav_layout: 'top' } }),
				children
			}
		}).body;
		expect(withoutMusic).not.toContain('href="/music"');
	});
});

describe('load del layout', () => {
	it('calcula hasMusic y activeCategory para música', async () => {
		const result = (await load(
			loadEvent(
				layoutFetch([
					{ id: 1, kind: 'music', name: 'Música', path: '/m', paths: [], item_count: 4 },
					{ id: 2, kind: 'clips', name: 'Vídeos', path: '/v', paths: [], item_count: 0 }
				])
			) as never
		)) as any;
		expect(result.hasMusic).toBe(true);
		expect(result.hasClips).toBe(true);
		expect(result.activeCategory).toBe('music');
	});

	it('tolera que falle la lista de bibliotecas', async () => {
		const fetchMock = layoutFetch();
		fetchMock.mockImplementation(async (input: RequestInfo | URL) => {
			const url = String(input);
			if (url.endsWith('/api/v1/libraries')) {
				throw new Error('sin conexión');
			}
			return layoutFetch([])(input);
		});
		const result = (await load(loadEvent(fetchMock as never) as never)) as any;
		expect(result.hasMusic).toBe(false);
		expect(result.hasClips).toBe(false);
	});
});

describe('menú de listas personales', () => {
	it('muestra Favoritos y Próximas a ver solo con sesión', () => {
		const withSession = render(Layout, {
			props: { data: layoutData({ activeCategory: 'favorites' }), children }
		}).body;
		expect(withSession).toContain('href="/favorites"');
		expect(withSession).toContain('href="/watchlist"');
		expect(withSession).toContain('aria-current="page"');
		expect(withSession).toContain('Favoritos');

		const withoutSession = render(Layout, {
			props: { data: layoutData({ hasSession: false }), children }
		}).body;
		expect(withoutSession).not.toContain('href="/favorites"');
		expect(withoutSession).not.toContain('href="/watchlist"');
	});

	it('marca la entrada activa en los dos layouts', () => {
		for (const nav_layout of ['side', 'top'] as const) {
			const body = render(Layout, {
				props: {
					data: layoutData({
						activeCategory: 'watchlist',
						ui: { ...DEFAULT_PUBLIC_UI, nav_layout }
					}),
					children
				}
			}).body;
			expect(body).toContain('href="/watchlist"');
			expect(body).toContain('aria-current="page"');
		}
	});
});

describe('load de listas en el layout', () => {
	it('marca favorites y watchlist en activeCategory', async () => {
		for (const [pathname, expected] of [
			['/favorites', 'favorites'],
			['/watchlist', 'watchlist']
		] as const) {
			const result = (await load(loadEvent(layoutFetch(), pathname) as never)) as any;
			expect(result.activeCategory).toBe(expected);
		}
	});
});


describe('menú filtrado por bibliotecas visibles', () => {
	function renderedFor(result: Record<string, unknown>, nav_layout: 'side' | 'top' = 'side') {
		return render(Layout, {
			props: {
				data: layoutData({
					...result,
					ui: { ...DEFAULT_PUBLIC_UI, nav_layout }
				}),
				children
			}
		}).body;
	}

	it('un usuario con solo vídeos ve Vídeos y no las secciones de películas', async () => {
		const result = (await load(
			loadEvent(layoutFetch([{ id: 2, kind: 'clips', name: 'Vídeos', item_count: 1 }])) as never
		)) as any;
		expect(result.hasVideo).toBe(false);
		expect(result.hasClips).toBe(true);
		expect(result.hasMusic).toBe(false);
		for (const layout of ['side', 'top'] as const) {
			const body = renderedFor(result, layout);
			expect(body).toContain('href="/videos"');
			expect(body).not.toContain('href="/movies"');
			expect(body).not.toContain('href="/series"');
			expect(body).not.toContain('href="/collections"');
		}
	});

	it('un usuario con una biblioteca de vídeo ve Películas, Series y Colecciones', async () => {
		const result = (await load(
			loadEvent(layoutFetch([{ id: 1, kind: 'video', name: 'Pelis', item_count: 0 }])) as never
		)) as any;
		expect(result.hasVideo).toBe(true);
		expect(result.hasClips).toBe(false);
		expect(result.hasMusic).toBe(false);
		const body = renderedFor(result);
		expect(body).toContain('href="/movies"');
		expect(body).toContain('href="/series"');
		expect(body).toContain('href="/collections"');
		expect(body).not.toContain('href="/videos"');
		expect(body).not.toContain('href="/music"');
	});

	it('el administrador ve todas las secciones aunque no haya bibliotecas', async () => {
		const result = (await load(loadEvent(layoutFetch([], 'admin')) as never)) as any;
		expect(result.isAdmin).toBe(true);
		expect(result.hasVideo).toBe(true);
		expect(result.hasClips).toBe(true);
		expect(result.hasMusic).toBe(true);
		const body = renderedFor(result);
		for (const href of ['/movies', '/series', '/collections', '/videos', '/music']) {
			expect(body).toContain(`href="${href}"`);
		}
	});
});

describe('cerrar sesión desde el menú', () => {
	it('muestra el formulario de cierre en los menús superior y lateral con sesión', () => {
		for (const nav_layout of ['side', 'top'] as const) {
			const body = render(Layout, {
				props: {
					data: layoutData({ ui: { ...DEFAULT_PUBLIC_UI, nav_layout } }),
					children
				}
			}).body;
			expect(body).toContain('Cerrar sesión');
			expect(body).toContain('action="/login?/logout"');
			expect(body).toContain('method="POST"');
		}
	});

	it('no lo muestra sin sesión', () => {
		const body = render(Layout, {
			props: { data: layoutData({ hasSession: false }), children }
		}).body;
		expect(body).not.toContain('Cerrar sesión');
		expect(body).not.toContain('/login?/logout');
	});
});
