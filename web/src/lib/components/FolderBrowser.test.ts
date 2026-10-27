import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
import type { FSBrowseResponse, FSRoot } from '$lib/api';
import FolderBrowser from './FolderBrowser.svelte';

const roots: FSRoot[] = [
	{ name: 'media', path: '/media' },
	{ name: 'discos', path: '/mnt/discos' }
];

function listing(overrides: Partial<FSBrowseResponse> = {}): FSBrowseResponse {
	return {
		path: '/media/peliculas',
		parent: '/media',
		entries: [
			{ name: 'Acción', path: '/media/peliculas/Acción' },
			{ name: 'Comedia', path: '/media/peliculas/Comedia' }
		],
		truncated: false,
		...overrides
	};
}

function body(props: Record<string, unknown>): string {
	return render(FolderBrowser, { props: props as never }).body;
}

describe('navegador de carpetas', () => {
	it('muestra las raíces al abrir', () => {
		const html = body({ open: true, roots });

		expect(html).toContain('Elegir carpeta');
		expect(html).toContain('media');
		expect(html).toContain('/media');
		expect(html).toContain('discos');
		expect(html).toContain('/mnt/discos');
	});

	it('muestra migas, carpetas, subir y elegir', () => {
		const html = body({ open: true, roots, listing: listing() });

		expect(html).toContain('media');
		expect(html).toContain('peliculas');
		expect(html).toContain('Acción');
		expect(html).toContain('/media/peliculas/Acción');
		expect(html).toContain('Subir');
		expect(html).toContain('Elegir esta carpeta');
	});

	it('indica cuando no hay subcarpetas', () => {
		const html = body({
			open: true,
			roots,
			listing: listing({ entries: [], path: '/media/vacia', parent: '/media' })
		});

		expect(html).toContain('No hay subcarpetas');
	});

	it('avisa cuando se trunca el listado', () => {
		const html = body({ open: true, roots, listing: listing({ truncated: true }) });

		expect(html).toContain('Se muestran las primeras 1000 carpetas');
	});

	it('muestra el error dentro del diálogo', () => {
		const html = body({ open: true, roots, error: 'No tienes permiso para esta carpeta.' });

		expect(html).toContain('role="alert"');
		expect(html).toContain('No tienes permiso para esta carpeta.');
	});

	it('escapa los nombres y las rutas', () => {
		const html = body({
			open: true,
			roots,
			listing: listing({
				entries: [
					{
						name: '<img src=x onerror=alert(1)>',
						path: '/media/<script>alert(1)</script>'
					}
				]
			})
		});

		expect(html).toContain('&lt;img src=x onerror=alert(1)>');
		expect(html).toContain('/media/&lt;script>alert(1)&lt;/script>');
		expect(html).not.toContain('<img src=x onerror=alert(1)>');
		expect(html).not.toContain('/media/<script>alert(1)</script>');
	});
});
