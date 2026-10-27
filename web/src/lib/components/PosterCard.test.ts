import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
import type { MediaItem } from '$lib/api';
import PosterCard from './PosterCard.svelte';

const item: MediaItem = {
	id: 7,
	library_id: 1,
	path: '/media/peliculas/Matrix (1999).mp4',
	title: 'Matrix',
	file_title: 'Matrix',
	year: 1999,
	kind: 'movie',
	season: null,
	episode: null,
	tmdb_id: 603,
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
	series_id: null,
	series_title: null,
	episode_title: null,
	still_url: null
};

describe('PosterCard', () => {
	it('muestra el estado de respaldo con el título y enlaza a la ficha', () => {
		const { body } = render(PosterCard, { props: { item } });

		expect(body).toContain('Matrix');
		expect(body).toContain('Película');
		expect(body).toContain('/items/7');
		expect(body).toContain('1999');
		expect(body).toContain('--poster-a');
		expect(body).not.toContain('Visto');
	});

	it('muestra la puntuación con una decimal y coma', () => {
		const { body } = render(PosterCard, { props: { item: { ...item, rating: 7.4 } } });

		expect(body).toContain('poster-rating');
		expect(body).toContain('>7,4</span>');
		expect(body).toContain('aria-label="Puntuación 7,4 de 10"');
	});

	it('omite la puntuación si no hay nota o no es de una película', () => {
		const withoutRating = render(PosterCard, { props: { item: { ...item, rating: null } } }).body;
		expect(withoutRating).not.toContain('poster-rating');

		const zero = render(PosterCard, { props: { item: { ...item, rating: 0 } } }).body;
		expect(zero).not.toContain('poster-rating');

		const clip = render(PosterCard, {
			props: { item: { ...item, library_kind: 'clips', rating: 7.4 } }
		}).body;
		expect(clip).not.toContain('poster-rating');

		const episode = render(PosterCard, {
			props: { item: { ...item, kind: 'episode', rating: 7.4 } }
		}).body;
		expect(episode).not.toContain('poster-rating');
	});

	it('marca discretamente los ítems vistos con texto accesible', () => {
		const { body } = render(PosterCard, { props: { item: { ...item, watched: true } } });

		expect(body).toContain('Visto');
		expect(body).toContain('poster-watched');
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

	it('muestra el progreso y el texto de continuación', () => {
		const { body } = render(PosterCard, {
			props: {
				item: {
					...item,
					progress: {
						position_seconds: 4320,
						duration_seconds: 7200,
						updated_at: '2026-10-26T10:00:00Z'
					}
				}
			}
		});

		expect(body).toContain('poster-progress');
		expect(body).toContain('role="progressbar"');
		expect(body).toContain('aria-valuenow="60"');
		expect(body).toContain('Continúas por 1 h 12 min');
	});

	it('en episodios con progreso indica la serie real y el capítulo', () => {
		const { body } = render(PosterCard, {
			props: {
				item: {
					...item,
					kind: 'episode',
					season: 2,
					episode: 5,
					series_title: 'The Big Bang Theory',
					progress: {
						position_seconds: 300,
						duration_seconds: 1800,
						updated_at: '2026-10-26T10:00:00Z'
					}
				}
			}
		});

		expect(body).toContain('The Big Bang Theory · T2 E5');
	});

	it('usa el fotograma apaisado y la etiqueta de episodio', () => {
		const { body } = render(PosterCard, {
			props: {
				item: {
					...item,
					kind: 'episode',
					season: 1,
					episode: 1,
					series_title: 'The Big Bang Theory',
					still_url: 'https://images.example/still.jpg'
				}
			}
		});

		expect(body).toContain('episode-card');
		expect(body).toContain('episode-frame');
		expect(body).toContain('src="https://images.example/still.jpg"');
		expect(body).toContain('alt=""');
		expect(body).toContain('EPISODIO');
		expect(body).toContain('The Big Bang Theory · T1 E1');
	});

	it('cae al cartel de la serie y después al respaldo', () => {
		const withSeriesArt = render(PosterCard, {
			props: {
				item: {
					...item,
					kind: 'episode',
					season: 1,
					episode: 1,
					series_poster_url: 'https://images.example/serie.jpg'
				}
			}
		}).body;
		expect(withSeriesArt).toContain('src="https://images.example/serie.jpg"');

		const withoutArt = render(PosterCard, {
			props: {
				item: { ...item, kind: 'episode', season: 1, episode: 1 }
			}
		}).body;
		expect(withoutArt).toContain('episode-fallback');
		expect(withoutArt).toContain('Matrix');
	});

	it('muestra el botón de favorito como hermano del enlace', () => {
		const { body } = render(PosterCard, {
			props: { item: { ...item, favorite: true } }
		});

		expect(body).toContain('action="/lists"');
		expect(body).toContain('aria-label="Quitar de favoritos"');
		expect(body).toContain('aria-pressed="true"');
		expect(body.indexOf('</a>')).toBeLessThan(body.indexOf('<form'));
	});

	it('omite el botón si el DTO no trae marcas', () => {
		const { body } = render(PosterCard, { props: { item } });
		expect(body).not.toContain('action="/lists"');
	});

	it('usa la misma paleta en cada render para el mismo título', () => {
		const first = render(PosterCard, { props: { item } }).body;
		const second = render(PosterCard, { props: { item } }).body;

		const color = /--poster-a: (#[0-9a-f]{6})/.exec(first)?.[1];
		expect(color).toBeTruthy();
		expect(second).toContain(`--poster-a: ${color}`);
	});
});

describe('indicador de colección', () => {
	it('muestra el nombre de la colección y el icono', () => {
		const { body } = render(PosterCard, {
			props: { item: { ...item, collections: [{ id: 5, name: 'La saga' }] } }
		});

		expect(body).toContain('poster-collections');
		expect(body).toContain('poster-collections-name');
		expect(body).toContain('La saga');
		expect(body).toContain('title="Colección: La saga"');
		expect(body).toContain('aria-label="Colección: La saga"');
	});

	it('muestra «+N» y el title con todos los nombres', () => {
		const { body } = render(PosterCard, {
			props: {
				item: {
					...item,
					collections: [
						{ id: 5, name: 'Saga Harry Potter' },
						{ id: 6, name: 'Otra saga' }
					]
				}
			}
		});

		expect(body).toContain('Saga Harry Potter +1');
		expect(body).toContain('title="Colecciones: Saga Harry Potter, Otra saga"');
		expect(body).toContain('aria-label="Colecciones: Saga Harry Potter, Otra saga"');
	});

	it('no aparece sin colecciones, en clips ni en episodios', () => {
		const none = render(PosterCard, { props: { item } }).body;
		expect(none).not.toContain('poster-collections');

		const clip = render(PosterCard, {
			props: {
				item: {
					...item,
					library_kind: 'clips',
					collections: [{ id: 5, name: 'La saga' }]
				}
			}
		}).body;
		expect(clip).not.toContain('poster-collections');

		const episode = render(PosterCard, {
			props: {
				item: {
					...item,
					kind: 'episode',
					collections: [{ id: 5, name: 'La saga' }]
				}
			}
		}).body;
		expect(episode).not.toContain('poster-collections');
	});
});
