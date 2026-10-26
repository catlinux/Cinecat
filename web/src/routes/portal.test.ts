import { render } from 'svelte/server';
import { describe, expect, it, vi } from 'vitest';
import type { MediaItem } from '$lib/api';
import { webVersion } from '$lib/version';
import PortalPage from './+page.svelte';
import { load } from './+page.server';

function jsonResponse(body: unknown, status = 200): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

const recentItem: MediaItem = {
	id: 9,
	library_id: 1,
	path: '/media/peliculas/El Padrino (1972).mkv',
	title: 'El Padrino',
	file_title: 'El Padrino',
	year: 1972,
	kind: 'movie',
	season: null,
	episode: null,
	tmdb_id: 238,
	overview: 'Una familia muy unida.',
	poster_url: null,
	original_title: 'The Godfather',
	imdb_id: 'tt0068646',
	tagline: 'Una oferta que no podrás rechazar.',
	genres: ['Drama', 'Crimen'],
	runtime_minutes: 175,
	rating: 8.7,
	directors: ['Francis Ford Coppola'],
	cast: ['Marlon Brando', 'Al Pacino'],
	backdrop_url: null,
	overview_language: 'es-ES',
	match_status: 'auto',
	match_score: 1,
	watched: false
};

function loadEvent(fetchImpl: typeof fetch, url = new URL('http://localhost/')) {
	return {
		locals: { sessionToken: 'token-123' },
		fetch: fetchImpl,
		url
	};
}

describe('carga del portal', () => {
	it('carga el total de películas y las recientes', async () => {
		const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
			const url = String(input);
			if (url.endsWith('/api/v1/items?kind=movie&limit=1')) {
				return jsonResponse({ items: [], total: 42, limit: 1, offset: 0 });
			}
			if (url.endsWith('/api/v1/items?kind=movie&sort=added&limit=12')) {
				return jsonResponse({ items: [recentItem], total: 42, limit: 12, offset: 0 });
			}
			throw new Error(`URL inesperada: ${url}`);
		});

		const result = (await load(loadEvent(fetchMock) as never)) as {
			moviesTotal: number;
			recent: MediaItem[];
			errorMessage: string | null;
		};

		expect(result.moviesTotal).toBe(42);
		expect(result.recent).toHaveLength(1);
		expect(result.errorMessage).toBeNull();
		expect(fetchMock).toHaveBeenCalledTimes(2);
	});

	it('redirige la antigua cartelera conservando la consulta', async () => {
		const fetchMock = vi.fn();
		const url = new URL('http://localhost/?q=matrix&page=2');

		await expect(
			load({ ...loadEvent(fetchMock, url), locals: { sessionToken: null } } as never)
		).rejects.toMatchObject({
			status: 308,
			location: '/movies?q=matrix&page=2'
		});
		expect(fetchMock).not.toHaveBeenCalled();
	});

	it('devuelve un mensaje de error sin romper la carga', async () => {
		const fetchMock = vi.fn().mockRejectedValue(new Error('sin conexión'));

		const result = (await load(loadEvent(fetchMock) as never)) as {
			moviesTotal: number;
			recent: MediaItem[];
			errorMessage: string | null;
		};

		expect(result.moviesTotal).toBe(0);
		expect(result.recent).toEqual([]);
		expect(result.errorMessage).toBeTruthy();
	});

	it('redirige al login si no hay sesión', async () => {
		await expect(
			load({
				locals: { sessionToken: null },
				fetch: vi.fn(),
				url: new URL('http://localhost/')
			} as never)
		).rejects.toMatchObject({ status: 303, location: '/login' });
	});
});

describe('portal', () => {
	it('muestra las categorías, el recuento y las recientes', () => {
		const { body } = render(PortalPage, {
			props: {
				data: {
					hasSession: true,
					isAdmin: false,
					activeCategory: null,
					serverVersion: null,
					serverCommit: null,
					webVersion,
					moviesTotal: 42,
					recent: [recentItem],
					errorMessage: null
				}
			}
		});

		expect(body).toContain('Tu cine de barrio');
		expect(body).toContain('Películas');
		expect(body).toContain('42 películas');
		expect(body).toContain('Colecciones');
		expect(body).toContain('Series');
		expect(body).toContain('Próximamente');
		expect(body).toContain('Añadido recientemente');
		expect(body).toContain('El Padrino');
		expect(body).toContain('Ver todas');
	});

	it('muestra el error de la API sin ocultar el portal', () => {
		const { body } = render(PortalPage, {
			props: {
				data: {
					hasSession: true,
					isAdmin: false,
					activeCategory: null,
					serverVersion: null,
					serverCommit: null,
					webVersion,
					moviesTotal: 0,
					recent: [],
					errorMessage: 'No se pudo cargar el portal.'
				}
			}
		});

		expect(body).toContain('No se pudo cargar el portal.');
		expect(body).toContain('Abrir cartelera');
		expect(body).toContain('Próximamente');
	});
});
