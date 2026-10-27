import { describe, expect, it } from 'vitest';
import type { MusicAlbum, MusicTrack } from '$lib/api';
import {
	albumInitials,
	albumMeta,
	albumQueueTracks,
	formatTotalDuration,
	formatTrackDuration,
	musicApiQuery,
	musicCountLabel,
	musicHref,
	musicLetterHref,
	musicTabHref,
	parseMusicLetter,
	parseMusicOrder,
	parseMusicPage,
	parseMusicView,
	readMusicFilters,
	toQueueTrack
} from './music';
import type { MusicQueueTrack } from './music-queue';

const album: MusicAlbum = {
	id: 7,
	title: 'Álbum de prueba',
	artist: 'Artista',
	artist_id: 3,
	year: 2020,
	track_count: 2,
	duration_seconds: 300,
	has_cover: true
};

const track: MusicTrack = {
	id: 11,
	title: 'Pista',
	track_no: 1,
	disc_no: 1,
	artist: 'Artista',
	duration_seconds: 120,
	codec: 'mp3',
	bitrate: 320000,
	playable: true
};

describe('parámetros de música', () => {
	it('valida vistas, órdenes, letras y páginas con valores por defecto', () => {
		expect(parseMusicView('artistas')).toBe('artistas');
		expect(parseMusicView('OTRA')).toBe('albumes');
		expect(parseMusicOrder('year')).toBe('year');
		expect(parseMusicOrder('raro')).toBe('title');
		expect(parseMusicLetter('ó')).toBe('Ó');
		expect(parseMusicLetter('#')).toBe('#');
		expect(parseMusicLetter('ab')).toBeNull();
		expect(parseMusicLetter(' ')).toBeNull();
		expect(parseMusicPage('3')).toBe(3);
		expect(parseMusicPage('0')).toBe(1);
		expect(parseMusicPage('x')).toBe(1);
	});

	it('lee los filtros de la URL y construye la consulta de la API', () => {
		const filters = readMusicFilters(
			new URL('http://localhost/music?vista=artistas&q=beat&letra=b&orden=year&page=2')
		);
		expect(filters).toEqual({
			view: 'artistas',
			q: 'beat',
			letter: 'B',
			order: 'year',
			page: 2
		});
		expect(musicApiQuery(filters, 2, 10).toString()).toBe('limit=10&offset=10&q=beat&letter=B');
		const albums = readMusicFilters(
			new URL('http://localhost/music?vista=albumes&q=road&orden=artist')
		);
		expect(musicApiQuery(albums, 1, 20).toString()).toBe(
			'limit=20&offset=0&q=road&sort=artist'
		);
	});

	it('construye enlaces conservando la vista y reiniciando la página', () => {
		const current = '/music?vista=albumes&page=3&orden=year';
		expect(musicHref(current, { q: 'road' })).toBe('/music?vista=albumes&orden=year&q=road');
		expect(musicTabHref(current, 'artistas')).toBe('/music?vista=artistas&orden=year');
		expect(musicTabHref(current, 'albumes')).toBe('/music?orden=year');
		expect(musicLetterHref(current, 'B', false)).toBe('/music?vista=albumes&orden=year&letra=B');
		expect(musicLetterHref('/music?letra=B', 'B', true)).toBe('/music');
	});
});

describe('textos y colas de música', () => {
	it('formatea duraciones y metadatos', () => {
		expect(formatTrackDuration(61)).toBe('1:01');
		expect(formatTrackDuration(null)).toBe('—');
		expect(formatTotalDuration(65)).toBe('1 min');
		expect(formatTotalDuration(3599)).toBe('1 h');
		expect(formatTotalDuration(3660)).toBe('1 h 1 min');
		expect(formatTotalDuration(0)).toBe('');
		expect(albumMeta(album)).toBe('2020 · 2 pistas');
		expect(albumInitials('Abbey Road')).toBe('AR');
		expect(albumInitials(' ')).toBe('♪');
		expect(musicCountLabel('albumes', 1)).toBe('1 álbum');
		expect(musicCountLabel('pistas', 2)).toBe('2 pistas');
		expect(musicCountLabel('artistas', 0)).toBe('0 artistas');
	});

	it('convierte pistas y álbumes a la cola del reproductor', () => {
		expect(toQueueTrack(track, album)).toEqual({
			id: 11,
			title: 'Pista',
			artist: 'Artista',
			artistId: 3,
			album: 'Álbum de prueba',
			albumId: 7,
			durationSeconds: 120
		} satisfies MusicQueueTrack);
		expect(albumQueueTracks({ ...album, tracks: [track] })).toHaveLength(1);
		expect(
			toQueueTrack({ ...track, album: 'Otro', album_id: 9 })
		).toMatchObject({ album: 'Otro', albumId: 9 });
	});
});
