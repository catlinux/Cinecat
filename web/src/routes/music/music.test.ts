import { describe, expect, it, vi } from 'vitest';
import type { MusicAlbum, MusicAlbumDetail, MusicArtist, MusicArtistDetail, MusicTrack } from '$lib/api';
import { DEFAULT_PUBLIC_UI } from '$lib/settings';
import { load as loadMusic } from './+page.server';
import { load as loadAlbum } from './albums/[id]/+page.server';
import { load as loadArtist } from './artists/[id]/+page.server';
import { GET as getCover } from './cover/[id]/+server';
import { GET as getPlay } from './play/[id]/+server';

function jsonResponse(body: unknown, status = 200): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

const album: MusicAlbum = {
	id: 7,
	title: 'Abbey Road',
	artist: 'The Beatles',
	artist_id: 3,
	year: 1969,
	track_count: 1,
	duration_seconds: 259,
	has_cover: true
};

const artist: MusicArtist = {
	id: 3,
	name: 'The Beatles',
	album_count: 1,
	track_count: 1
};

const track: MusicTrack = {
	id: 11,
	title: 'Come Together',
	track_no: 1,
	disc_no: 1,
	artist: 'The Beatles',
	duration_seconds: 259,
	codec: 'mp3',
	bitrate: 320000,
	playable: true
};

const albumDetail: MusicAlbumDetail = { ...album, tracks: [track] };
const artistDetail: MusicArtistDetail = { ...artist, albums: [album] };

function musicLoadEvent(fetchImpl: typeof fetch, url = 'http://localhost/music') {
	return {
		locals: { sessionToken: 'token-123' },
		fetch: fetchImpl,
		url: new URL(url),
		parent: async () => ({ ui: DEFAULT_PUBLIC_UI }),
		params: {},
		request: new Request(url),
		route: { id: '/music' },
		cookies: {},
		setHeaders: vi.fn(),
		getClientAddress: () => '127.0.0.1',
		isDataRequest: false,
		isSubRequest: false,
		platform: undefined
	};
}

function detailEvent(
	fetchImpl: typeof fetch,
	id: string,
	route: string
) {
	return {
		locals: { sessionToken: 'token-123' },
		fetch: fetchImpl,
		url: new URL(`http://localhost${route}`),
		params: { id },
		parent: async () => ({ ui: DEFAULT_PUBLIC_UI }),
		request: new Request(`http://localhost${route}`),
		route: { id: `${route.split('/').slice(0, -1).join('/')}/[id]` },
		cookies: {},
		setHeaders: vi.fn(),
		getClientAddress: () => '127.0.0.1',
		isDataRequest: false,
		isSubRequest: false,
		platform: undefined
	};
}

function endpointEvent(fetchImpl: typeof fetch, path: string, token: string | null = 'token-123', ifNoneMatch = '') {
	return {
		params: { id: path.split('/').pop() ?? '' },
		locals: { sessionToken: token },
		fetch: fetchImpl,
		url: new URL(`http://localhost${path}`),
		request: new Request(`http://localhost${path}`, {
			headers: ifNoneMatch ? { 'If-None-Match': ifNoneMatch } : {}
		})
	};
}

function musicFetch(overrides: Record<string, () => Response> = {}) {
	return vi.fn(async (input: RequestInfo | URL) => {
		const url = String(input);
		for (const [suffix, response] of Object.entries(overrides)) {
			if (url.includes(suffix)) {
				return response();
			}
		}
		if (url.includes('/api/v1/music/albums/7')) {
			return jsonResponse(albumDetail);
		}
		if (url.includes('/api/v1/music/albums')) {
			return jsonResponse({ albums: [album], total: 1, limit: 50, offset: 0 });
		}
		if (url.includes('/api/v1/music/artists/3')) {
			return jsonResponse(artistDetail);
		}
		if (url.includes('/api/v1/music/artists')) {
			return jsonResponse({ artists: [artist], total: 1, limit: 50, offset: 0 });
		}
		if (url.includes('/api/v1/music/tracks')) {
			return jsonResponse({ tracks: [track], total: 1, limit: 50, offset: 0 });
		}
		throw new Error(`URL inesperada: ${url}`);
	});
}

describe('load de /music', () => {
	it('carga álbumes y valida los parámetros', async () => {
		const fetchMock = musicFetch();
		const result = (await loadMusic(
			musicLoadEvent(fetchMock, 'http://localhost/music?vista=albumes&orden=year&letra=b&page=1') as never
		)) as any;

		expect(result).toMatchObject({
			view: 'albumes',
			order: 'year',
			letter: 'B',
			total: 1,
			page: 1,
			last: 1,
			errorMessage: null
		});
		expect(result.albums).toHaveLength(1);
		expect(String(fetchMock.mock.calls[0][0])).toContain('/api/v1/music/albums?');
	});

	it('carga artistas y pistas con q', async () => {
		const artists = (await loadMusic(
			musicLoadEvent(musicFetch(), 'http://localhost/music?vista=artistas&q=beat') as never
		)) as any;
		expect(artists.artists[0].name).toBe('The Beatles');
		expect(artists.view).toBe('artistas');

		const tracks = (await loadMusic(
			musicLoadEvent(musicFetch(), 'http://localhost/music?vista=pistas&q=come') as never
		)) as any;
		expect(tracks.tracks[0].title).toBe('Come Together');
		expect(tracks.view).toBe('pistas');
	});

	it('no pide pistas sin búsqueda', async () => {
		const fetchMock = musicFetch();
		const result = (await loadMusic(
			musicLoadEvent(fetchMock, 'http://localhost/music?vista=pistas') as never
		)) as any;
		expect(result.total).toBe(0);
		expect(fetchMock).not.toHaveBeenCalled();
	});

	it('redirige al login con 401 y muestra un error genérico con 502', async () => {
		const unauthorized = vi.fn(async () => jsonResponse({ error: 'sesión' }, 401));
		await expect(
			loadMusic(musicLoadEvent(unauthorized as never) as never)
		).rejects.toMatchObject({ status: 303, location: '/login' });

		const broken = vi.fn(async () => jsonResponse({ error: 'fallo' }, 500));
		const result = (await loadMusic(musicLoadEvent(broken as never) as never)) as any;
		expect(result.errorMessage).toBeTruthy();
		expect(result.albums).toEqual([]);
	});
});

describe('load de álbum y artista', () => {
	it('carga el detalle del álbum', async () => {
		const result = (await loadAlbum(detailEvent(musicFetch(), '7', '/music/albums/7') as never)) as any;
		expect(result.album.id).toBe(7);
		expect(result.album.tracks).toHaveLength(1);
	});

	it('devuelve 404 para un id no válido o inexistente', async () => {
		await expect(
			loadAlbum(detailEvent(musicFetch(), '0', '/music/albums/0') as never)
		).rejects.toMatchObject({ status: 404 });

		const notFound = vi.fn(async () => jsonResponse({ error: 'no' }, 404));
		await expect(
			loadAlbum(detailEvent(notFound as never, '7', '/music/albums/7') as never)
		).rejects.toMatchObject({ status: 404 });
	});

	it('traduce 401 y 502 al cargar álbum y artista', async () => {
		await expect(
			loadAlbum(
				detailEvent(
					vi.fn(async () => jsonResponse({ error: 'sesión' }, 401)) as never,
					'7',
					'/music/albums/7'
				) as never
			)
		).rejects.toMatchObject({ status: 303, location: '/login' });

		await expect(
			loadAlbum(
				detailEvent(
					vi.fn(async () => jsonResponse({ error: 'fallo' }, 500)) as never,
					'7',
					'/music/albums/7'
				) as never
			)
		).rejects.toMatchObject({ status: 502 });

		await expect(
			loadArtist(detailEvent(musicFetch(), '0', '/music/artists/0') as never)
		).rejects.toMatchObject({ status: 404 });

		await expect(
			loadArtist(
				detailEvent(
					vi.fn(async () => jsonResponse({ error: 'fallo' }, 500)) as never,
					'3',
					'/music/artists/3'
				) as never
			)
		).rejects.toMatchObject({ status: 502 });
	});

	it('carga el artista y las pistas de sus álbumes para reproducir todo', async () => {
		const result = (await loadArtist(detailEvent(musicFetch(), '3', '/music/artists/3') as never)) as any;
		expect(result.artist.name).toBe('The Beatles');
		expect(result.queueTracks).toHaveLength(1);
		expect(result.queueTracks[0]).toMatchObject({ id: 11, albumId: 7, artistId: 3 });
	});

	it('tolera un álbum roto al cargar el artista', async () => {
		const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
			const url = String(input);
			if (url.includes('/api/v1/music/artists/3')) {
				return jsonResponse(artistDetail);
			}
			return jsonResponse({ error: 'roto' }, 500);
		});
		const result = (await loadArtist(detailEvent(fetchMock as never, '3', '/music/artists/3') as never)) as any;
		expect(result.queueTracks).toEqual([]);
	});
});

describe('endpoints de portada y reproducción', () => {
	it('sirve la portada con caché y reenvía el ETag', async () => {
		const fetchMock = vi.fn(async (_input: RequestInfo | URL, _init?: RequestInit) =>
			new Response(new Uint8Array([1, 2, 3]), {
				status: 200,
				headers: { 'Content-Type': 'image/jpeg', ETag: '"abc"' }
			})
		);
		const response = await getCover(endpointEvent(fetchMock as never, '/music/cover/7') as never);
		expect(response.status).toBe(200);
		expect(response.headers.get('Cache-Control')).toBe('private, max-age=86400');
		expect(response.headers.get('ETag')).toBe('"abc"');
		expect((await response.arrayBuffer()).byteLength).toBe(3);
		expect((fetchMock.mock.calls[0][1]?.headers as Record<string, string>).Authorization).toBe(
			'Bearer token-123'
		);
	});

	it('devuelve 304 y un PNG transparente ante 404 o error', async () => {
		const notModified = vi.fn(async () =>
			new Response(null, { status: 304, headers: { ETag: '"abc"' } })
		);
		const response = await getCover(
			endpointEvent(notModified as never, '/music/cover/7', 'token-123', '"abc"') as never
		);
		expect(response.status).toBe(304);

		const missing = vi.fn(async () => new Response(null, { status: 404 }));
		const fallback = await getCover(endpointEvent(missing as never, '/music/cover/7') as never);
		expect(fallback.status).toBe(200);
		expect(fallback.headers.get('Content-Type')).toBe('image/png');
		expect(fallback.headers.get('Cache-Control')).toBe('private, max-age=60');
		expect((await fallback.arrayBuffer()).byteLength).toBeGreaterThan(0);

		const unauthorized = await getCover(
			endpointEvent(vi.fn() as never, '/music/cover/7', null) as never
		);
		expect(unauthorized.status).toBe(401);
	});

	it('reenvía la reproducción con no-store', async () => {
		const fetchMock = vi.fn(async (_input: RequestInfo | URL, _init?: RequestInit) =>
			jsonResponse({
				url: '/api/v1/stream/music/11?exp=1&sig=x',
				mime: 'audio/mpeg',
				expires_at: '2026-10-27T12:00:00Z',
				playable: true
			})
		);
		const response = await getPlay(endpointEvent(fetchMock as never, '/music/play/11') as never);
		expect(response.status).toBe(200);
		expect(response.headers.get('Cache-Control')).toBe('no-store');
		expect(await response.json()).toMatchObject({ mime: 'audio/mpeg', playable: true });
		expect((fetchMock.mock.calls[0][1]?.headers as Record<string, string>).Authorization).toBe(
			'Bearer token-123'
		);
	});

	it('traduce 401, 403, 404 y 502 de la reproducción', async () => {
		for (const [status, expected] of [
			[401, 401],
			[403, 403],
			[404, 404],
			[500, 502]
		] as const) {
			const response = await getPlay(
				endpointEvent(
					vi.fn(async () => jsonResponse({ error: 'fallo' }, status)) as never,
					'/music/play/11'
				) as never
			);
			expect(response.status).toBe(expected);
			expect(response.headers.get('Cache-Control')).toBe('no-store');
			expect((await response.json()).error).toBeTruthy();
		}
	});

	it('sin sesión devuelve 401 y un id inválido 404', async () => {
		const noSession = await getPlay(
			endpointEvent(vi.fn() as never, '/music/play/11', null) as never
		);
		expect(noSession.status).toBe(401);
		const badID = await getPlay(endpointEvent(vi.fn() as never, '/music/play/0') as never);
		expect(badID.status).toBe(404);
	});
});
