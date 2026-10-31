import { createRawSnippet } from 'svelte';
import { render } from 'svelte/server';
import { describe, expect, it, vi } from 'vitest';
import { DEFAULT_PUBLIC_UI, type PublicUI } from '$lib/settings';
import { versionMismatch, webVersion } from '$lib/version';
import Layout from './+layout.svelte';
import { load } from './+layout.server';

function jsonResponse(body: unknown, status = 200): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

const children = createRawSnippet(() => ({
	render: () => '<p>Contenido</p>'
}));

function uiWith(overrides: Partial<PublicUI> = {}): PublicUI {
	return { ...DEFAULT_PUBLIC_UI, ...overrides };
}

function layoutData(serverVersion: string | null, serverCommit: string | null = null) {
	return {
		hasSession: false,
		isAdmin: false,
		hasVideo: true,
		hasClips: false,
		hasMusic: false,
		hasAnyLibraryAccess: true,
		sections: { movies: true, series: true, collections: true },
		isHome: false,
		activeCategory: null,
		serverVersion,
		serverCommit,
		webVersion,
		ui: DEFAULT_PUBLIC_UI
	};
}

function renderFooter(serverVersion: string | null, serverCommit: string | null = null): string {
	return render(Layout, {
		props: {
			data: layoutData(serverVersion, serverCommit),
			children
		}
	}).body;
}

function visibleText(html: string): string {
	return html.replace(/<!--[\s\S]*?-->/g, '');
}

describe('cálculo del aviso de versión', () => {
	it('no avisa si las versiones coinciden', () => {
		expect(versionMismatch(webVersion, webVersion)).toBe(false);
	});

	it('avisa si las versiones son distintas', () => {
		expect(versionMismatch('999.0.0', webVersion)).toBe(true);
	});

	it('no avisa si el servidor no responde', () => {
		expect(versionMismatch(null, webVersion)).toBe(false);
	});
});

describe('carga de la versión del servidor', () => {
	it('deja la versión nula sin romper la carga si la API falla', async () => {
		const fetchMock = vi.fn().mockRejectedValue(new Error('sin conexión'));

		const result = (await load({
			locals: { sessionToken: null },
			fetch: fetchMock,
			url: new URL('http://localhost/')
		} as never)) as { serverVersion: string | null; webVersion: string };

		expect(result.serverVersion).toBeNull();
		expect(result.webVersion).toBe(webVersion);
	});

	it('devuelve la versión y el commit del servidor', async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			jsonResponse({
				server: '999.0.0',
				commit: 'abc1234',
				built_at: '2026-10-26T09:00:00Z'
			})
		);

		const result = (await load({
			locals: { sessionToken: null },
			fetch: fetchMock,
			url: new URL('http://localhost/')
		} as never)) as {
			serverVersion: string | null;
			serverCommit: string | null;
			webVersion: string;
		};

		expect(result.serverVersion).toBe('999.0.0');
		expect(result.serverCommit).toBe('abc1234');
		expect(fetchMock.mock.calls[0][0]).toBe('http://api.test/api/v1/system/version');
	});

	it('marca Películas como categoría activa en su ruta', async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			jsonResponse({ server: '999.0.0', commit: '', built_at: '' })
		);

		const result = (await load({
			locals: { sessionToken: null },
			fetch: fetchMock,
			url: new URL('http://localhost/movies')
		} as never)) as { activeCategory: string | null };

		expect(result.activeCategory).toBe('movies');
	});

	it('marca Colecciones como categoría activa en sus rutas', async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			jsonResponse({ server: '999.0.0', commit: '', built_at: '' })
		);

		const result = (await load({
			locals: { sessionToken: null },
			fetch: fetchMock,
			url: new URL('http://localhost/collections/5')
		} as never)) as { activeCategory: string | null };

		expect(result.activeCategory).toBe('collections');
	});

	it('marca Series como categoría activa en sus rutas', async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			jsonResponse({ server: '999.0.0', commit: '', built_at: '' })
		);

		const result = (await load({
			locals: { sessionToken: null },
			fetch: fetchMock,
			url: new URL('http://localhost/series/5')
		} as never)) as { activeCategory: string | null };

		expect(result.activeCategory).toBe('series');
	});

	it('marca Vídeos como categoría activa en sus rutas', async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			jsonResponse({ server: '999.0.0', commit: '', built_at: '' })
		);

		const result = (await load({
			locals: { sessionToken: null },
			fetch: fetchMock,
			url: new URL('http://localhost/videos/4')
		} as never)) as { activeCategory: string | null };

		expect(result.activeCategory).toBe('videos');
	});
});

describe('carga de los ajustes públicos', () => {
	it('usa los valores por defecto sin sesión', async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			jsonResponse({ server: '999.0.0', commit: '', built_at: '' })
		);
		const locals = { sessionToken: null } as { sessionToken: string | null; publicUI?: unknown };

		const result = (await load({
			locals,
			fetch: fetchMock,
			url: new URL('http://localhost/')
		} as never)) as { ui: typeof DEFAULT_PUBLIC_UI; isHome: boolean };

		expect(result.ui).toEqual(DEFAULT_PUBLIC_UI);
		expect(result.isHome).toBe(true);
		expect(locals.publicUI).toEqual(DEFAULT_PUBLIC_UI);
	});

	it('aplica los ajustes públicos con sesión', async () => {
		const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
			const path = new URL(String(input)).pathname;
			if (path === '/api/v1/system/version') {
				return jsonResponse({ server: '999.0.0', commit: '', built_at: '' });
			}
			if (path === '/api/v1/settings/public') {
				return jsonResponse({
					'ui.server_name': 'Mi cine',
					'ui.nav_layout': 'top',
					'ui.show_series': false,
					'ui.accent': 'dorado'
				});
			}
			if (path === '/api/v1/auth/me') {
				return jsonResponse({ role: 'admin' });
			}
			throw new Error(`URL inesperada: ${path}`);
		});

		const result = (await load({
			locals: { sessionToken: 'token-123' },
			fetch: fetchMock,
			url: new URL('http://localhost/movies')
		} as never)) as {
			ui: typeof DEFAULT_PUBLIC_UI;
			isAdmin: boolean;
			activeCategory: string | null;
		};

		expect(result.ui.server_name).toBe('Mi cine');
		expect(result.ui.nav_layout).toBe('top');
		expect(result.ui.show_series).toBe(false);
		expect(result.ui.accent).toBe('dorado');
		expect(result.ui.tagline).toBe(DEFAULT_PUBLIC_UI.tagline);
		expect(result.isAdmin).toBe(true);
		expect(result.activeCategory).toBe('movies');
	});

	it('activa Vídeos en el layout solo si hay bibliotecas de clips con vídeos', async () => {
		const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
			const path = new URL(String(input)).pathname;
			if (path === '/api/v1/system/version') {
				return jsonResponse({ server: '999.0.0', commit: '', built_at: '' });
			}
			if (path === '/api/v1/settings/public') {
				return jsonResponse({});
			}
			if (path === '/api/v1/auth/me') {
				return jsonResponse({ role: 'user' });
			}
			if (path === '/api/v1/libraries') {
				return jsonResponse({
					libraries: [
						{ id: 4, name: 'Vídeos', kind: 'clips', item_count: 2 },
						{ id: 5, name: 'Vacía', kind: 'clips', item_count: 0 }
					]
				});
			}
			throw new Error(`URL inesperada: ${path}`);
		});

		const result = (await load({
			locals: { sessionToken: 'token-123' },
			fetch: fetchMock,
			url: new URL('http://localhost/movies')
		} as never)) as { hasClips: boolean };

		expect(result.hasClips).toBe(true);
	});

	it('tolera que falle el listado de bibliotecas al calcular hasClips', async () => {
		const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
			const path = new URL(String(input)).pathname;
			if (path === '/api/v1/system/version') {
				return jsonResponse({ server: '999.0.0', commit: '', built_at: '' });
			}
			if (path === '/api/v1/settings/public') {
				return jsonResponse({});
			}
			if (path === '/api/v1/auth/me') {
				return jsonResponse({ role: 'user' });
			}
			return jsonResponse({ error: 'fallo' }, 502);
		});

		const result = (await load({
			locals: { sessionToken: 'token-123' },
			fetch: fetchMock,
			url: new URL('http://localhost/')
		} as never)) as { hasClips: boolean };

		expect(result.hasClips).toBe(false);
	});

	it('vuelve a los valores por defecto si falla la API de ajustes', async () => {
		const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
			const path = new URL(String(input)).pathname;
			if (path === '/api/v1/system/version') {
				return jsonResponse({ server: '999.0.0', commit: '', built_at: '' });
			}
			if (path === '/api/v1/settings/public') {
				return jsonResponse({ error: 'fallo' }, 500);
			}
			if (path === '/api/v1/auth/me') {
				return jsonResponse({ role: 'viewer' });
			}
			throw new Error(`URL inesperada: ${path}`);
		});

		const result = (await load({
			locals: { sessionToken: 'token-123' },
			fetch: fetchMock,
			url: new URL('http://localhost/')
		} as never)) as { ui: typeof DEFAULT_PUBLIC_UI };

		expect(result.ui).toEqual(DEFAULT_PUBLIC_UI);
	});
});

describe('pie de página', () => {
	it('muestra la versión del servidor y el commit', () => {
		const body = renderFooter(webVersion, 'abc1234');

		expect(visibleText(body)).toContain(`Cinecat ${webVersion} (abc1234) · GPL-3.0`);
		expect(body).toContain('Servidor:');
		expect(body).not.toContain('La web y el servidor tienen versiones distintas');
	});

	it('avisa de forma visible si las versiones no coinciden', () => {
		const body = renderFooter('999.0.0', 'def5678');

		expect(visibleText(body)).toContain('Cinecat 999.0.0 (def5678) · GPL-3.0');
		expect(body).toContain(
			'La web y el servidor tienen versiones distintas: recarga o actualiza.'
		);
		expect(body).toContain('role="status"');
	});

	it('usa la versión web y no avisa si el servidor no responde', () => {
		const body = renderFooter(null);

		expect(visibleText(body)).toContain(`Cinecat ${webVersion} · GPL-3.0`);
		expect(body).not.toContain('La web y el servidor tienen versiones distintas');
	});

	it('marca Películas como categoría activa y muestra el resto de enlaces', () => {
		const { body } = render(Layout, {
			props: {
				data: { ...layoutData(null), activeCategory: 'movies' },
				children
			}
		});

		expect(body).toContain('href="/movies"');
		expect(body).toContain('aria-current="page"');
		expect(body).toContain('Colecciones');
		expect(body).toContain('href="/series"');
		expect(body).not.toContain('Próximamente');
	});

	it('marca el enlace de Colecciones como activo', () => {
		const { body } = render(Layout, {
			props: {
				data: { ...layoutData(null), activeCategory: 'collections' },
				children
			}
		});

		expect(body).toContain('href="/collections"');
		expect(body).toContain('aria-current="page"');
	});

	it('marca el enlace de Series como activo', () => {
		const { body } = render(Layout, {
			props: {
				data: { ...layoutData(null), activeCategory: 'series' },
				children
			}
		});

		expect(body).toContain('href="/series"');
		expect(body).toContain('aria-current="page"');
	});
});

describe('interfaz configurable', () => {
	it('aplica el nombre, el eslogan y el pie de los ajustes', () => {
		const { body, head } = render(Layout, {
			props: {
				data: {
					...layoutData(webVersion),
					ui: uiWith({
						server_name: 'Mi cine',
						tagline: 'Siempre en sesión'
					})
				},
				children
			}
		});

		expect(head).toContain('<title>Mi cine</title>');
		expect(body).toContain('Mi cine');
		expect(body).toContain('Siempre en sesión');
		expect(visibleText(body)).toContain(`Mi cine ${webVersion} · GPL-3.0`);
	});

	it('escapa el nombre y el eslogan', () => {
		const { body } = render(Layout, {
			props: {
				data: {
					...layoutData(null),
					ui: uiWith({
						server_name: '<b>Cine</b>',
						tagline: '<script>alert(1)</script>'
					})
				},
				children
			}
		});

		expect(body).toContain('&lt;b>Cine&lt;/b>');
		expect(body).toContain('&lt;script>alert(1)&lt;/script>');
		expect(body).not.toContain('<b>Cine</b>');
	});

	it('pinta la barra lateral por defecto con la navegación principal', () => {
		const { body } = render(Layout, {
			props: {
				data: { ...layoutData(null), isHome: true },
				children
			}
		});

		expect(body).toContain('app-frame-side');
		expect(body).toContain('side-nav');
		expect(body).toContain('<nav aria-label="Principal"');
		expect(body).toContain('href="/"');
		expect(body).toContain('aria-current="page"');
		expect(body).toContain('Sesión continua');
		expect(body).not.toContain('class="marquee"');
	});

	it('publica las columnas de rejilla y de fila en el contenedor principal', () => {
		const { body } = render(Layout, {
			props: {
				data: {
					...layoutData(null),
					ui: uiWith({ grid_columns: 5, row_columns: 4 })
				},
				children
			}
		});

		expect(body).toContain('--grid-cols: 5');
		expect(body).toContain('--ui-grid-cols: 5');
		expect(body).toContain('--row-cols: 4');
		expect(body).toContain('--ui-row-cols: 4');
	});

	it('usa la cabecera superior si nav_layout es top y añade Inicio', () => {
		const { body } = render(Layout, {
			props: {
				data: {
					...layoutData(null),
					ui: uiWith({ nav_layout: 'top', tagline: '' })
				},
				children
			}
		});

		expect(body).toContain('class="marquee"');
		expect(body).toContain('<nav class="category-nav" aria-label="Principal"');
		expect(body).toContain('>Inicio</a>');
		expect(body).not.toContain('side-nav');
		expect(body).not.toContain('Sesión continua');
	});

	it('oculta Colecciones y Series del menú cuando los ajustes lo piden', () => {
		const { body } = render(Layout, {
			props: {
				data: {
					...layoutData(null),
					ui: uiWith({ show_collections: false, show_series: false })
				},
				children
			}
		});

		expect(body).not.toContain('href="/collections"');
		expect(body).not.toContain('href="/series"');
		expect(body).toContain('href="/movies"');
	});

	it('muestra Vídeos en el menú lateral solo si hay clips', () => {
		const withClips = render(Layout, {
			props: {
				data: { ...layoutData(null), hasClips: true, activeCategory: 'videos' },
				children
			}
		}).body;
		expect(withClips).toContain('href="/videos"');
		expect(withClips).toContain('Vídeos');
		expect(withClips).toContain('aria-current="page"');

		const withoutClips = render(Layout, {
			props: { data: layoutData(null), children }
		}).body;
		expect(withoutClips).not.toContain('href="/videos"');
	});

	it('muestra Vídeos en el menú superior solo si hay clips', () => {
		const withClips = render(Layout, {
			props: {
				data: {
					...layoutData(null),
					hasClips: true,
					activeCategory: 'videos',
					ui: uiWith({ nav_layout: 'top' })
				},
				children
			}
		}).body;
		expect(withClips).toContain('class="marquee"');
		expect(withClips).toContain('href="/videos"');
		expect(withClips).toContain('aria-current="page"');

		const withoutClips = render(Layout, {
			props: {
				data: { ...layoutData(null), ui: uiWith({ nav_layout: 'top' }) },
				children
			}
		}).body;
		expect(withoutClips).not.toContain('href="/videos"');
	});

	it('solo muestra Administración a los administradores y Mi cuenta con sesión', () => {
		const user = render(Layout, {
			props: {
				data: { ...layoutData(null), hasSession: true, isAdmin: false },
				children
			}
		}).body;
		expect(user).not.toContain('href="/admin"');
		expect(user).toContain('href="/account"');

		const admin = render(Layout, {
			props: {
				data: { ...layoutData(null), hasSession: true, isAdmin: true },
				children
			}
		}).body;
		expect(admin).toContain('href="/admin"');
		expect(admin).toContain('href="/account"');
	});
});
