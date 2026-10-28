import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
import { movePortalRow, type PortalRow } from '$lib/settings';
import PortalRowsEditor from './PortalRowsEditor.svelte';

const rows: PortalRow[] = [
	{ id: 'recent_movies', visible: true, title: 'Recientes', kicker: 'Estrenos' },
	{ id: 'continue_movies', visible: false, title: 'Seguir', kicker: 'En pausa' },
	{ id: 'favorites', visible: true, title: 'Favoritos', kicker: 'Selección' }
];

function hiddenRows(body: string): PortalRow[] {
	const match = body.match(/name="portal_rows" value="([^"]*)"/);
	if (!match) {
		throw new Error('Falta el JSON de las filas');
	}
	return JSON.parse(match[1].replaceAll('&quot;', '"').replaceAll('&amp;', '&'));
}

describe('PortalRowsEditor', () => {
	it('pinta los campos, la visibilidad y envía el JSON correcto', () => {
		const body = render(PortalRowsEditor, { props: { rows } }).body;

		expect(body).toContain('action="?/portalRows"');
		expect(body).toContain('name="row_count" value="3"');
		expect(hiddenRows(body)).toEqual(rows);
		expect(body).toContain('name="row_id_0" value="recent_movies"');
		expect(body).toContain('name="row_title_0" value="Recientes"');
		expect(body).toContain('name="row_kicker_0" value="Estrenos"');
		expect(body).toContain('name="row_visible_1"');
		expect(body).toContain('maxlength="60"');
		expect(body).toContain('maxlength="40"');

		const secondRow = body.slice(
			body.indexOf('name="row_id_1"'),
			body.indexOf('</fieldset>', body.indexOf('name="row_id_1"'))
		);
		expect(secondRow).not.toContain('checked');

		expect(body).toContain('aria-label="Subir Recientes"');
		expect(body).toContain('aria-label="Bajar Favoritos"');
		expect(body).toContain('aria-live="polite"');
		expect(body).toContain('Guardar filas');
		expect(body).toContain('Restablecer por defecto');
		expect(body).toContain('name="key" value="ui.portal_rows"');
	});

	it('deshabilita Subir y Bajar en los extremos', () => {
		const body = render(PortalRowsEditor, { props: { rows } }).body;

		expect(body).toMatch(/value="up:recent_movies" disabled=""/);
		expect(body).toMatch(/value="down:favorites" disabled=""/);
		expect(body).not.toMatch(/value="down:recent_movies" disabled=""/);
		expect(body).not.toMatch(/value="up:favorites" disabled=""/);
	});

	it('muestra los mensajes de éxito y de error', () => {
		const ok = render(PortalRowsEditor, {
			props: {
				rows,
				form: { portalRowsAction: true, success: true, message: 'Filas del portal guardadas.' }
			}
		}).body;
		expect(ok).toContain('role="status"');
		expect(ok).toContain('Filas del portal guardadas.');

		const error = render(PortalRowsEditor, {
			props: {
				rows,
				form: { portalRowsAction: true, success: false, message: 'filas no válidas' }
			}
		}).body;
		expect(error).toContain('role="alert"');
		expect(error).toContain('filas no válidas');
	});

	it('mueve filas conservando el resto del orden', () => {
		const moved = movePortalRow(rows, 'favorites', -1);
		expect(moved?.map((row) => row.id)).toEqual([
			'recent_movies',
			'favorites',
			'continue_movies'
		]);
		expect(rows.map((row) => row.id)).toEqual([
			'recent_movies',
			'continue_movies',
			'favorites'
		]);
		expect(movePortalRow(rows, 'recent_movies', -1)).toBeNull();
		expect(movePortalRow(rows, 'favorites', 1)).toBeNull();
	});
});
