import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
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
