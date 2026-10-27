import { describe, expect, it } from 'vitest';
import type { MediaItem } from './api';
import {
	episodeCardSubtitle,
	episodeImageURL,
	episodeNeighbors,
	formatAudioChannels,
	formatBitrate,
	formatBytes,
	formatDateTime,
	formatDurationSeconds,
	formatFps,
	formatGenres,
	formatLanguage,
	formatMovieMeta,
	formatRating,
	formatResolution,
	formatScore,
	formatRuntime,
	formatSampleRate,
	matchStatusLabel,
	overviewInSpanish,
	splitCast,
	truncateSynopsis
} from './media';

function episodeItem(
	id: number,
	season: number,
	episode: number,
	title = `Episodio ${episode}`
): MediaItem {
	return {
		id,
		library_id: 1,
		path: `/media/series/${id}.mkv`,
		title,
		file_title: `file-${id}`,
		year: 2010,
		kind: 'episode',
		season,
		episode,
		tmdb_id: null,
		overview: null,
		poster_url: null,
		poster_custom: false,
		original_title: null,
		imdb_id: null,
		tagline: null,
		genres: [],
		runtime_minutes: 42,
		rating: null,
		directors: [],
		cast: [],
		backdrop_url: null,
		backdrop_custom: false,
		overview_language: null,
		match_status: null,
		match_score: null,
		watched: false,
		progress: null,
		collections: [],
		series_id: 5,
		series_title: 'Una serie',
		episode_title: title,
		still_url: null
	};
}

describe('formato de la ficha', () => {
	it('formatea la duración en horas y minutos', () => {
		expect(formatRuntime(112)).toBe('1 h 52 min');
		expect(formatRuntime(60)).toBe('1 h');
		expect(formatRuntime(45)).toBe('45 min');
		expect(formatRuntime(0)).toBe('');
		expect(formatRuntime(null)).toBe('');
	});

	it('formatea la nota con una cifra decimal y coma', () => {
		expect(formatRating(7.8)).toBe('★ 7,8');
		expect(formatRating(10)).toBe('★ 10,0');
		expect(formatRating(null)).toBe('');
	});

	it('formatea la puntuación de la carátula sin estrella', () => {
		expect(formatScore(7.4)).toBe('7,4');
		expect(formatScore(10)).toBe('10,0');
		expect(formatScore(0)).toBe('');
		expect(formatScore(-1)).toBe('');
		expect(formatScore(null)).toBe('');
	});

	it('une los géneros sin entradas vacías', () => {
		expect(formatGenres(['Drama', ' Crimen ', ''])).toBe('Drama, Crimen');
		expect(formatGenres([])).toBe('');
	});

	it('construye la línea de datos de la película', () => {
		expect(
			formatMovieMeta({
				year: 1999,
				runtime_minutes: 112,
				genres: ['Drama', 'Crimen'],
				rating: 7.8
			})
		).toBe('1999 · 1 h 52 min · Drama, Crimen · ★ 7,8');
	});

	it('pliega el reparto a partir del límite', () => {
		const cast = Array.from({ length: 13 }, (_, index) => `Actor ${index + 1}`);
		const split = splitCast(cast);

		expect(split.visible).toHaveLength(12);
		expect(split.hidden).toEqual(['Actor 13']);
	});
});

describe('estado de identificación y sinopsis', () => {
	it('reconoce las sinopsis en castellano', () => {
		expect(overviewInSpanish('es-ES')).toBe(true);
		expect(overviewInSpanish('ES')).toBe(true);
		expect(overviewInSpanish('en-US')).toBe(false);
		expect(overviewInSpanish(null)).toBe(false);
	});

	it('describe el estado de la identificación', () => {
		expect(matchStatusLabel({ match_status: 'auto', match_score: 0.92 })).toBe(
			'Identificada automáticamente (92 %)'
		);
		expect(matchStatusLabel({ match_status: 'auto', match_score: null })).toBe(
			'Identificada automáticamente'
		);
		expect(matchStatusLabel({ match_status: 'manual', match_score: 1 })).toBe(
			'Corregida a mano'
		);
		expect(matchStatusLabel({ match_status: 'none', match_score: null })).toBe(
			'Marcada como «no es ninguna»'
		);
		expect(matchStatusLabel({ match_status: null, match_score: null })).toBe('Sin identificar');
	});

	it('recorta la sinopsis del candidato', () => {
		const text = `  ${'Una sinopsis muy larga '.repeat(20)}  `;
		const result = truncateSynopsis(text);

		expect(result.length).toBeLessThanOrEqual(140);
		expect(result.endsWith('…')).toBe(true);
		expect(result).not.toContain('  ');
	});
});

describe('navegación entre episodios', () => {
	const season1 = [episodeItem(11, 1, 1), episodeItem(12, 1, 2)];
	const season2 = [episodeItem(21, 2, 1), episodeItem(22, 2, 2)];
	const seasons = [{ season: 1 }, { season: 2 }];

	it('encuentra el anterior y el siguiente dentro de la temporada', () => {
		const middle = episodeItem(13, 1, 2);
		const result = episodeNeighbors(
			middle,
			seasons,
			new Map([
				[1, [episodeItem(11, 1, 1), middle, episodeItem(14, 1, 3)]]
			])
		);

		expect(result.previous?.id).toBe(11);
		expect(result.next?.id).toBe(14);
	});

	it('salta a la temporada anterior en el primer episodio', () => {
		const first = episodeItem(21, 2, 1);
		const result = episodeNeighbors(first, seasons, new Map([
			[1, season1],
			[2, season2]
		]));

		expect(result.previous?.id).toBe(12);
		expect(result.next?.id).toBe(22);
	});

	it('salta a la temporada siguiente en el último episodio', () => {
		const last = episodeItem(12, 1, 2);
		const result = episodeNeighbors(last, seasons, new Map([
			[1, season1],
			[2, season2]
		]));

		expect(result.previous?.id).toBe(11);
		expect(result.next?.id).toBe(21);
	});

	it('no cruza los extremos de la serie', () => {
		const first = episodeNeighbors(
			episodeItem(11, 1, 1),
			seasons,
			new Map([[1, season1]])
		);
		const last = episodeNeighbors(
			episodeItem(22, 2, 2),
			seasons,
			new Map([[2, season2]])
		);

		expect(first.previous).toBeNull();
		expect(first.next?.id).toBe(12);
		expect(last.previous?.id).toBe(21);
		expect(last.next).toBeNull();
	});

	it('respeta los huecos entre temporadas', () => {
		const season3 = [episodeItem(31, 3, 1), episodeItem(32, 3, 2)];
		const firstOfSeason3 = episodeNeighbors(
			episodeItem(31, 3, 1),
			[{ season: 1 }, { season: 3 }],
			new Map([
				[1, season1],
				[3, season3]
			])
		);

		expect(firstOfSeason3.previous?.id).toBe(12);
		expect(firstOfSeason3.next?.id).toBe(32);
	});

	it('no inventa vecinos si la temporada solo tiene un episodio', () => {
		const only = episodeItem(41, 1, 1);
		const result = episodeNeighbors(only, [{ season: 1 }], new Map([[1, [only]]]));

		expect(result.previous).toBeNull();
		expect(result.next).toBeNull();
	});
});

describe('formatos del panel de archivo', () => {
	it('formatea tamaños en bytes', () => {
		expect(formatBytes(512)).toBe('512 B');
		expect(formatBytes(734003200)).toBe('700 MB');
		expect(formatBytes(1500000000)).toBe('1,4 GB');
		expect(formatBytes(null)).toBe('—');
	});

	it('formatea tasas de bits', () => {
		expect(formatBitrate(5000000)).toBe('5 Mb/s');
		expect(formatBitrate(1371742)).toBe('1,4 Mb/s');
		expect(formatBitrate(448000)).toBe('448 kb/s');
		expect(formatBitrate(800000, true)).toBe('0,8 Mb/s');
		expect(formatBitrate(800)).toBe('800 b/s');
		expect(formatBitrate(0)).toBe('—');
	});

	it('describe canales y disposición', () => {
		expect(formatAudioChannels(2, 'stereo')).toBe('estéreo');
		expect(formatAudioChannels(1, 'mono')).toBe('mono');
		expect(formatAudioChannels(6, '5.1(side)')).toBe('5.1');
		expect(formatAudioChannels(8, '7.1')).toBe('7.1');
		expect(formatAudioChannels(3, '')).toBe('3 canales');
	});

	it('traduce los idiomas conocidos y conserva los desconocidos', () => {
		expect(formatLanguage('es')).toBe('Español');
		expect(formatLanguage('spa')).toBe('Español');
		expect(formatLanguage('en')).toBe('Inglés');
		expect(formatLanguage('fr')).toBe('Francés');
		expect(formatLanguage('de')).toBe('Alemán');
		expect(formatLanguage('it')).toBe('Italiano');
		expect(formatLanguage('pt')).toBe('Portugués');
		expect(formatLanguage('ca')).toBe('Catalán');
		expect(formatLanguage('ja')).toBe('Japonés');
		expect(formatLanguage('xx')).toBe('xx');
		expect(formatLanguage(null)).toBe('');
	});

	it('formatea fps, resolución, frecuencia y duración', () => {
		expect(formatFps(23.976)).toBe('23,976');
		expect(formatFps(25)).toBe('25');
		expect(formatFps(null)).toBe('—');
		expect(formatResolution(1920, 1080, '16:9')).toBe('1920 × 1080 (16:9)');
		expect(formatResolution(720, 576, '')).toBe('720 × 576');
		expect(formatSampleRate(48000)).toBe('48 kHz');
		expect(formatSampleRate(44100)).toBe('44,1 kHz');
		expect(formatDurationSeconds(6732)).toBe('1 h 52 min');
		expect(formatDurationSeconds(90)).toBe('1 min');
		expect(formatDurationSeconds(45)).toBe('45 s');
		expect(formatDurationSeconds(null)).toBe('');
	});

	it('formatea la fecha de modificación', () => {
		expect(formatDateTime('2026-10-26T10:00:00Z')).toContain('2026');
		expect(formatDateTime('fecha inválida')).toBe('fecha inválida');
		expect(formatDateTime(null)).toBe('');
	});

	it('elige la imagen y el pie de una tarjeta de episodio', () => {
		const item = episodeItem(1, 2, 3);
		expect(episodeImageURL(item)).toBeNull();
		expect(episodeImageURL({ ...item, series_poster_url: 'serie.jpg' })).toBe('serie.jpg');
		expect(
			episodeImageURL({
				...item,
				still_url: 'still.jpg',
				series_backdrop_url: 'fondo.jpg'
			})
		).toBe('still.jpg');
		expect(
			episodeCardSubtitle({ ...item, series_title: 'The Big Bang Theory' })
		).toBe('The Big Bang Theory · T2 E3');
	});
});
