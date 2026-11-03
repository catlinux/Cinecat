import { describe, expect, it } from 'vitest';
import {
	nextPhotoIndex,
	parsePhotoFolder,
	parsePhotoSort,
	photoAspectRatio,
	photoBreadcrumbs,
	photoCountLabel,
	photoDateLabel,
	photoFolderHref,
	photoLibraryHref,
	photoOriginalURL,
	photoThumbURL,
	previousPhotoIndex
} from './photos';

describe('filtros y rutas de fotos', () => {
	it('normaliza el orden y las carpetas relativas', () => {
		expect(parsePhotoSort('NAME')).toBe('name');
		expect(parsePhotoSort('otro')).toBe('date');
		expect(parsePhotoFolder('')).toBe('');
		expect(parsePhotoFolder('2024\\Verano')).toBe('2024/Verano');
		expect(parsePhotoFolder('../fuera')).toBeNull();
		expect(parsePhotoFolder('/etc')).toBeNull();
		expect(parsePhotoFolder('a//b')).toBeNull();
		expect(parsePhotoFolder('C:\\fotos')).toBeNull();
	});

	it('construye migas y enlaces conservando el orden', () => {
		expect(photoBreadcrumbs('2024/Verano')).toEqual([
			{ name: '2024', path: '2024' },
			{ name: 'Verano', path: '2024/Verano' }
		]);
		expect(photoLibraryHref(7)).toBe('/photos/7');
		expect(photoFolderHref(7, '2024/Verano', 'name')).toBe(
			'/photos/7?folder=2024%2FVerano&sort=name'
		);
		expect(photoFolderHref(7, '', 'date')).toBe('/photos/7');
		expect(photoThumbURL(3)).toBe('/photos/thumb/3');
		expect(photoOriginalURL(3)).toBe('/photos/original/3');
	});

	it('calcula siguiente, anterior y etiquetas', () => {
		expect(nextPhotoIndex(0, 3)).toBe(1);
		expect(nextPhotoIndex(2, 3)).toBe(0);
		expect(nextPhotoIndex(9, 3)).toBe(0);
		expect(previousPhotoIndex(0, 3)).toBe(2);
		expect(previousPhotoIndex(1, 3)).toBe(0);
		expect(previousPhotoIndex(9, 3)).toBe(2);
		expect(photoCountLabel(0)).toBe('0 fotos');
		expect(photoCountLabel(1)).toBe('1 foto');
		expect(photoCountLabel(4)).toBe('4 fotos');
		expect(photoAspectRatio(4, 3)).toBe('4 / 3');
		expect(photoAspectRatio(0, 0)).toBe('4 / 3');
	});

	it('formatea fechas y tolera valores ausentes', () => {
		expect(photoDateLabel(null)).toBe('');
		expect(photoDateLabel(Number.NaN)).toBe('');
		expect(photoDateLabel(Date.UTC(2020, 0, 2, 12) / 1000)).toContain('2020');
	});
});
