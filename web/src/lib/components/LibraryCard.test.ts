import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
import type { Library } from '$lib/api';
import LibraryCard from './LibraryCard.svelte';

const library: Library = {
	id: 4,
	name: 'Peliculas',
	kind: 'video',
	path: '/media/peliculas',
	paths: [
		{ id: 10, path: '/media/peliculas' },
		{ id: 11, path: '/mnt/otro-disco/peliculas' }
	],
	item_count: 12
};

function body(props: Record<string, unknown>): string {
	return render(LibraryCard, { props: props as never }).body;
}

describe('tarjeta de biblioteca', () => {
	it('muestra nombre, tipo, fichas y todas las carpetas', () => {
		const html = body({ library });

		expect(html).toContain('Peliculas');
		expect(html).toContain('Películas y series');
		expect(html).toContain('12 fichas');
		expect(html).toContain('/media/peliculas');
		expect(html).toContain('/mnt/otro-disco/peliculas');
		expect(html).toContain('Añadir carpeta');
		expect(html).toContain('Renombrar');
		expect(html).toContain('Escanear');
		expect(html).toContain('Borrar biblioteca');
	});

	it('incluye la confirmación de quitar carpeta y de borrar biblioteca', () => {
		const html = body({ library });

		expect(html).toContain('Se quitarán de Cinecat las fichas de esta carpeta');
		expect(html).toContain('los archivos del disco no se tocan');
		expect(html).toContain('Escribe «Peliculas» para confirmar');
		expect(html).toContain('action="?/removePath"');
		expect(html).toContain('action="?/delete"');
	});

	it('conserva el nombre y la ruta escritos tras un error', () => {
		const html = body({
			library,
			form: {
				action: 'rename',
				libraryId: 4,
				renameName: 'Cine',
				message: 'No se pudo renombrar.'
			}
		});

		expect(html).toContain('value="Cine"');
	});
});
