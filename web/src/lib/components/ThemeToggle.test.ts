import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
import ThemeToggle from './ThemeToggle.svelte';

describe('ThemeToggle', () => {
	it('renderiza las tres opciones con Auto activa por defecto', () => {
		const { body } = render(ThemeToggle);

		expect(body).toContain('Tema de la interfaz');
		expect(body).toContain('Auto');
		expect(body).toContain('Claro');
		expect(body).toContain('Oscuro');
		expect(body).toContain('aria-pressed="true"');
	});
});
