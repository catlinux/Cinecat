import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
import type { MediaItem } from '$lib/api';
import VideoCard from './VideoCard.svelte';

function video(overrides: Partial<MediaItem> = {}): MediaItem {
	return {
		id: 12,
		library_id: 4,
		library_kind: 'clips',
		path: 'Conciertos/Clip.mp4',
		title: 'Cumple de Ana',
		file_title: 'Cumple de Ana',
		year: null,
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
		match_status: 'none',
		match_score: null,
		watched: false,
		progress: null,
		collections: [],
		series_id: null,
		series_title: null,
		episode_title: null,
		still_url: null,
		...overrides
	};
}

describe('tarjeta de vídeo', () => {
	it('pinta la portada de respaldo, el título y el enlace a la ficha', () => {
		const { body } = render(VideoCard, { props: { item: video() } });

		expect(body).toContain('class="video-card');
		expect(body).toContain('href="/items/12"');
		expect(body).toContain('Cumple de Ana');
		expect(body).toContain('video-fallback');
		expect(body).toContain('--video-a:');
	});

	it('muestra visto y la barra de progreso', () => {
		const { body } = render(VideoCard, {
			props: {
				item: video({
					watched: true,
					progress: {
						position_seconds: 30,
						duration_seconds: 120,
						updated_at: '2026-10-27T10:00:00Z'
					}
				})
			}
		});

		expect(body).toContain('Visto');
		expect(body).toContain('role="progressbar"');
		expect(body).toContain('aria-valuenow="25"');
		expect(body).toContain('Continúas por 30 s');
	});

	it('escapa el título', () => {
		const { body } = render(VideoCard, {
			props: { item: video({ title: '<script>alert(1)</script>' }) }
		});

		expect(body).toContain('&lt;script>alert(1)&lt;/script>');
		expect(body).not.toContain('<script>alert(1)</script>');
	});
});
