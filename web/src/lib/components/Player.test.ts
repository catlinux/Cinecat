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

	it('pinta los controles propios, las pistas y los subtítulos', () => {
		const { body } = render(Player, {
			props: {
				src: 'http://api.test/api/v1/stream/7?exp=1&sig=abc',
				mode: 'remux',
				title: 'El Padrino',
				itemId: 7,
				startAt: 0,
				nextEpisodeUrl: '/items/8/play',
				options: {
					url: '/api/v1/stream/7?exp=1&sig=abc',
					mode: 'remux',
					expires_at: '2026-10-26T12:00:00Z',
					audio_tracks: [
						{
							index: 0,
							language: 'spa',
							language_name: 'Español',
							codec: 'ac3',
							channels: 6,
							channel_layout: '5.1',
							title: '',
							default: true
						}
					],
					selected_audio: 0,
					qualities: [
						{ value: 'auto', label: 'Auto (hasta 1080p)' },
						{ value: 'original', label: 'Original' }
					],
					selected_quality: 'auto',
					subtitles: [
						{
							id: 'e0',
							source: 'embedded',
							language: 'spa',
							language_name: 'Español',
							title: '',
							codec: 'subrip',
							forced: false,
							default: false,
							url: '/api/v1/stream/7/subtitles/e0.vtt?exp=1&sig=abc'
						}
					],
					selected_subtitle: 'e0'
				}
			}
		});

		expect(body).toContain('aria-label="Reproducir"');
		expect(body).toContain('aria-label="Retroceder 10 segundos"');
		expect(body).toContain('Audio · Español');
		expect(body).toContain('Subtítulos');
		expect(body).toContain('Siguiente episodio');
		expect(body).toContain('<track');
		expect(body).toContain('kind="subtitles"');
		expect(body).toContain('src="http://api.test/api/v1/stream/7/subtitles/e0.vtt');
		expect(body).toContain('data-subtitle-id="e0"');
		expect(body).toContain('role="menuitemradio"');
		expect(body).toContain('aria-checked="true"');
		expect(body).toContain('Recodificado');
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
