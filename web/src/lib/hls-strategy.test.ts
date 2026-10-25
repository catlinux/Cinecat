import { describe, expect, it } from 'vitest';
import { chooseHlsStrategy } from './hls-strategy';

describe('chooseHlsStrategy', () => {
	it('usa HLS nativo cuando el navegador lo soporta (Safari/iOS)', () => {
		expect(chooseHlsStrategy(true, true)).toBe('native');
		expect(chooseHlsStrategy(true, false)).toBe('native');
	});

	it('usa hls.js cuando no hay HLS nativo pero hls.js sí está soportado', () => {
		expect(chooseHlsStrategy(false, true)).toBe('hlsjs');
	});

	it('da error claro si no hay ninguna forma de reproducir HLS', () => {
		expect(chooseHlsStrategy(false, false)).toBe('unsupported');
	});
});
