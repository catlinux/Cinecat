import { describe, expect, it } from 'vitest';
import { clampPage, pageCount, pageHref, pageWindow } from './pagination';

describe('pageWindow', () => {
	it('con una sola página devuelve solo la primera', () => {
		expect(pageWindow(1, 1)).toEqual([1]);
		expect(pageWindow(8, 1)).toEqual([1]);
	});

	it('con pocas páginas las muestra todas', () => {
		expect(pageWindow(3, 5)).toEqual([1, 2, 3, 4, 5]);
		expect(pageWindow(1, 7)).toEqual([1, 2, 3, 4, 5, 6, 7]);
	});

	it('en el primer extremo deja hueco antes de la última', () => {
		expect(pageWindow(1, 10)).toEqual([1, 2, 3, null, 10]);
	});

	it('en el centro muestra dos vecinos y huecos', () => {
		expect(pageWindow(5, 10)).toEqual([1, null, 3, 4, 5, 6, 7, null, 10]);
	});

	it('en el último extremo deja hueco después de la primera', () => {
		expect(pageWindow(10, 10)).toEqual([1, null, 8, 9, 10]);
	});
});

describe('clampPage', () => {
	it('normaliza valores inválidos, cero y negativos a 1', () => {
		expect(clampPage('abc', 5)).toBe(1);
		expect(clampPage('', 5)).toBe(1);
		expect(clampPage('0', 5)).toBe(1);
		expect(clampPage('-2', 5)).toBe(1);
		expect(clampPage(null, 5)).toBe(1);
	});

	it('recorta a la última página', () => {
		expect(clampPage('99', 5)).toBe(5);
		expect(clampPage(5, 10)).toBe(5);
	});

	it('acepta una página válida y trunca decimales', () => {
		expect(clampPage(' 4 ', 10)).toBe(4);
		expect(clampPage(3.7, 10)).toBe(3);
	});
});

describe('pageCount', () => {
	it('con cero resultados siempre hay una página', () => {
		expect(pageCount(0, 48)).toBe(1);
	});

	it('calcula las páginas necesarias', () => {
		expect(pageCount(1, 48)).toBe(1);
		expect(pageCount(48, 48)).toBe(1);
		expect(pageCount(49, 48)).toBe(2);
		expect(pageCount(96, 48)).toBe(2);
		expect(pageCount(97, 48)).toBe(3);
	});
});

describe('pageHref', () => {
	it('conserva los demás parámetros y cambia page', () => {
		const href = pageHref('/movies?q=matrix&field=cast&letter=%23', 3);
		const url = new URL(href, 'http://cinecat.local');
		expect(url.pathname).toBe('/movies');
		expect(url.searchParams.get('q')).toBe('matrix');
		expect(url.searchParams.get('field')).toBe('cast');
		expect(url.searchParams.get('letter')).toBe('#');
		expect(url.searchParams.get('page')).toBe('3');
	});

	it('acepta una URL absoluta y devuelve una ruta relativa', () => {
		const href = pageHref(new URL('http://localhost/movies?sort=year'), 2);
		expect(href).toBe('/movies?sort=year&page=2');
	});
});
