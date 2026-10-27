import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
import type { Collection } from '$lib/api';
import CollectionCard from './CollectionCard.svelte';

const collection: Collection = {
	id: 4,
	name: 'Saga',
	overview: null,
	poster_url: null,
	backdrop_url: null,
	item_count: 3,
	source: 'manual',
	edited: false
};

describe('CollectionCard', () => {
	it('muestra la colección y el enlace', () => {
		const { body } = render(CollectionCard, { props: { collection } });
		expect(body).toContain('Saga');
		expect(body).toContain('href="/collections/4"');
		expect(body).toContain('3 películas');
	});
	it('pone el botón de favorito fuera del enlace', () => {
		const { body } = render(CollectionCard, {
			props: { collection: { ...collection, favorite: true } }
		});
		expect(body).toContain('action="/lists"');
		expect(body).toContain('aria-label="Quitar de favoritos"');
		expect(body.indexOf('</a>')).toBeLessThan(body.indexOf('<form'));
	});
});
