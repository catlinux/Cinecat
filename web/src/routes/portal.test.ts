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
	poster_custom: false,
	original_title: 'The Godfather',
	imdb_id: 'tt0068646',
	tagline: 'Una oferta que no podrás rechazar.',
	genres: ['Drama', 'Crimen'],
	runtime_minutes: 175,
	rating: 8.7,
	directors: ['Francis Ford Coppola'],
	cast: ['Marlon Brando', 'Al Pacino'],
	backdrop_url: null,
	backdrop_custom: false,
	overview_language: 'es-ES',
	match_status: 'auto',
	match_score: 1,
	watched: false,
	progress: null
};

const continueItem: MediaItem = {
	...recentItem,
	id: 10,
	title: 'El Padrino: Parte II',
	file_title: 'El Padrino Parte II',
	progress: {
		position_seconds: 4320,
		duration_seconds: 7200,
		updated_at: '2026-10-26T10:00:00Z'
	}
};

const nextEpisode: MediaItem = {
	...recentItem,
	id: 11,
	title: 'El episodio',
	file_title: 'El episodio',
	kind: 'episode',
	season: 2,
	episode: 5,
	progress: {
		position_seconds: 300,
		duration_seconds: 1800,
		updated_at: '2026-10-26T10:01:00Z'
	}
};

function homeResponse(overrides: Record<string, unknown> = {}) {
	return {
		continue: [continueItem],
		next_up: [],
		recent_movies: [recentItem],
		counts: { movies: 42, episodes: 2 },
		...overrides
	};
}

function loadEvent(fetchImpl: typeof fetch, url = new URL('http://localhost/')) {
	return {
		locals: { sessionToken: 'token-123' },
		fetch: fetchImpl,
		url
	};
}

function portalData(overrides: Record<string, unknown> = {}) {
	return {
		hasSession: true,
		isAdmin: false,
		activeCategory: null,
		serverVersion: null,
		serverCommit: null,
		webVersion,
		moviesTotal: 42,
		continueWatching: [continueItem],
		nextUp: [],
		recent: [recentItem],
		errorMessage: null,
		...overrides
	};
}

describe('carga del portal', () => {
	it('carga el resumen del home con una sola llamada', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse(homeResponse()));

		const result = (await load(loadEvent(fetchMock) as never)) as {
			moviesTotal: number;
			continueWatching: MediaItem[];
			nextUp: MediaItem[];
			recent: MediaItem[];
			errorMessage: string | null;
		};

		expect(result.moviesTotal).toBe(42);
		expect(result.continueWatching).toHaveLength(1);
		expect(result.nextUp).toEqual([]);
		expect(result.recent).toHaveLength(1);
		expect(result.errorMessage).toBeNull();
		expect(fetchMock).toHaveBeenCalledTimes(1);
		expect(String(fetchMock.mock.calls[0][0])).toBe('http://api.test/api/v1/home');
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

	it('devuelve un estado vacío con mensaje si la API falla', async () => {
		const fetchMock = vi.fn().mockRejectedValue(new Error('sin conexión'));

		const result = (await load(loadEvent(fetchMock) as never)) as {
			moviesTotal: number;
			continueWatching: MediaItem[];
			nextUp: MediaItem[];
			recent: MediaItem[];
			errorMessage: string | null;
		};

		expect(result.moviesTotal).toBe(0);
		expect(result.continueWatching).toEqual([]);
		expect(result.nextUp).toEqual([]);
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
	it('muestra Continuar viendo antes de las recientes', () => {
		const { body } = render(PortalPage, { props: { data: portalData() } });

		expect(body).toContain('Tu cine de barrio');
		expect(body).toContain('Películas');
		expect(body).toContain('42 películas');
		expect(body).toContain('Colecciones');
		expect(body).toContain('Series');
		expect(body).toContain('Próximamente');
		expect(body).toContain('Continuar viendo');
		expect(body).toContain('Continúas por 1 h 12 min');
		expect(body).toContain('Añadido recientemente');
		expect(body).toContain('El Padrino');
		expect(body).toContain('Ver todas');
		expect(body.indexOf('Continuar viendo')).toBeLessThan(body.indexOf('Añadido recientemente'));
		expect(body).not.toContain('Siguiente episodio');
	});

	it('no muestra Continuar viendo si no hay elementos', () => {
		const { body } = render(PortalPage, {
			props: { data: portalData({ continueWatching: [] }) }
		});

		expect(body).not.toContain('Continuar viendo');
		expect(body).toContain('Añadido recientemente');
	});

	it('muestra Siguiente episodio cuando next_up trae elementos', () => {
		const { body } = render(PortalPage, {
			props: { data: portalData({ nextUp: [nextEpisode] }) }
		});

		expect(body).toContain('Siguiente episodio');
		expect(body).toContain('Serie · T2 E5');
	});

	it('muestra el error de la API sin ocultar el portal', () => {
		const { body } = render(PortalPage, {
			props: {
				data: portalData({
					moviesTotal: 0,
					continueWatching: [],
					nextUp: [],
					recent: [],
					errorMessage: 'No se pudo cargar el portal.'
				})
			}
		});

		expect(body).toContain('No se pudo cargar el portal.');
		expect(body).toContain('Abrir cartelera');
		expect(body).toContain('Próximamente');
	});
});
