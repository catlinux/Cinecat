import { createRawSnippet } from 'svelte';
import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
import RowScroller from './RowScroller.svelte';

const children = createRawSnippet(() => ({
	render: () => '<li>Contenido de la fila</li>'
}));

function renderRow(href: string | null = '/movies') {
	return render(RowScroller, {
		props: {
			title: 'Continuar viendo',
			kicker: 'En pausa',
			href,
			hrefLabel: 'Ver todas',
			children
		}
	}).body;
}

describe('fila desplazable', () => {
	it('muestra el título, el enlace y las flechas accesibles', () => {
		const body = renderRow();

		expect(body).toContain('Continuar viendo');
		expect(body).toContain('En pausa');
		expect(body).toContain('href="/movies"');
		expect(body).toContain('Ver todas →');
		expect(body).toContain('aria-label="Anterior"');
		expect(body).toContain('aria-label="Siguiente"');
		expect(body).toContain('type="button"');
		expect(body).toContain('role="region"');
		expect(body).toContain('aria-label="Continuar viendo"');
		expect(body).toContain('tabindex="0"');
		expect(body).toContain('Contenido de la fila');
	});

	it('oculta las flechas hasta que el cliente mide la fila', () => {
		const body = renderRow();

		expect(body).toMatch(/class="row-arrows[^"]*"[^>]*\bhidden\b/);
	});

	it('funciona sin enlace cuando la fila no tiene destino', () => {
		const body = renderRow(null);

		expect(body).not.toContain('Ver todas');
		expect(body).toContain('Continuar viendo');
	});
});
