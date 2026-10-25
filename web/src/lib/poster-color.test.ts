import { describe, expect, it } from 'vitest';
import { posterColors } from './poster-color';

describe('posterColors', () => {
	it('es determinista para el mismo título', () => {
		expect(posterColors('Matrix')).toEqual(posterColors('Matrix'));
		expect(posterColors('Amélie')).toEqual(posterColors('Amélie'));
	});

	it('devuelve colores de la paleta Marquesina', () => {
		const { from, to } = posterColors('El laberinto del fauno');
		expect(from).toMatch(/^#[0-9a-f]{6}$/);
		expect(to).toMatch(/^#[0-9a-f]{6}$/);
		expect(from).not.toBe(to);
	});
});
