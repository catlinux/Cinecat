import { describe, expect, it } from 'vitest';
import type { MediaItem } from './api';
import {
	SERIES_PAGE_SIZE,
	clearSeriesFiltersHref,
	formatEpisodeCode,
	formatEpisodeHeading,
	formatSeriesCounts,
	formatSeriesProgress,
	readSeriesFilters,
	seasonWatched,
	selectSeriesSeason,
	seriesApiQuery,
	seriesFilterChips,
	seriesPlayChoice,
	seriesWatched
} from './series';

function episode(id: number, season: number, episodeNumber: number, title: string): MediaItem {
	return {
		id,
		library_id: 1,
		path: `/media/series/${id}.mkv`,
		title,
		file_title: `file-${id}`,
		year: 2000,
		kind: 'episode',
		season,
		episode: episodeNumber,
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
		watched: false,
		progress: null,
		collections: [],
		series_id: 5,
		series_title: 'Serie',
		episode_title: title,
		still_url: null
	};
}

describe('filtros de series', () => {
	it('lee q, letra, género, orden y unmatched', () => {
		const filters = readSeriesFilters(
			new URL('http://localhost/series?q=robot&letter=m&genre=Drama&sort=year&unmatched=1'),
			true
		);
		expect(filters).toEqual({
			q: 'robot',
			letter: 'M',
			genre: 'Drama',
			sort: 'year',
			unmatched: true
		});
	});

	it('normaliza valores no válidos y oculta unmatched a quien no es admin', () => {
		expect(
			readSeriesFilters(
				new URL('http://localhost/series?letter=abc&sort=otro&unmatched=1'),
				false
			)
		).toEqual({ q: '', letter: null, genre: null, sort: 'title', unmatched: false });
	});

	it('construye la consulta de la API con paginación', () => {
		const filters = readSeriesFilters(
			new URL('http://localhost/series?q=big&letter=B&genre=Comedia&sort=added&unmatched=1'),
			true
		);
		const params = seriesApiQuery(filters, 2);
		expect(params.get('limit')).toBe(String(SERIES_PAGE_SIZE));
		expect(params.get('offset')).toBe(String(SERIES_PAGE_SIZE));
		expect(params.get('q')).toBe('big');
		expect(params.get('letter')).toBe('B');
		expect(params.get('genre')).toBe('Comedia');
		expect(params.get('sort')).toBe('added');
		expect(params.get('unmatched')).toBe('true');
	});

	it('genera etiquetas y limpia filtros conservando la ruta', () => {
		const filters = readSeriesFilters(
			new URL('http://localhost/series?q=big&letter=B&genre=Comedia&unmatched=1&page=2'),
			true
		);
		const chips = seriesFilterChips(filters, '/series?q=big&letter=B&genre=Comedia&unmatched=1&page=2');
		expect(chips.map((chip) => chip.label)).toEqual([
			'Búsqueda: “big”',
			'Letra: B',
			'Género: Comedia',
			'Sin identificar'
		]);
		expect(clearSeriesFiltersHref('/series?q=big&letter=B&page=2')).toBe('/series');
	});
});

describe('textos de series y episodios', () => {
	it('usa singular y plural en temporadas y episodios', () => {
		expect(formatSeriesCounts(0, 0)).toBe('0 temporadas · 0 episodios');
		expect(formatSeriesCounts(1, 1)).toBe('1 temporada · 1 episodio');
		expect(formatSeriesCounts(2, 12)).toBe('2 temporadas · 12 episodios');
	});

	it('calcula la insignia de progreso', () => {
		expect(formatSeriesProgress(0, 10)).toBe('0/10 vistos');
		expect(formatSeriesProgress(3, 10)).toBe('3/10 vistos');
		expect(formatSeriesProgress(10, 10)).toBe('Vista');
		expect(formatSeriesProgress(0, 0)).toBe('');
		expect(seriesWatched({ watched_count: 2, episode_count: 2 })).toBe(true);
		expect(seasonWatched({ watched_count: 1, episode_count: 2 })).toBe(false);
	});

	it('formatea la posición y el título del episodio', () => {
		const item = episode(1, 2, 5, 'El episodio');
		expect(formatEpisodeCode(item)).toBe('T2 E5');
		expect(formatEpisodeHeading(item)).toBe('E5 · El episodio');
		expect(formatEpisodeCode(episode(2, 1, 2, 'Dos'))).toBe('T1 E2');
	});

	it('elige la temporada pedida, la del siguiente episodio o la primera', () => {
		const seasons = [{ season: 1 }, { season: 2 }];
		expect(selectSeriesSeason(seasons, episode(1, 2, 1, 'x'), '2')).toBe(2);
		expect(selectSeriesSeason(seasons, episode(1, 2, 1, 'x'), '9')).toBe(1);
		expect(selectSeriesSeason(seasons, episode(1, 2, 1, 'x'), null)).toBe(2);
		expect(selectSeriesSeason(seasons, null, null)).toBe(1);
		expect(selectSeriesSeason([], null, '1')).toBeNull();
	});
});

describe('botón principal de la serie', () => {
	it('empieza por el primer episodio sin actividad', () => {
		const first = episode(1, 1, 1, 'Piloto');
		const choice = seriesPlayChoice({ watched_count: 0, episode_count: 2 }, first, first);
		expect(choice?.label).toBe('Empezar · T1 E1');
		expect(choice?.item.id).toBe(1);
	});

	it('continúa con el siguiente episodio o el que tiene progreso', () => {
		const next = episode(2, 2, 5, 'El episodio');
		const choice = seriesPlayChoice({ watched_count: 1, episode_count: 6 }, next, null);
		expect(choice?.label).toBe('Continuar · T2 E5 · El episodio');
		expect(choice?.item.id).toBe(2);

		const withProgress = { ...next, progress: { position_seconds: 20, duration_seconds: 100, updated_at: '2026-01-01T00:00:00Z' } };
		const resumed = seriesPlayChoice({ watched_count: 0, episode_count: 6 }, withProgress, null);
		expect(resumed?.label).toContain('Continuar');
	});

	it('vuelve a ver el primero cuando todo está visto', () => {
		const first = episode(1, 1, 1, 'Piloto');
		const choice = seriesPlayChoice({ watched_count: 4, episode_count: 4 }, null, first);
		expect(choice?.label).toBe('Volver a ver');
		expect(choice?.item.id).toBe(1);
	});

	it('no muestra botón sin episodios', () => {
		expect(seriesPlayChoice({ watched_count: 0, episode_count: 0 }, null, null)).toBeNull();
	});
});
