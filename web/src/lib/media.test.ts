import { describe, expect, it } from 'vitest';
import {
	formatGenres,
	formatMovieMeta,
	formatRating,
	formatRuntime,
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
