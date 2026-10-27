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
		hasClips: false,
		hasMusic: false,
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

function layoutFetch(libraries: unknown[] = []) {
	return vi.fn(async (input: RequestInfo | URL) => {
		const url = String(input);
		if (url.endsWith('/api/v1/system/version')) {
			return jsonResponse({ server: webVersion, commit: '', built_at: '' });
		}
		if (url.endsWith('/api/v1/settings/public')) {
			return jsonResponse({});
		}
		if (url.endsWith('/api/v1/auth/me')) {
			return jsonResponse({ username: 'ana', role: 'user' });
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
		expect(result.hasClips).toBe(false);
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
