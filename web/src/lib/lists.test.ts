import { describe, expect, it } from 'vitest';
import {
	listPageHref,
	listToggleLabel,
	listToggleResult,
	parseListKind,
	parseToggleKind,
	readListsPage,
	sanitizeRedirectTo
} from './lists';

describe('listas de usuario', () => {
	it('sanea redirectTo a rutas internas', () => {
		expect(sanitizeRedirectTo('/items/7?x=1#y')).toBe('/items/7?x=1#y');
		expect(sanitizeRedirectTo('')).toBe('/');
		expect(sanitizeRedirectTo('//evil.com')).toBe('/');
		expect(sanitizeRedirectTo('https://evil.com')).toBe('/');
		expect(sanitizeRedirectTo('javascript:alert(1)')).toBe('/');
		expect(sanitizeRedirectTo('/\\evil')).toBe('/');
		expect(sanitizeRedirectTo('/ruta\nrara')).toBe('/');
	});

	it('revierte el estado optimista si la petición falla', () => {
		expect(listToggleResult(false, 'success')).toEqual({ active: true, error: null });
		expect(listToggleResult(true, 'success')).toEqual({ active: false, error: null });
		expect(listToggleResult(false, 'failure')).toEqual({
			active: false,
			error: 'No se pudo actualizar la lista.'
		});
		expect(listToggleResult(true, 'error')).toEqual({
			active: true,
			error: 'No se pudo actualizar la lista.'
		});
	});

	it('genera etiquetas accesibles según el estado', () => {
		expect(listToggleLabel('favorite', false)).toBe('Añadir a favoritos');
		expect(listToggleLabel('favorite', true)).toBe('Quitar de favoritos');
		expect(listToggleLabel('watchlist', false)).toBe('Añadir a próximas a ver');
		expect(listToggleLabel('watchlist', true)).toBe('Quitar de próximas a ver');
	});

	it('valida los parámetros de las páginas', () => {
		expect(readListsPage(new URL('http://localhost/favorites?kind=items&page=2'))).toEqual({
			kind: 'items',
			page: 2
		});
		expect(readListsPage(new URL('http://localhost/favorites?kind=raro&page=0'))).toEqual({
			kind: null,
			page: 1
		});
		expect(readListsPage(new URL('http://localhost/favorites?kind=series&page=2.5'))).toEqual({
			kind: 'series',
			page: 1
		});
		expect(parseListKind('collections')).toBe('collections');
		expect(parseToggleKind('collection')).toBe('collection');
		expect(parseToggleKind('items')).toBeNull();
	});

	it('construye enlaces de paginación por bloque', () => {
		expect(listPageHref('/favorites?page=4', 'items', 2)).toBe('/favorites?page=2&kind=items');
		expect(listPageHref('/favorites?kind=series&page=3', 'series', 1)).toBe(
			'/favorites?kind=series'
		);
	});
});
