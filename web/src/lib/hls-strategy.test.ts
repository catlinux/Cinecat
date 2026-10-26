import { describe, expect, it } from 'vitest';
import { chooseHlsStrategy } from './hls-strategy';

describe('chooseHlsStrategy', () => {
	it('usa hls.js siempre que esté soportado, aunque haya HLS nativo', () => {
		expect(chooseHlsStrategy(true, true)).toBe('hlsjs');
		expect(chooseHlsStrategy(false, true)).toBe('hlsjs');
	});

	it('usa HLS nativo solo si hls.js no está soportado (iPhone sin MSE)', () => {
		expect(chooseHlsStrategy(true, false)).toBe('native');
	});

	it('da error claro si no hay ninguna forma de reproducir HLS', () => {
		expect(chooseHlsStrategy(false, false)).toBe('unsupported');
	});
});
