import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
import type { UploadTargetRoot } from '$lib/api';
import MovieUploadDialog from './MovieUploadDialog.svelte';

const roots: UploadTargetRoot[] = [
	{
		library_id: 1,
		library_name: 'Películas',
		library_kind: 'video',
		path: '/media/peliculas',
		movie_count: 12,
		series_count: 0
	},
	{
		library_id: 2,
		library_name: 'Otra',
		library_kind: 'video',
		path: '/media/otra',
		movie_count: 3,
		series_count: 2
	},
	{
		library_id: 3,
		library_name: 'Clips',
		library_kind: 'clips',
		path: '/media/clips',
		movie_count: 50,
		series_count: 0
	}
];

describe('MovieUploadDialog', () => {
	it('muestra la subida guiada y elige la raíz con más películas', () => {
		const { body } = render(MovieUploadDialog, {
			props: { roots, allowed: true }
		});

		expect(body).toContain('Subir película');
		expect(body).toContain('Crear una carpeta para cada película');
		expect(body).toContain('type="checkbox"');
		expect(body).not.toMatch(/type="checkbox"[^>]*checked/);
		expect(body).toContain('No cierres la página hasta que terminen las subidas.');
		expect(body).toContain('/media/peliculas');
		expect(body).toContain('/media/otra');
		expect(body).toContain('value="/media/peliculas" selected');
		expect(body).not.toContain('/media/clips');
	});

	it('avisa si el ajuste de archivos está desactivado', () => {
		const { body } = render(MovieUploadDialog, {
			props: { roots, allowed: false }
		});
		expect(body).toContain('Administración → Biblioteca');
	});
});
