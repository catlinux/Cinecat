import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
import type { MusicAlbum } from '$lib/api';
import AlbumCard from './AlbumCard.svelte';

const album: MusicAlbum = {
	id: 7,
	title: 'Abbey Road',
	artist: 'The Beatles',
	artist_id: 3,
	year: 1969,
	track_count: 17,
	duration_seconds: 2800,
	has_cover: true
};

describe('AlbumCard', () => {
	it('muestra carátula, título, artista, año y pistas', () => {
		const { body } = render(AlbumCard, { props: { album } });

		expect(body).toContain('href="/music/albums/7"');
		expect(body).toContain('src="/music/cover/7"');
		expect(body).toContain('Abbey Road');
		expect(body).toContain('The Beatles');
		expect(body).toContain('1969');
		expect(body).toContain('17 pistas');
		expect(body).toContain('album-fallback');
		expect(body).toContain('AR');
	});

	it('escapa el texto de las etiquetas', () => {
		const { body } = render(AlbumCard, {
			props: {
				album: { ...album, title: '<script>alert(1)</script>', artist: 'A & B' }
			}
		});

		expect(body).not.toContain('<script>alert(1)</script>');
		expect(body).toContain('&lt;script');
		expect(body).toContain('A &amp; B');
	});
});
