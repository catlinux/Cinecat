import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
import type { Credits } from '$lib/api';
import CastRow from './CastRow.svelte';

function renderRow(props: {
	credits?: Credits | null;
	cast?: string[];
	directors?: string[];
}) {
	return render(CastRow, { props }).body;
}

const credits: Credits = {
	crew: [
		{ name: 'Ana Directora', job: 'Director', photo_url: 'https://images.example/ana.jpg' },
		{ name: 'Luis Guionista', job: 'Writer', photo_url: '' }
	],
	cast: [
		{ name: 'Mara Sol', character: 'Ripley', photo_url: '' },
		{ name: 'Pablo Extra', character: '', photo_url: 'https://images.example/pablo.jpg' }
	]
};

describe('reparto y equipo', () => {
	it('pone primero el reparto y después el equipo traducido', () => {
		const body = renderRow({ credits });

		expect(body).toContain('Reparto y equipo');
		expect(body).toContain('Ana Directora');
		expect(body).toContain('Dirección');
		expect(body).toContain('Luis Guionista');
		expect(body).toContain('Guion');
		expect(body).toContain('Mara Sol');
		expect(body).toContain('como Ripley');
		expect(body).toContain('Pablo Extra');
		expect(body).toContain('https://images.example/ana.jpg');
		expect(body).toContain('alt="Ana Directora"');
		expect(body).toContain('loading="lazy"');
		expect(body).toContain('class="row-fixed"');
		expect(body.indexOf('Mara Sol')).toBeLessThan(body.indexOf('Ana Directora'));
		expect(body.indexOf('Ana Directora')).toBeLessThan(body.indexOf('Luis Guionista'));
	});

	it('usa iniciales cuando no hay foto y omite el personaje vacío', () => {
		const body = renderRow({ credits });

		expect(body).toContain('>LG<');
		expect(body).toContain('>MS<');
		expect((body.match(/como Ripley/g) ?? []).length).toBe(1);
		expect(body).not.toContain('como </p>');
	});

	it('completa el reparto con nombres si faltan los créditos de reparto', () => {
		const body = renderRow({
			credits: {
				crew: [{ name: 'Ana Directora', job: 'Director', photo_url: '' }],
				cast: []
			},
			cast: ['Marlon Brando']
		});

		expect(body).toContain('Ana Directora');
		expect(body).toContain('Marlon Brando');
		expect(body).toContain('>MB<');
		expect(body.indexOf('Marlon Brando')).toBeLessThan(body.indexOf('Ana Directora'));
	});

	it('mantiene los nombres de respaldo si todavía no hay créditos', () => {
		const body = renderRow({
			credits: { crew: [], cast: [] },
			directors: ['Francis Ford Coppola'],
			cast: ['Marlon Brando']
		});

		expect(body).toContain('Francis Ford Coppola');
		expect(body).toContain('Dirección');
		expect(body).toContain('Marlon Brando');
		expect(body).toContain('>MB<');
		expect(body.indexOf('Marlon Brando')).toBeLessThan(body.indexOf('Francis Ford Coppola'));
	});

	it('no dibuja la fila si no hay nadie que mostrar', () => {
		const body = renderRow({ credits: { crew: [], cast: [] }, cast: [], directors: [] });

		expect(body).not.toContain('Reparto y equipo');
	});
});
