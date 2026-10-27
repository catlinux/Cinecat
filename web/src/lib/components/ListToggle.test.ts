import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
import ListToggle from './ListToggle.svelte';

describe('ListToggle', () => {
	it('renderiza un formulario sin JavaScript con los campos y aria correctos', () => {
		const { body } = render(ListToggle, {
			props: { kind: 'item', id: 7, list: 'favorite', active: false }
		});

		expect(body).toContain('method="POST"');
		expect(body).toContain('action="/lists"');
		expect(body).toContain('name="kind" value="item"');
		expect(body).toContain('name="id" value="7"');
		expect(body).toContain('name="list" value="favorite"');
		expect(body).toContain('name="on" value="1"');
		expect(body).toContain('name="redirectTo"');
		expect(body).toContain('aria-label="Añadir a favoritos"');
		expect(body).toContain('aria-pressed="false"');
		expect(body).toContain('<svg');
	});

	it('cambia el valor y la etiqueta cuando está activo', () => {
		const { body } = render(ListToggle, {
			props: { kind: 'series', id: 3, list: 'watchlist', active: true, label: 'Próximas a ver' }
		});

		expect(body).toContain('name="on" value="0"');
		expect(body).toContain('aria-label="Quitar de próximas a ver"');
		expect(body).toContain('aria-pressed="true"');
		expect(body).toContain('list-toggle--active');
		expect(body).toContain('Próximas a ver');
	});

	it('usa el tamaño pequeño para las tarjetas', () => {
		const { body } = render(ListToggle, {
			props: { kind: 'collection', id: 8, list: 'favorite', active: false, size: 'small' }
		});
		expect(body).toContain('list-toggle-form--small');
		expect(body).toContain('list-toggle--small');
	});
});
