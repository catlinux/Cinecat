import { describe, expect, it } from 'vitest';
import {
	parseVideoLibraryID,
	parseVideoPath,
	parseVideoRoot,
	videoBreadcrumbs,
	videoCountLabel,
	videoFolderHref,
	videoHeading,
	videoRootHref
} from './videos';

describe('parámetros de vídeos', () => {
	it('valida el identificador de biblioteca', () => {
		expect(parseVideoLibraryID('7')).toBe(7);
		expect(parseVideoLibraryID('0')).toBeNull();
		expect(parseVideoLibraryID('-1')).toBeNull();
		expect(parseVideoLibraryID('abc')).toBeNull();
		expect(parseVideoLibraryID(null)).toBeNull();
	});

	it('valida root opcional y positivo', () => {
		expect(parseVideoRoot(null)).toEqual({ ok: true, value: null });
		expect(parseVideoRoot('')).toEqual({ ok: true, value: null });
		expect(parseVideoRoot('3')).toEqual({ ok: true, value: 3 });
		expect(parseVideoRoot('0').ok).toBe(false);
		expect(parseVideoRoot('x').ok).toBe(false);
	});

	it('valida rutas relativas y rechaza travesías o rutas absolutas', () => {
		expect(parseVideoPath(null, true)).toEqual({ ok: true, value: '' });
		expect(parseVideoPath('', true)).toEqual({ ok: true, value: '' });
		expect(parseVideoPath('Pelis/Sub', true)).toEqual({ ok: true, value: 'Pelis/Sub' });
		expect(parseVideoPath('Pelis', false).ok).toBe(false);
		expect(parseVideoPath('/Pelis', true).ok).toBe(false);
		expect(parseVideoPath('../Pelis', true).ok).toBe(false);
		expect(parseVideoPath('Pelis//Sub', true).ok).toBe(false);
		expect(parseVideoPath('Pelis/./Sub', true).ok).toBe(false);
		expect(parseVideoPath('C:\\Pelis', true).ok).toBe(false);
	});
});

describe('enlaces y migas de vídeos', () => {
	it('construye los enlaces de raíz y carpeta sin exponer rutas absolutas', () => {
		expect(videoRootHref(4, 9)).toBe('/videos/4?root=9');
		expect(videoFolderHref(4, 9, 'Conciertos/Año 2020')).toBe(
			'/videos/4?root=9&path=Conciertos%2FA%C3%B1o+2020'
		);
	});

	it('construye migas clicables con biblioteca, raíz y subcarpetas', () => {
		expect(
			videoBreadcrumbs(
				4,
				'Vídeos de casa',
				{ id: 9, name: 'Conciertos' },
				[
					{ name: 'Conciertos', path: '' },
					{ name: 'Año 2020', path: 'Año 2020' }
				]
			)
		).toEqual([
			{ name: 'Vídeos', href: '/videos' },
			{ name: 'Vídeos de casa', href: '/videos/4' },
			{ name: 'Conciertos', href: '/videos/4?root=9' },
			{ name: 'Año 2020', href: '/videos/4?root=9&path=A%C3%B1o+2020' }
		]);
	});

	it('deja solo Vídeos y biblioteca cuando no hay raíz', () => {
		expect(videoBreadcrumbs(4, 'Vídeos de casa', null, [])).toEqual([
			{ name: 'Vídeos', href: '/videos' },
			{ name: 'Vídeos de casa', href: '/videos/4' }
		]);
	});
});

describe('textos de vídeos', () => {
	it('cuenta vídeos en singular y plural', () => {
		expect(videoCountLabel(0)).toBe('0 vídeos');
		expect(videoCountLabel(1)).toBe('1 vídeo');
		expect(videoCountLabel(12)).toBe('12 vídeos');
	});

	it('elige el nombre de la carpeta actual o el título de reserva', () => {
		expect(videoHeading('Pelis/Sub', [{ name: 'Pelis', path: 'Pelis' }, { name: 'Sub', path: 'Pelis/Sub' }], 'Vídeos')).toBe('Sub');
		expect(videoHeading('', [], 'Vídeos')).toBe('Vídeos');
	});
});
