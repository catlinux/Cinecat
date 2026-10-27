import { describe, expect, it } from 'vitest';
import { detectEpisode } from './episode-detect';

describe('detección de temporada y episodio', () => {
	it('reconoce S01E05, s1e5 y sus variantes con separadores', () => {
		expect(detectEpisode('Serie S01E05.mkv')).toEqual({ season: 1, episode: 5 });
		expect(detectEpisode('Serie s1e5.mkv')).toEqual({ season: 1, episode: 5 });
		expect(detectEpisode('ShowS01E005.avi')).toEqual({ season: 1, episode: 5 });
		expect(detectEpisode('Serie S01.E05.mkv')).toEqual({ season: 1, episode: 5 });
		expect(detectEpisode('Serie S01 E05.mkv')).toEqual({ season: 1, episode: 5 });
		expect(detectEpisode('Serie S00E12.mkv')).toEqual({ season: 0, episode: 12 });
	});

	it('reconoce 1x05 y variantes', () => {
		expect(detectEpisode('Serie 1x05.mkv')).toEqual({ season: 1, episode: 5 });
		expect(detectEpisode('Show1x05.mkv')).toEqual({ season: 1, episode: 5 });
		expect(detectEpisode('Serie 10X123.mkv')).toEqual({ season: 10, episode: 123 });
		expect(detectEpisode('1920x1080.mkv')).toEqual({ season: null, episode: null });
	});

	it('reconoce Temporada y Episodio escritos', () => {
		expect(detectEpisode('Serie Temporada 1 Episodio 5.mkv')).toEqual({
			season: 1,
			episode: 5
		});
		expect(detectEpisode('Serie temporada 2 capítulo 12.mkv')).toEqual({
			season: 2,
			episode: 12
		});
		expect(detectEpisode('Serie Temporada 3 Ep. 7.mkv')).toEqual({ season: 3, episode: 7 });
		expect(detectEpisode('Serie Temporada 3 - Episodio 7.mkv')).toEqual({
			season: 3,
			episode: 7
		});
		expect(detectEpisode('Serie Episodio - 7.mkv', 2)).toEqual({ season: 2, episode: 7 });
	});

	it('reconoce solo el episodio y aplica la temporada por defecto', () => {
		expect(detectEpisode('Serie Capitulo 5.mkv', 4)).toEqual({ season: 4, episode: 5 });
		expect(detectEpisode('Serie Episodio 12.mkv', 2)).toEqual({ season: 2, episode: 12 });
		expect(detectEpisode('Serie Capítulo 3.mkv')).toEqual({ season: null, episode: 3 });
		expect(detectEpisode('Serie Temporada 2 Episodio 5.mkv', 9)).toEqual({
			season: 2,
			episode: 5
		});
	});

	it('reconoce un número suelto al final o tras un separador', () => {
		expect(detectEpisode('Serie - 05.mkv', 1)).toEqual({ season: 1, episode: 5 });
		expect(detectEpisode('Serie_05.mkv', 1)).toEqual({ season: 1, episode: 5 });
		expect(detectEpisode('Serie 05 1080p.mkv', 1)).toEqual({ season: 1, episode: 5 });
		expect(detectEpisode('Serie 2024 07.mkv', 3)).toEqual({ season: 3, episode: 7 });
		expect(detectEpisode('Serie 720p.mkv')).toEqual({ season: null, episode: null });
		expect(detectEpisode('1917.mkv')).toEqual({ season: null, episode: null });
	});

	it('no inventa datos si no hay episodio', () => {
		expect(detectEpisode('Sin número.mkv')).toEqual({ season: null, episode: null });
		expect(detectEpisode('Serie S01E00.mkv')).toEqual({ season: 1, episode: 0 });
	});
});
