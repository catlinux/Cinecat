import { render } from 'svelte/server';
import { describe, expect, it, vi } from 'vitest';
import type { Candidate, MediaItem } from '$lib/api';
import ItemPage from './+page.svelte';
import { actions, load } from './+page.server';

function jsonResponse(body: unknown, status = 200): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

function makeItem(overrides: Partial<MediaItem> = {}): MediaItem {
	return {
		id: 7,
		library_id: 1,
		path: '/media/peliculas/El Padrino (1972).mkv',
		title: 'El Padrino',
		file_title: 'El Padrino (1972)',
		year: 1972,
		kind: 'movie',
		season: null,
		episode: null,
		tmdb_id: 238,
		overview: 'Una familia muy unida.',
		poster_url: 'https://images.example/poster.jpg',
		poster_custom: false,
		original_title: 'The Godfather',
		imdb_id: 'tt0068646',
		tagline: 'Una oferta que no podrás rechazar.',
		genres: ['Drama', 'Crimen'],
		runtime_minutes: 112,
		rating: 7.8,
		directors: ['Francis Ford Coppola'],
		cast: Array.from({ length: 13 }, (_, index) => `Actor ${index + 1}`),
		backdrop_url: 'https://images.example/backdrop.jpg',
		backdrop_custom: false,
		overview_language: 'en-US',
		match_status: 'auto',
		match_score: 0.92,
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

interface RenderOptions {
	isAdmin?: boolean;
	corregir?: boolean;
	form?: unknown;
	similarItems?: unknown;
}

function renderPage(item: MediaItem, options: RenderOptions = {}): string {
	return render(ItemPage, {
		props: {
			data: {
				item,
				isAdmin: options.isAdmin ?? false,
				corregir: options.corregir ?? false,
				similarItems: options.similarItems
			},
			form: options.form ?? null
		} as never
	}).body;
}

function loadEvent(fetchImpl: typeof fetch, path = '/items/7', isAdmin = false) {
	return {
		params: { id: '7' },
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
	id = '7'
) {
	const body = new URLSearchParams(fields);
	return {
		request: new Request(`http://localhost/items/${id}?/${action}`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
			body
		}),
		locals: { sessionToken: 'token-123' },
		fetch: fetchImpl,
		params: { id },
		url: new URL(`http://localhost/items/${id}`)
	};
}

describe('carga de la ficha', () => {
	it('carga la película y el rol del layout', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse(makeItem()));

		const result = (await load(loadEvent(fetchMock, '/items/7?corregir=1', true) as never)) as {
			item: MediaItem;
			isAdmin: boolean;
			corregir: boolean;
		};

		expect(result.item.title).toBe('El Padrino');
		expect(result.isAdmin).toBe(true);
		expect(result.corregir).toBe(true);
		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe('http://api.test/api/v1/items/7');
		expect(init.headers.Authorization).toBe('Bearer token-123');
	});

	it('ignora corregir=1 si el usuario no es administrador', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse(makeItem()));

		const result = (await load(loadEvent(fetchMock, '/items/7?corregir=1', false) as never)) as {
			corregir: boolean;
		};

		expect(result.corregir).toBe(false);
	});

	it('devuelve 404 si el ítem no existe', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ error: 'no existe' }, 404));

		await expect(load(loadEvent(fetchMock) as never)).rejects.toMatchObject({ status: 404 });
	});

	it('redirige al login si la sesión no es válida', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ error: 'sin sesión' }, 401));

		await expect(load(loadEvent(fetchMock) as never)).rejects.toMatchObject({
			status: 303,
			location: '/login'
		});
	});

	it('devuelve 502 si no se puede contactar con la API', async () => {
		const fetchMock = vi.fn().mockRejectedValue(new Error('sin conexión'));

		await expect(load(loadEvent(fetchMock) as never)).rejects.toMatchObject({ status: 502 });
	});

	it('no pide similares ni paneles de identificación para un clip', async () => {
		const clip = makeItem({
			library_kind: 'clips',
			tmdb_id: null,
			poster_url: null,
			overview: null,
			original_title: null,
			imdb_id: null,
			match_status: 'none'
		});
		const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
			const url = new URL(String(input));
			if (url.pathname === '/api/v1/items/7') {
				return jsonResponse(clip);
			}
			if (url.pathname.endsWith('/similar')) {
				throw new Error('no se debe pedir similar');
			}
			return jsonResponse({ error: 'no disponible' }, 503);
		});

		const result = (await load(loadEvent(fetchMock, '/items/7?corregir=1', true) as never)) as {
			similarItems: MediaItem[] | null;
			corregir: boolean;
			imagenes: boolean;
		};

		expect(result.similarItems).toBeNull();
		expect(result.corregir).toBe(false);
		expect(result.imagenes).toBe(false);
		expect(fetchMock.mock.calls.map(([url]) => String(url)).some((url) => url.includes('/similar'))).toBe(
			false
		);
	});
});

describe('ficha de película', () => {
	it('muestra los datos completos, el reparto y los enlaces', () => {
		const body = renderPage(makeItem());

		expect(body).toContain('El Padrino');
		expect(body).toContain('<em>The Godfather</em>');
		expect(body).toContain('1972 · 1 h 52 min · Drama, Crimen · ★ 7,8');
		expect(body).toContain('Una oferta que no podrás rechazar.');
		expect(body).toContain('Sinopsis no disponible en castellano');
		expect(body).toContain('Una familia muy unida.');
		expect(body).toContain('Reparto y equipo');
		expect(body).toContain('Dirección');
		expect(body).toContain('Francis Ford Coppola');
		expect(body).toContain('Actor 13');
		expect(body).toContain('https://www.imdb.com/title/tt0068646/');
		expect(body).toContain('https://www.themoviedb.org/movie/238');
		expect(body).toContain('rel="noopener noreferrer"');
		expect(body).toContain('target="_blank"');
		expect(body).toContain('https://images.example/backdrop.jpg');

		expect(body.indexOf('Actor 1')).toBeLessThan(body.indexOf('Francis Ford Coppola'));
	});

	it('no avisa del idioma si la sinopsis está en castellano', () => {
		const body = renderPage(makeItem({ overview_language: 'es-ES' }));

		expect(body).not.toContain('Sinopsis no disponible en castellano');
	});

	it('muestra «Más como este» al final, antes de los paneles de administración', () => {
		const withSimilar = renderPage(makeItem(), {
			isAdmin: true,
			similarItems: [makeItem({ id: 8, title: 'El padrino II', poster_url: null })]
		});
		expect(withSimilar).toContain('Más como este');
		expect(withSimilar).toContain('El padrino II');
		const similarAt = withSimilar.indexOf('Más como este');
		expect(withSimilar.indexOf('Reparto y equipo')).toBeLessThan(similarAt);
		expect(withSimilar.indexOf('Enlaces')).toBeLessThan(similarAt);
		expect(withSimilar.indexOf('Archivo')).toBeLessThan(similarAt);
		expect(similarAt).toBeLessThan(withSimilar.indexOf('Cambiar imagen'));
		expect(similarAt).toBeLessThan(withSimilar.indexOf('Corregir ficha'));

		const withoutSimilar = renderPage(makeItem(), { similarItems: [] });
		expect(withoutSimilar).not.toContain('Más como este');
	});

	it('convierte Reproducir en Continuar y muestra la barra de progreso', () => {
		const body = renderPage(
			makeItem({
				progress: {
					position_seconds: 4320,
					duration_seconds: 7200,
					updated_at: '2026-10-26T10:00:00Z'
				}
			})
		);

		expect(body).toContain('Continuar · 1 h 12 min');
		expect(body).toContain('Desde el principio');
		expect(body).toContain('/items/7/play?t=0');
		expect(body).toContain('role="progressbar"');
		expect(body).toContain('aria-valuenow="60"');
	});

	it('mantiene Reproducir si no hay progreso', () => {
		const body = renderPage(makeItem());
		expect(body).toContain('Reproducir');
		expect(body).not.toContain('Continuar ·');
	});

	it('enlaza las colecciones a las que pertenece la película', () => {
		const body = renderPage(
			makeItem({ collections: [{ id: 5, name: 'La saga' }] })
		);

		expect(body).toContain('Forma parte de');
		expect(body).toContain('/collections/5');
		expect(body).toContain('La saga');
	});

	it('muestra el botón de visto en sus dos estados', () => {
		const pending = renderPage(makeItem({ watched: false }));
		expect(pending).toContain('Marcar como visto');
		expect(pending).toContain('aria-pressed="false"');
		expect(pending).toContain('name="watched" value="1"');

		const watched = renderPage(makeItem({ watched: true }));
		expect(watched).toContain('Marcada como vista · Desmarcar');
		expect(watched).toContain('aria-pressed="true"');
		expect(watched).toContain('name="watched" value="0"');
	});

	it('se ve bien cuando la película no está identificada', () => {
		const body = renderPage(
			makeItem({
				title: 'Archivo sin identificar',
				file_title: 'Archivo sin identificar',
				original_title: null,
				year: null,
				tmdb_id: null,
				overview: null,
				poster_url: null,
				imdb_id: null,
				tagline: null,
				genres: [],
				runtime_minutes: null,
				rating: null,
				directors: [],
				cast: [],
				backdrop_url: null,
				overview_language: null,
				match_status: null,
				match_score: null
			})
		);

		expect(body).toContain('Archivo sin identificar');
		expect(body).toContain('Sin identificar');
		expect(body).not.toContain('Dirección');
		expect(body).not.toContain('Reparto');
		expect(body).not.toContain('Ficha en IMDb');
	});

	it('muestra la ficha mínima de un clip y oculta identificación, reparto y similares', () => {
		const body = renderPage(
			makeItem({
				library_kind: 'clips',
				title: 'Concierto de verano',
				overview: 'No debería mostrarse',
				poster_url: null,
				backdrop_url: null,
				tmdb_id: null,
				imdb_id: null,
				genres: [],
				directors: [],
				cast: ['No debería mostrarse'],
				collections: [{ id: 5, name: 'No debería mostrarse' }],
				match_status: 'none'
			}),
			{
				isAdmin: true,
				corregir: true,
				similarItems: [makeItem({ id: 99, title: 'Similar oculta' })]
			}
		);

		expect(body).toContain('Concierto de verano');
		expect(body).toContain('← Volver a Vídeos');
		expect(body).toContain('href="/videos"');
		expect(body).toContain('Reproducir');
		expect(body).toContain('Marcar como visto');
		expect(body).toContain('Archivo');
		expect(body).not.toContain('No debería mostrarse');
		expect(body).not.toContain('Sin identificar');
		expect(body).not.toContain('Reparto y equipo');
		expect(body).not.toContain('Más como este');
		expect(body).not.toContain('Forma parte de');
		expect(body).not.toContain('Corregir ficha');
		expect(body).not.toContain('Cambiar imagen');
		expect(body).not.toContain('Ficha en IMDb');
		expect(body).not.toContain('Ficha en TMDB');
	});

	it('muestra el reparto del episodio y mantiene la ficha sencilla', () => {
		const body = renderPage(
			makeItem({
				kind: 'episode',
				season: 2,
				episode: 3,
				title: 'El episodio',
				original_title: 'The Episode',
				still_url: 'https://images.example/episodio.jpg',
				directors: ['Una directora'],
				cast: ['Un actor']
			})
		);

		expect(body).toContain('Temporada 2, episodio 3');
		expect(body).toContain('T2E3');
		expect(body).toContain('episode-hero');
		expect(body).toContain('https://images.example/episodio.jpg');
		expect(body).toContain('Reparto y equipo');
		expect(body).toContain('Una directora');
		expect(body).toContain('Un actor');
		expect(body).not.toContain('Corregir ficha');
	});

	it('solo muestra el panel de corrección a administradores', () => {
		const admin = renderPage(makeItem(), { isAdmin: true, corregir: true });
		expect(admin).toContain('Corregir ficha');
		expect(admin).toContain('Estado actual:');
		expect(admin).toContain('Identificada automáticamente (92 %)');
		expect(admin).toContain('Buscar en TMDB');
		expect(admin).toContain('No es ninguna');
		expect(admin).toContain('Volver a automático');
		expect(admin).toContain('El Padrino (1972)');
		expect(admin).toContain('/media/peliculas/El Padrino (1972).mkv');
		expect(admin).toMatch(/<details[^>]*\bopen\b/);

		const user = renderPage(makeItem(), { isAdmin: false, corregir: true });
		expect(user).not.toContain('Corregir ficha');
	});

	it('lista los candidatos con su sinopsis recortada y el botón para elegirlos', () => {
		const candidate: Candidate = {
			tmdb_id: 238,
			title: 'El Padrino',
			original_title: 'The Godfather',
			year: 1972,
			overview: `${'Una sinopsis larga '.repeat(20)} final`,
			poster_url: 'https://images.example/candidate.jpg',
			popularity: 90,
			vote_count: 1000
		};
		const body = renderPage(makeItem(), {
			isAdmin: true,
			form: {
				panel: true,
				candidates: [candidate],
				search: { q: 'El Padrino', year: 1972 }
			}
		});

		expect(body).toContain('https://images.example/candidate.jpg');
		expect(body).toContain('The Godfather');
		expect(body).toContain('Es esta');
		expect(body).toContain('name="tmdb_id" value="238"');
		expect(body).toContain('…');

		const empty = renderPage(makeItem(), {
			isAdmin: true,
			form: { candidates: [], search: { q: 'No existe', year: null } }
		});
		expect(empty).toContain('Sin candidatos: prueba otro título o año');
	});
});

describe('acciones de la ficha', () => {
	it('marca como vista con PUT', async () => {
		const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));

		const result = (await actions.watched!(actionEvent(fetchMock, 'watched', {
			watched: '1'
		}) as never)) as { success?: boolean; message?: string };

		expect(result.success).toBe(true);
		expect(result.message).toContain('Marcada');
		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe('http://api.test/api/v1/items/7/watched');
		expect(init.method).toBe('PUT');
		expect(init.headers.Authorization).toBe('Bearer token-123');
	});

	it('desmarca con DELETE', async () => {
		const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));

		const result = (await actions.watched!(actionEvent(fetchMock, 'watched', {
			watched: '0'
		}) as never)) as { success?: boolean; message?: string };

		expect(result.success).toBe(true);
		expect(result.message).toContain('Desmarcada');
		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe('http://api.test/api/v1/items/7/watched');
		expect(init.method).toBe('DELETE');
	});

	it('fija una coincidencia con PUT', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse(makeItem()));

		const result = (await actions.match!(actionEvent(fetchMock, 'match', {
			tmdb_id: '238'
		}) as never)) as { success?: boolean; message?: string };

		expect(result.success).toBe(true);
		expect(result.message).toContain('corregida');
		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe('http://api.test/api/v1/items/7/match');
		expect(init.method).toBe('PUT');
		expect(JSON.parse(init.body)).toEqual({ tmdb_id: 238 });
	});

	it('marca «no es ninguna» con tmdb_id null', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse(makeItem()));

		const result = (await actions.nomatch!(actionEvent(fetchMock, 'nomatch') as never)) as {
			success?: boolean;
		};

		expect(result.success).toBe(true);
		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe('http://api.test/api/v1/items/7/match');
		expect(init.method).toBe('PUT');
		expect(JSON.parse(init.body)).toEqual({ tmdb_id: null });
	});

	it('vuelve a automático con DELETE', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse(makeItem()));

		const result = (await actions.automatic!(actionEvent(fetchMock, 'automatic') as never)) as {
			success?: boolean;
		};

		expect(result.success).toBe(true);
		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe('http://api.test/api/v1/items/7/match');
		expect(init.method).toBe('DELETE');
	});

	it('busca candidatos con el título y el año', async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			jsonResponse({ candidates: [{ tmdb_id: 238, title: 'El Padrino' }] })
		);

		const result = (await actions.candidates!(
			actionEvent(fetchMock, 'candidates', { q: 'El Padrino', year: '1972' }) as never
		)) as {
			candidates: Candidate[];
			search: { q: string; year: number | null };
		};

		expect(result.candidates).toHaveLength(1);
		expect(result.search).toEqual({ q: 'El Padrino', year: 1972 });
		const [url] = fetchMock.mock.calls[0];
		const parsed = new URL(String(url));
		expect(parsed.pathname).toBe('/api/v1/items/7/candidates');
		expect(parsed.searchParams.get('q')).toBe('El Padrino');
		expect(parsed.searchParams.get('year')).toBe('1972');
	});

	it('traduce los errores de permisos y de TMDB', async () => {
		const forbidden = vi.fn().mockResolvedValue(jsonResponse({ error: 'no admin' }, 403));
		const forbiddenResult = (await actions.match!(
			actionEvent(forbidden, 'match', { tmdb_id: '238' }) as never
		)) as { status: number; data: { message: string } };
		expect(forbiddenResult.status).toBe(403);
		expect(forbiddenResult.data.message).toContain('administradores');

		const unavailable = vi.fn().mockResolvedValue(jsonResponse({ error: 'sin TMDB' }, 503));
		const unavailableResult = (await actions.candidates!(
			actionEvent(unavailable, 'candidates', { q: 'El Padrino' }) as never
		)) as { status: number; data: { message: string } };
		expect(unavailableResult.status).toBe(503);
		expect(unavailableResult.data.message).toBe('TMDB no está configurado en el servidor');

		const broken = vi.fn().mockResolvedValue(jsonResponse({ error: 'fallo' }, 502));
		const brokenResult = (await actions.candidates!(
			actionEvent(broken, 'candidates', { q: 'El Padrino' }) as never
		)) as { status: number; data: { message: string } };
		expect(brokenResult.status).toBe(502);
		expect(brokenResult.data.message).toBe('No se pudo consultar TMDB');
	});
});
