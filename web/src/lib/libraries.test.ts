import { describe, expect, it } from 'vitest';
import {
	LIBRARY_KIND_OPTIONS,
	breadcrumbs,
	browserParent,
	libraryKindLabel,
	parentPath
} from './libraries';

describe('textos de tipo de biblioteca', () => {
	it('traduce los tres tipos', () => {
		expect(libraryKindLabel('video')).toBe('Películas y series');
		expect(libraryKindLabel('clips')).toBe('Vídeos');
		expect(libraryKindLabel('music')).toBe('Música');
		expect(libraryKindLabel(undefined)).toBe('Películas y series');
	});

	it('deja Vídeos disponible y Música como próximamente', () => {
		expect(LIBRARY_KIND_OPTIONS).toEqual([
			{ value: 'video', label: 'Películas y series', disabled: false },
			{ value: 'clips', label: 'Vídeos (videoclips, conciertos…)', disabled: false },
			{ value: 'music', label: 'Música', disabled: true }
		]);
	});
});

describe('migas de pan', () => {
	it('construye las migas de una ruta de Linux sin incluir la raíz del sistema', () => {
		expect(breadcrumbs('/media/Multimedia/Pelis')).toEqual([
			{ name: 'media', path: '/media' },
			{ name: 'Multimedia', path: '/media/Multimedia' },
			{ name: 'Pelis', path: '/media/Multimedia/Pelis' }
		]);
	});

	it('construye las migas de una ruta de Windows', () => {
		expect(breadcrumbs('C:\\Media\\Pelis')).toEqual([
			{ name: 'C:\\', path: 'C:\\' },
			{ name: 'Media', path: 'C:\\Media' },
			{ name: 'Pelis', path: 'C:\\Media\\Pelis' }
		]);
	});
});

describe('subir de carpeta', () => {
	it('sube una ruta de Linux', () => {
		expect(parentPath('/media/Multimedia/Pelis')).toBe('/media/Multimedia');
		expect(parentPath('/media')).toBe('/');
		expect(parentPath('/')).toBeNull();
	});

	it('sube una ruta de Windows sin salir de la unidad', () => {
		expect(parentPath('C:\\Media\\Pelis')).toBe('C:\\Media');
		expect(parentPath('C:\\Media')).toBe('C:\\');
		expect(parentPath('C:\\')).toBeNull();
	});

	it('usa el parent del servidor y lo deja vacío en una raíz', () => {
		expect(
			browserParent({
				path: '/media/pelis',
				parent: '/media',
				entries: [],
				truncated: false
			})
		).toBe('/media');
		expect(
			browserParent({
				path: '/media',
				parent: null,
				entries: [],
				truncated: false
			})
		).toBeNull();
	});
});
