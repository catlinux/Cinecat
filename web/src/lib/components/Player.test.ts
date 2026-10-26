import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
import { progressEnabled, resumeTarget } from '$lib/progress';
import Player from './Player.svelte';

describe('Player', () => {
	it('renderiza un <video> nativo con controles y la URL en modo directo', () => {
		const { body } = render(Player, {
			props: { src: 'http://api.test/api/v1/stream/1?exp=1&sig=abc', mode: 'direct', title: 'Matrix' }
		});

		expect(body).toContain('<video');
		expect(body).toContain('controls');
		expect(body).toContain('src="http://api.test/api/v1/stream/1?exp=1');
		expect(body).toContain('aria-label="Matrix"');
	});

	it('no pone src en el HTML del servidor en modo remux (lo decide el navegador)', () => {
		const { body } = render(Player, {
			props: { src: 'http://api.test/playlist.m3u8', mode: 'remux' }
		});

		expect(body).toContain('<video');
		expect(body).not.toContain('src="http://api.test/playlist.m3u8"');
	});
});

describe('lógica de reanudación y guardado', () => {
	it('reanuda una sola vez', () => {
		expect(resumeTarget(70, 3600, false)).toBe(70);
		expect(resumeTarget(70, 3600, true)).toBeNull();
	});

	it('no reanuda si queda menos de 15 segundos', () => {
		expect(resumeTarget(85, 100, false)).toBe(85);
		expect(resumeTarget(86, 100, false)).toBeNull();
	});

	it('no guarda nada sin un identificador válido', () => {
		expect(progressEnabled(undefined)).toBe(false);
		expect(progressEnabled(0)).toBe(false);
		expect(progressEnabled(7)).toBe(true);
	});
});
