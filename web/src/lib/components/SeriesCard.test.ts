import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
import type { Series } from '$lib/api';
import SeriesCard from './SeriesCard.svelte';

const series: Series = {
	id: 3,
	title: 'The Big Bang Theory',
	file_title: 'The Big Bang Theory',
	year: 2007,
	poster_url: null,
	genres: [],
	season_count: 2,
	episode_count: 12,
	watched_count: 4,
	match_status: 'auto'
};

describe('SeriesCard', () => {
	it('muestra la serie y el enlace', () => {
		const { body } = render(SeriesCard, { props: { series } });
		expect(body).toContain('The Big Bang Theory');
		expect(body).toContain('href="/series/3"');
		expect(body).toContain('2 temporadas');
	});
	it('muestra la puntuación de la serie cuando existe', () => {
		const { body } = render(SeriesCard, { props: { series: { ...series, rating: 8.2 } } });
		expect(body).toContain('series-rating');
		expect(body).toContain('>8,2</span>');
		expect(body).toContain('aria-label="Puntuación 8,2 de 10"');
	});

	it('omite la puntuación si falta o es cero', () => {
		const missing = render(SeriesCard, { props: { series } }).body;
		expect(missing).not.toContain('series-rating');

		const zero = render(SeriesCard, { props: { series: { ...series, rating: 0 } } }).body;
		expect(zero).not.toContain('series-rating');
	});

	it('pone el botón de favorito fuera del enlace', () => {
		const { body } = render(SeriesCard, { props: { series: { ...series, favorite: false } } });
		expect(body).toContain('action="/lists"');
		expect(body).toContain('aria-label="Añadir a favoritos"');
		expect(body.indexOf('</a>')).toBeLessThan(body.indexOf('<form'));
	});
});
