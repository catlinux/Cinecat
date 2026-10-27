import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
import FolderTile from './FolderTile.svelte';

describe('loseta de carpeta', () => {
	it('muestra nombre, recuento y enlace', () => {
		const { body } = render(FolderTile, {
			props: {
				name: 'Conciertos',
				itemCount: 12,
				href: '/videos/4?root=9&path=Conciertos'
			}
		});

		expect(body).toContain('class="folder-tile');
		expect(body).toContain('href="/videos/4?root=9&amp;path=Conciertos"');
		expect(body).toContain('Conciertos');
		expect(body).toContain('12 vídeos');
		expect(body).toContain('folder-tile-art');
		expect(body).not.toContain('/thumb');
	});

	it('usa la miniatura del primer vídeo como portada atenuada', () => {
		const { body } = render(FolderTile, {
			props: {
				name: 'Conciertos',
				itemCount: 12,
				href: '/videos/4?root=9&path=Conciertos',
				coverItemId: 21
			}
		});

		expect(body).toContain('src="/items/21/thumb"');
		expect(body).toContain('loading="lazy"');
		expect(body).toContain('decoding="async"');
		expect(body).toContain('folder-tile-shade');
		expect(body).toContain('folder-tile--cover');
		expect(body).not.toContain('folder-tile-art');
	});

	it('escapa el nombre de la carpeta', () => {
		const { body } = render(FolderTile, {
			props: {
				name: '<img src=x onerror=alert(1)>',
				itemCount: 1,
				href: '/videos/4'
			}
		});

		expect(body).toContain('&lt;img src=x onerror=alert(1)>');
		expect(body).not.toContain('<img src=x onerror=alert(1)>');
		expect(body).toContain('1 vídeo');
	});
});
