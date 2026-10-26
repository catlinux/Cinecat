import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
import type { MediaItem } from '$lib/api';
import PosterCard from './PosterCard.svelte';

const item: MediaItem = {
	id: 7,
	library_id: 1,
	path: '/media/peliculas/Matrix (1999).mp4',
	title: 'Matrix',
	year: 1999,
	kind: 'movie',
	season: null,
	episode: null,
	tmdb_id: 603,
	overview: null,
	poster_url: null
};

describe('PosterCard', () => {
	it('muestra el estado de respaldo con el título y enlaza a la ficha', () => {
		const { body } = render(PosterCard, { props: { item } });

		expect(body).toContain('Matrix');
		expect(body).toContain('Película');
		expect(body).toContain('/items/7');
		expect(body).toContain('1999');
		expect(body).toContain('--poster-a');
	});

	it('muestra el cartel real cuando posterUrl está informado', () => {
		const { body } = render(PosterCard, {
			props: { item, posterUrl: 'https://images.example/matrix.jpg' }
		});

		expect(body).toContain('<img');
		expect(body).toContain('src="https://images.example/matrix.jpg"');
		expect(body).toContain('alt="Cartel de Matrix"');
		expect(body).not.toContain('poster-fallback');
	});

	it('usa la misma paleta en cada render para el mismo título', () => {
		const first = render(PosterCard, { props: { item } }).body;
		const second = render(PosterCard, { props: { item } }).body;

		const color = /--poster-a: (#[0-9a-f]{6})/.exec(first)?.[1];
		expect(color).toBeTruthy();
		expect(second).toContain(`--poster-a: ${color}`);
	});
});
