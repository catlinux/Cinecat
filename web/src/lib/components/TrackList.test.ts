import { render } from 'svelte/server';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import type { MusicAlbum, MusicTrack } from '$lib/api';
import { musicPlayer, resetMusicPlayer } from '$lib/music-player.svelte';
import TrackList from './TrackList.svelte';

const album: MusicAlbum = {
	id: 7,
	title: 'Álbum',
	artist: 'Artista',
	artist_id: 3,
	year: 2020,
	track_count: 3,
	duration_seconds: 400,
	has_cover: false
};

const tracks: MusicTrack[] = [
	{
		id: 1,
		title: 'Primera',
		track_no: 1,
		disc_no: 1,
		artist: 'Artista',
		duration_seconds: 61,
		codec: 'mp3',
		bitrate: 320000,
		playable: true
	},
	{
		id: 2,
		title: 'Segunda',
		track_no: 2,
		disc_no: 1,
		artist: 'Artista',
		duration_seconds: 62,
		codec: 'mp3',
		bitrate: 320000,
		playable: false
	},
	{
		id: 3,
		title: 'Tercera',
		track_no: 1,
		disc_no: 2,
		artist: 'Artista',
		duration_seconds: 63,
		codec: 'flac',
		bitrate: 900000,
		playable: true
	}
];

describe('TrackList', () => {
	beforeEach(() => resetMusicPlayer());
	afterEach(() => resetMusicPlayer());

	it('muestra discos, duración, acciones y pista actual', () => {
		musicPlayer.tracks = [
			{
				id: 1,
				title: 'Primera',
				artist: 'Artista',
				album: 'Álbum',
				albumId: 7,
				durationSeconds: 61
			}
		];
		musicPlayer.index = 0;
		const { body } = render(TrackList, { props: { tracks, album } });

		expect(body).toContain('Disco 1');
		expect(body).toContain('Disco 2');
		expect(body).toContain('Primera');
		expect(body).toContain('1:01');
		expect(body).toContain('Reproducir Primera');
		expect(body).toContain('Reproducir a continuación Primera');
		expect(body).toContain('Añadir Primera a la cola');
		expect(body).toContain('aria-current="true"');
		expect(body).toContain('Sonando ahora');
	});

	it('deshabilita las pistas no reproducibles con el motivo', () => {
		const { body } = render(TrackList, { props: { tracks, album } });

		expect(body).toContain('Reproducir Segunda');
		expect(body).toContain('disabled');
		expect(body).toContain('Formato no compatible con el navegador');
	});

	it('muestra álbum y artista en resultados de búsqueda y escapa texto', () => {
		const search: MusicTrack[] = [
			{
				...tracks[0],
				title: '<b>Título</b>',
				album: 'Mi álbum',
				album_id: 9
			}
		];
		const { body } = render(TrackList, { props: { tracks: search, showAlbum: true } });

		expect(body).toContain('&lt;b>Título&lt;/b>');
		expect(body).not.toContain('<b>Título</b>');
		expect(body).toContain('Mi álbum');
		expect(body).toContain('Artista');
	});
});
