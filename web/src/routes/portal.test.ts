import { render } from 'svelte/server';
import { describe, expect, it, vi } from 'vitest';
import type { MediaItem, Series } from '$lib/api';
import { DEFAULT_PUBLIC_UI } from '$lib/settings';
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
	progress: null,
	collections: [],
	series_id: null,
	series_title: null,
	episode_title: null,
	still_url: null
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

const clipContinue: MediaItem = {
	...continueItem,
	id: 30,
	library_id: 4,
	library_kind: 'clips',
	title: 'Concierto de verano',
	file_title: 'Concierto de verano',
	year: null,
	tmdb_id: null,
	overview: null,
	poster_url: null,
	original_title: null,
	imdb_id: null,
	tagline: null,
	genres: [],
	runtime_minutes: null,
	rating: null,
	directors: [],
	cast: [],
	backdrop_url: null,
	overview_language: null,
	match_status: 'none',
	match_score: null
};

const nextEpisode: MediaItem = {
	...recentItem,
	id: 11,
	title: 'El episodio',
	file_title: 'El episodio',
	kind: 'episode',
	season: 2,
	episode: 5,
	series_title: 'The Big Bang Theory',
	episode_title: 'El episodio',
	still_url: 'https://images.example/episodio.jpg',
	progress: {
		position_seconds: 300,
		duration_seconds: 1800,
		updated_at: '2026-10-26T10:01:00Z'
	}
};

const recentSeries: Series = {
	id: 3,
	title: 'The Big Bang Theory',
	file_title: 'The Big Bang theory',
	year: 2007,
	poster_url: 'https://images.example/serie.jpg',
	genres: ['Comedia'],
	season_count: 2,
	episode_count: 12,
	watched_count: 4,
	match_status: 'auto'
};

function homeResponse(overrides: Record<string, unknown> = {}) {
	return {
		continue: [continueItem],
		next_up: [],
		recent_movies: [recentItem],
		recent_series: [recentSeries],
		has_clips: false,
		counts: { movies: 42, episodes: 2, series: 5 },
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
		hasVideo: true,
		hasAnyLibraryAccess: true,
		isHome: false,
		activeCategory: null,
		serverVersion: null,
		serverCommit: null,
		webVersion,
		ui: DEFAULT_PUBLIC_UI,
		moviesTotal: 42,
		collectionsTotal: 7,
		seriesTotal: 5,
		hasClips: false,
		hasMusic: false,
		continueWatching: [continueItem],
		nextUp: [],
		recent: [recentItem],
		recentSeries: [recentSeries],
		watchlist: [],
		favorites: [],
		errorMessage: null,
		...overrides
	};
}

function collectionsResponse(total = 7): Response {
	return jsonResponse({ collections: [], total, limit: 1, offset: 0 });
}

function seriesResponse(total = 9): Response {
	return jsonResponse({ series: [], total, limit: 1, offset: 0 });
}

describe('carga del portal', () => {
	it('carga el resumen del home y el total de colecciones', async () => {
		const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
			const url = new URL(String(input));
			if (url.pathname === '/api/v1/collections') {
				return collectionsResponse();
			}
			if (url.pathname === '/api/v1/home') {
				return jsonResponse(homeResponse());
			}
			throw new Error(`URL inesperada: ${url}`);
		});

		const result = (await load(loadEvent(fetchMock) as never)) as {
			moviesTotal: number;
			collectionsTotal: number | null;
			seriesTotal: number | null;
			hasClips: boolean;
			continueWatching: MediaItem[];
			nextUp: MediaItem[];
			recent: MediaItem[];
			recentSeries: Series[];
			errorMessage: string | null;
		};

		expect(result.moviesTotal).toBe(42);
		expect(result.collectionsTotal).toBe(7);
		expect(result.seriesTotal).toBe(5);
		expect(result.hasClips).toBe(false);
		expect(result.continueWatching).toHaveLength(1);
		expect(result.nextUp).toEqual([]);
		expect(result.recent).toHaveLength(1);
		expect(result.recentSeries).toHaveLength(1);
		expect(result.recentSeries[0].title).toBe('The Big Bang Theory');
		expect(result.errorMessage).toBeNull();
		expect(fetchMock).toHaveBeenCalledTimes(2);
	});

	it('tolera un servidor antiguo sin recent_series', async () => {
		const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
			const url = new URL(String(input));
			if (url.pathname === '/api/v1/collections') {
				return collectionsResponse();
			}
			if (url.pathname === '/api/v1/home') {
				return jsonResponse(homeResponse({ recent_series: undefined }));
			}
			throw new Error(`URL inesperada: ${url}`);
		});

		const result = (await load(loadEvent(fetchMock) as never)) as {
			recentSeries: Series[];
			errorMessage: string | null;
		};

		expect(result.recentSeries).toEqual([]);
		expect(result.errorMessage).toBeNull();
	});

	it('mantiene el enlace de colecciones sin recuento si su API falla', async () => {
		const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
			const url = new URL(String(input));
			if (url.pathname === '/api/v1/collections') {
				return jsonResponse({ error: 'sin colecciones' }, 502);
			}
			if (url.pathname === '/api/v1/home') {
				return jsonResponse(homeResponse());
			}
			throw new Error(`URL inesperada: ${url}`);
		});

		const result = (await load(loadEvent(fetchMock) as never)) as {
			collectionsTotal: number | null;
			errorMessage: string | null;
		};

		expect(result.collectionsTotal).toBeNull();
		expect(result.errorMessage).toBeNull();
	});

	it('mantiene el recuento de series cuando falla el home', async () => {
		const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
			const url = new URL(String(input));
			if (url.pathname === '/api/v1/collections') {
				return collectionsResponse();
			}
			if (url.pathname === '/api/v1/home') {
				return jsonResponse({ error: 'home roto' }, 502);
			}
			if (url.pathname === '/api/v1/series') {
				return seriesResponse(9);
			}
			throw new Error(`URL inesperada: ${url}`);
		});

		const result = (await load(loadEvent(fetchMock) as never)) as {
			seriesTotal: number | null;
			errorMessage: string | null;
		};

		expect(result.seriesTotal).toBe(9);
		expect(result.errorMessage).toBeTruthy();
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
			collectionsTotal: number | null;
			continueWatching: MediaItem[];
			nextUp: MediaItem[];
			recent: MediaItem[];
			recentSeries: Series[];
			errorMessage: string | null;
		};

		expect(result.moviesTotal).toBe(0);
		expect(result.collectionsTotal).toBeNull();
		expect(result.continueWatching).toEqual([]);
		expect(result.nextUp).toEqual([]);
		expect(result.recent).toEqual([]);
		expect(result.recentSeries).toEqual([]);
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

		expect(body).toContain('Tu cine en casa');
		expect(body).toContain('Una biblioteca personal con olor a palomitas');
		expect(body).toContain('Películas');
		expect(body).toContain('42 películas');
		expect(body).toContain('Colecciones');
		expect(body).toContain('href="/collections"');
		expect(body).toContain('7 colecciones');
		expect(body).toContain('href="/series"');
		expect(body).toContain('5 series');
		expect(body).toContain('Series');
		expect(body).not.toContain('Próximamente');
		expect(body).toContain('class="portal-categories"');
		expect((body.match(/class="portal-category(?:\s|")/g) ?? []).length).toBe(3);
		expect(body).toContain('Continuar viendo');
		expect(body).toContain('Continúas por 1 h 12 min');
		expect(body).toContain('Películas añadidas recientemente');
		expect(body).toContain('Series añadidas recientemente');
		expect(body).toContain('El Padrino');
		expect(body).toContain('The Big Bang Theory');
		expect(body).toContain('href="/series?sort=added"');
		expect(body).toContain('Ver todas');
		expect(body).toContain('class="row-scroller');
		expect(body).toContain('aria-label="Anterior"');
		expect(body).toContain('aria-label="Siguiente"');
		expect(body).toMatch(/class="row-arrows[^"]*"[^>]*\bhidden\b/);
		expect(body.indexOf('Continuar viendo')).toBeLessThan(
			body.indexOf('Películas añadidas recientemente')
		);
		expect(body).not.toContain('Siguiente episodio');
	});

	it('añade la categoría Vídeos y usa VideoCard en Continuar viendo', () => {
		const withClips = render(PortalPage, {
			props: {
				data: portalData({ hasClips: true, continueWatching: [clipContinue] })
			}
		}).body;

		expect(withClips).toContain('href="/videos"');
		expect(withClips).toContain('Videoclips y conciertos');
		expect(withClips).toContain('Concierto de verano');
		expect(withClips).toContain('class="video-card');
		expect(withClips).toContain('href="/items/30"');
		expect((withClips.match(/class="portal-category(?:\s|")/g) ?? []).length).toBe(4);
	});

	it('añade la categoría Música cuando hay pistas', () => {
		const withMusic = render(PortalPage, {
			props: { data: portalData({ hasMusic: true }) }
		}).body;

		expect(withMusic).toContain('href="/music"');
		expect(withMusic).toContain('Discoteca');
		expect(withMusic).toContain('Álbumes, artistas y la banda sonora de la casa');
		expect((withMusic.match(/class="portal-category(?:\s|")/g) ?? []).length).toBe(4);
	});

	it('aplica el nombre, el título y la introducción de los ajustes', () => {
		const { body, head } = render(PortalPage, {
			props: {
				data: portalData({
					ui: {
						...DEFAULT_PUBLIC_UI,
						server_name: 'Mi cine',
						portal_title: 'La sala de casa',
						portal_intro: 'Una sala <luminosa> y tranquila.'
					}
				}) as never
			}
		});

		expect(head).toContain('<title>La sala de casa · Mi cine</title>');
		expect(body).toContain('La sala de casa');
		expect(body).toContain('Una sala &lt;luminosa> y tranquila.');
		expect(body).not.toContain('<luminosa>');
	});

	it('oculta la introducción si el ajuste está vacío', () => {
		const { body } = render(PortalPage, {
			props: { data: portalData({ ui: { ...DEFAULT_PUBLIC_UI, portal_intro: '' } }) as never }
		});

		expect(body).not.toContain('portal-lede');
		expect(body).toContain('Tu cine en casa');
	});

	it('mantiene Colecciones activa aunque no haya recuento', () => {
		const { body } = render(PortalPage, {
			props: { data: portalData({ collectionsTotal: null }) }
		});

		expect(body).toContain('href="/collections"');
		expect(body).toContain('Colecciones');
		expect(body).toContain('Entrar');
	});

	it('mantiene Series activa aunque no haya recuento', () => {
		const { body } = render(PortalPage, {
			props: { data: portalData({ seriesTotal: null }) }
		});

		expect(body).toContain('href="/series"');
		expect(body).toContain('Series');
		expect(body).toContain('Entrar');
	});

	it('separa Continuar viendo en películas y series', () => {
		const episode: MediaItem = { ...continueItem, id: 11, title: 'Piloto', kind: 'episode' };
		const both = render(PortalPage, {
			props: { data: portalData({ continueWatching: [continueItem, episode] }) }
		}).body;
		expect(both).toContain('Continuar viendo películas');
		expect(both).toContain('Continuar viendo series');

		const onlyMovies = render(PortalPage, {
			props: { data: portalData({ continueWatching: [continueItem] }) }
		}).body;
		expect(onlyMovies).toContain('Continuar viendo películas');
		expect(onlyMovies).not.toContain('Continuar viendo series');
	});

	it('no muestra Continuar viendo si no hay elementos', () => {
		const { body } = render(PortalPage, {
			props: { data: portalData({ continueWatching: [] }) }
		});

		expect(body).not.toContain('Continuar viendo');
		expect(body).toContain('Películas añadidas recientemente');
	});

	it('no muestra la fila de series si está vacía', () => {
		const { body } = render(PortalPage, {
			props: { data: portalData({ recentSeries: [] }) }
		});

		expect(body).not.toContain('Series añadidas recientemente');
		expect(body).toContain('Películas añadidas recientemente');
	});

	it('oculta la fila de series si ui.show_series está desactivado', () => {
		const { body } = render(PortalPage, {
			props: {
				data: portalData({
					ui: { ...DEFAULT_PUBLIC_UI, show_series: false }
				})
			}
		});

		expect(body).not.toContain('Series añadidas recientemente');
	});

	it('muestra Siguiente episodio cuando next_up trae elementos', () => {
		const upcoming = { ...nextEpisode, progress: null };
		const { body } = render(PortalPage, {
			props: { data: portalData({ nextUp: [upcoming] }) }
		});

		expect(body).toContain('Siguiente episodio');
		expect(body).toContain('The Big Bang Theory · T2 E5');
		expect(body).toContain('El episodio');
		expect(body).toContain('/items/11');
		expect(body).toContain('episode-card');
		expect(body).toContain('https://images.example/episodio.jpg');
		expect(body).toContain('EPISODIO');
		expect(body).toContain('row-wide');
	});

	it('usa el cartel de la serie y después el respaldo en el siguiente episodio', () => {
		const withPoster = {
			...nextEpisode,
			still_url: null,
			series_poster_url: 'https://images.example/serie.jpg'
		};
		const posterBody = render(PortalPage, {
			props: { data: portalData({ nextUp: [withPoster] }) }
		}).body;
		expect(posterBody).toContain('src="https://images.example/serie.jpg"');

		const withoutArt = { ...withPoster, series_poster_url: null };
		const fallbackBody = render(PortalPage, {
			props: { data: portalData({ nextUp: [withoutArt] }) }
		}).body;
		expect(fallbackBody).toContain('episode-fallback');
		expect(fallbackBody).toContain('El episodio');
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
		expect(body).toContain('href="/series"');
	});
});

describe('filas de listas personales del portal', () => {
	const favoriteItem = { ...recentItem, id: 51, title: 'Favorita', favorite: true };
	const favoriteClip = { ...clipContinue, id: 52, title: 'Vídeo favorito', favorite: true };
	const favoriteSeries = { ...recentSeries, id: 53, title: 'Serie favorita', favorite: true };

	it('muestra próximas a ver y favoritos mezclando ítems y series', () => {
		const { body } = render(PortalPage, {
			props: {
				data: portalData({
					watchlist: [
						{ kind: 'item', item: { ...recentItem, id: 50, title: 'Próxima' } },
						{ kind: 'series', series: { ...recentSeries, id: 54, title: 'Serie próxima' } }
					],
					favorites: [
						{ kind: 'item', item: favoriteItem },
						{ kind: 'item', item: favoriteClip },
						{ kind: 'series', series: favoriteSeries }
					]
				})
			}
		});

		expect(body).toContain('Próximas a ver');
		expect(body).toContain('href="/watchlist"');
		expect(body).toContain('Tus favoritos');
		expect(body).toContain('href="/favorites"');
		expect(body).toContain('Próxima');
		expect(body).toContain('Serie próxima');
		expect(body).toContain('Favorita');
		expect(body).toContain('Vídeo favorito');
		expect(body).toContain('Serie favorita');
		expect(body).toContain('class="video-card');
		expect(body).toContain('class="series-card');
		expect(body.indexOf('Continuar viendo')).toBeLessThan(body.indexOf('Próximas a ver'));
		expect(body.indexOf('Series añadidas recientemente')).toBeLessThan(body.indexOf('Tus favoritos'));
	});

	it('tolera listas ausentes y respeta show_series', () => {
		const withoutLists = portalData({ watchlist: undefined, favorites: undefined });
		const { body } = render(PortalPage, { props: { data: withoutLists } });
		expect(body).not.toContain('Tus favoritos');
		expect(body).not.toContain('Próximas a ver');

		const hiddenSeries = render(PortalPage, {
			props: {
				data: portalData({
					ui: { ...DEFAULT_PUBLIC_UI, show_series: false },
					watchlist: [
						{ kind: 'item', item: favoriteItem },
						{ kind: 'series', series: favoriteSeries }
					]
				})
			}
		}).body;
		expect(hiddenSeries).toContain('Favorita');
		expect(hiddenSeries).not.toContain('Serie favorita');
	});
});

describe('portal sin bibliotecas', () => {
	it('muestra un mensaje claro y no pinta secciones vacías', () => {
		const { body } = render(PortalPage, {
			props: {
				data: portalData({
					hasVideo: false,
					hasClips: false,
					hasMusic: false,
					hasAnyLibraryAccess: false,
					moviesTotal: 0,
					collectionsTotal: 0,
					seriesTotal: 0,
					continueWatching: [],
					nextUp: [],
					recent: [],
					recentSeries: [],
					watchlist: [],
					favorites: []
				})
			}
		});
		expect(body).toContain('Todavía no tienes ninguna biblioteca asignada');
		expect(body).toContain('Pídele acceso al administrador');
		expect(body).not.toContain('portal-categories');
	});
});
