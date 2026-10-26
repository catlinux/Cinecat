import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
import type { SeriesSeason } from '$lib/api';
import SeasonTabs from './SeasonTabs.svelte';

const seasons: SeriesSeason[] = [
	{
		season: 1,
		name: 'Temporada 1',
		overview: null,
		poster_url: 'https://images.example/t1.jpg',
		episode_count: 10,
		watched_count: 0
	},
	{
		season: 2,
		name: 'Temporada 2',
		overview: null,
		poster_url: null,
		episode_count: 8,
		watched_count: 3
	}
];

function renderTabs(seriesPoster: string | null = null): string {
	return render(SeasonTabs, {
		props: { seasons, activeSeason: 1, seriesPoster, seriesTitle: 'The Big Bang Theory' }
	}).body;
}

describe('selector de temporadas', () => {
	it('muestra tarjetas con carátula, nombre y vistos/total', () => {
		const body = renderTabs();

		expect(body).toContain('Temporada 1');
		expect(body).toContain('0/10 vistos');
		expect(body).toContain('Temporada 2');
		expect(body).toContain('3/8 vistos');
		expect(body).toContain('href="?season=1"');
		expect(body).toContain('href="?season=2"');
		expect(body).toContain('aria-current="page"');
		expect(body).toContain('season-card-active');
		expect(body).toContain('https://images.example/t1.jpg');
		expect(body).toContain('alt=""');
		expect(body).toContain('loading="lazy"');
	});

	it('usa el cartel de la serie si la temporada no tiene', () => {
		const body = renderTabs('https://images.example/serie.jpg');

		expect(body).toContain('https://images.example/serie.jpg');
		expect((body.match(/https:\/\/images\.example\/serie\.jpg/g) ?? []).length).toBe(1);
	});

	it('dibuja el respaldo con iniciales si no hay ninguna imagen', () => {
		const body = renderTabs();

		expect(body).toContain('season-card-fallback');
		expect(body).toContain('>TT<');
	});
});
