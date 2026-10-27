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
	it('pone el botón de favorito fuera del enlace', () => {
		const { body } = render(SeriesCard, { props: { series: { ...series, favorite: false } } });
		expect(body).toContain('action="/lists"');
		expect(body).toContain('aria-label="Añadir a favoritos"');
		expect(body.indexOf('</a>')).toBeLessThan(body.indexOf('<form'));
	});
});
