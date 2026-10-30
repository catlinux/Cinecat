import { describe, expect, it } from 'vitest';
import type { MediaItem } from './api';
import {
	collectionApiQuery,
	collectionFiltersActive,
	collectionIdentificationLabel,
	firstUnwatched,
	formatCollectionCount,
	formatMissingYear,
	formatViewedCount,
	hasUnwatched,
	moveItem,
	readCollectionFilters,
	tmdbCollectionURL,
	viewedCount
} from './collections';

function movie(id: number, watched: boolean): MediaItem {
	return {
		id,
		library_id: 1,
		path: `/media/${id}.mkv`,
		title: `Pelicula ${id}`,
		file_title: `Pelicula ${id}`,
		year: 2000,
		kind: 'movie',
		season: null,
		episode: null,
		tmdb_id: null,
		overview: null,
		poster_url: null,
		poster_custom: false,
		original_title: null,
		imdb_id: null,
		tagline: null,
		genres: [],
		runtime_minutes: null,
		rating: null,
		directors: [],
		cast: [],
		backdrop_url: null,
		backdrop_custom: false,
		overview_language: null,
		match_status: null,
		match_score: null,
		watched,
		progress: null,
		collections: [],
		series_id: null,
		series_title: null,
		episode_title: null,
		still_url: null
	};
}

describe('textos y estado de colecciones', () => {
	it('usa singular y plural para el número de películas', () => {
		expect(formatCollectionCount(0)).toBe('0 películas');
		expect(formatCollectionCount(1)).toBe('1 película');
		expect(formatCollectionCount(5)).toBe('5 películas');
	});

	it('calcula cuántas películas se han visto', () => {
		const items = [movie(1, true), movie(2, false), movie(3, true)];
		expect(viewedCount(items)).toBe(2);
		expect(formatViewedCount(2, 3)).toBe('2 de 3 vistas');
		expect(formatViewedCount(1, 1)).toBe('1 de 1 vista');
	});

	it('elige la primera sin ver y cae en la primera si todas están vistas', () => {
		expect(firstUnwatched([movie(1, true), movie(2, false), movie(3, false)])?.id).toBe(2);
		expect(firstUnwatched([movie(1, true), movie(2, true)])?.id).toBe(1);
		expect(firstUnwatched([])).toBeNull();
		expect(hasUnwatched([movie(1, true), movie(2, false)])).toBe(true);
		expect(hasUnwatched([movie(1, true)])).toBe(false);
	});

	it('calcula la nueva lista al subir o bajar', () => {
		expect(moveItem([1, 2, 3], 2, 'up')).toEqual([2, 1, 3]);
		expect(moveItem([1, 2, 3], 2, 'down')).toEqual([1, 3, 2]);
		expect(moveItem([1, 2, 3], 1, 'up')).toEqual([1, 2, 3]);
		expect(moveItem([1, 2, 3], 3, 'down')).toEqual([1, 2, 3]);
		expect(moveItem([1], 1, 'up')).toEqual([1]);
		expect(moveItem([1, 2], 9, 'up')).toEqual([1, 2]);
	});
});

describe('identificación de colecciones en TMDB', () => {
	it('construye el enlace solo con un id válido', () => {
		expect(tmdbCollectionURL(900)).toBe('https://www.themoviedb.org/collection/900');
		expect(tmdbCollectionURL(null)).toBeNull();
		expect(tmdbCollectionURL(0)).toBeNull();
		expect(tmdbCollectionURL(-3)).toBeNull();
	});

	it('describe el estado de identificación', () => {
		expect(collectionIdentificationLabel({ tmdb_collection_id: 900 })).toBe(
			'Identificada en TMDB con el id 900'
		);
		expect(collectionIdentificationLabel({ tmdb_collection_id: null })).toBe(
			'Sin identificar en TMDB'
		);
		expect(collectionIdentificationLabel({})).toBe('Sin identificar en TMDB');
	});

	it('formatea el año de una película que falta', () => {
		expect(formatMissingYear(1999)).toBe('1999');
		expect(formatMissingYear(0)).toBe('Año desconocido');
		expect(formatMissingYear(Number.NaN)).toBe('Año desconocido');
	});
});

describe('filtros de colecciones', () => {
	it('lee q, sort y page de la URL', () => {
		const filters = readCollectionFilters(
			new URL('http://localhost/collections?q=saga&sort=size&page=3')
		);
		expect(filters).toEqual({ q: 'saga', sort: 'size', page: 3 });
	});

	it('normaliza sort y page no válidos', () => {
		const filters = readCollectionFilters(
			new URL('http://localhost/collections?sort=otro&page=-2')
		);
		expect(filters).toEqual({ q: '', sort: 'name', page: 1 });
	});

	it('construye la consulta de la API con página y filtros', () => {
		const params = collectionApiQuery({ q: 'saga', sort: 'added', page: 1 }, 2, 48);
		expect(params.get('limit')).toBe('48');
		expect(params.get('offset')).toBe('48');
		expect(params.get('q')).toBe('saga');
		expect(params.get('sort')).toBe('added');
	});

	it('detecta si hay filtros activos', () => {
		expect(collectionFiltersActive({ q: '', sort: 'name', page: 1 })).toBe(false);
		expect(collectionFiltersActive({ q: 'saga', sort: 'name', page: 1 })).toBe(true);
		expect(collectionFiltersActive({ q: '', sort: 'size', page: 1 })).toBe(true);
	});
});
