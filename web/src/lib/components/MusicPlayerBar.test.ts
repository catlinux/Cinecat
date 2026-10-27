import { render } from 'svelte/server';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { musicPlayer, resetMusicPlayer } from '$lib/music-player.svelte';
import MusicPlayerBar from './MusicPlayerBar.svelte';

describe('MusicPlayerBar', () => {
	beforeEach(() => resetMusicPlayer());
	afterEach(() => resetMusicPlayer());

	it('deja un único audio y no muestra la barra con la cola vacía', () => {
		const { body } = render(MusicPlayerBar);

		expect((body.match(/<audio/g) ?? []).length).toBe(1);
		expect(body).not.toContain('music-player-bar');
	});

	it('muestra la pista, enlaces, controles, progreso y cola', () => {
		musicPlayer.tracks = [
			{
				id: 4,
				title: 'Come Together',
				artist: 'The Beatles',
				artistId: 8,
				album: 'Abbey Road',
				albumId: 7,
				durationSeconds: 259
			}
		];
		musicPlayer.index = 0;
		musicPlayer.duration = 259;
		musicPlayer.currentTime = 12;
		const { body } = render(MusicPlayerBar);

		expect(body).toContain('music-player-bar');
		expect(body).toContain('Come Together');
		expect(body).toContain('href="/music/artists/8"');
		expect(body).toContain('href="/music/albums/7"');
		expect(body).toContain('aria-label="Reproducir"');
		expect(body).toContain('aria-label="Pista anterior"');
		expect(body).toContain('aria-label="Pista siguiente"');
		expect(body).toContain('aria-label="Progreso de la pista"');
		expect(body).toContain('role="slider"');
		expect(body).toContain('aria-valuetext="0:12 de 4:19"');
		expect(body).toContain('Cola (1)');
		expect(body).toContain('Vaciar cola');
		expect(body).toContain('aria-label="Quitar Come Together de la cola"');
	});

	it('escapa el título y el artista', () => {
		musicPlayer.tracks = [
			{
				id: 1,
				title: '<script>x</script>',
				artist: 'A & B',
				album: '',
				albumId: 0,
				durationSeconds: 10
			}
		];
		musicPlayer.index = 0;
		const { body } = render(MusicPlayerBar);

		expect(body).not.toContain('<script>x</script>');
		expect(body).toContain('&lt;script');
		expect(body).toContain('A &amp; B');
	});
});
