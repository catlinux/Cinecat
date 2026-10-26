import { describe, expect, it } from 'vitest';
import {
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
	formatRuntime,
	formatSampleRate,
	matchStatusLabel,
	overviewInSpanish,
	splitCast,
	truncateSynopsis
} from './media';

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
});
