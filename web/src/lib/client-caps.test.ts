import { describe, expect, it } from 'vitest';
import {
	HEVC10_MIME,
	HEVC_MIME,
	capsCookie,
	capsFromCookie,
	detectClientCapabilities,
	sanitizeClientCapabilities
} from './client-caps';

describe('capacidades del cliente', () => {
	it('detecta HEVC y HEVC 10 bits por cualquiera de las dos comprobaciones', () => {
		expect(detectClientCapabilities((type) => type === HEVC_MIME, () => '')).toBe('hevc');
		expect(detectClientCapabilities(() => false, (type) => (type === HEVC_MIME ? 'probably' : ''))).toBe(
			'hevc'
		);
		expect(detectClientCapabilities(() => false, (type) => (type === HEVC10_MIME ? 'maybe' : ''))).toBe(
			'hevc,hevc10'
		);
		expect(detectClientCapabilities(() => false, () => '')).toBe('');
	});

	it('sanea la lista de capacidades y hevc10 implica hevc', () => {
		expect(sanitizeClientCapabilities('HEVC10, otro ,hevc')).toBe('hevc,hevc10');
		expect(sanitizeClientCapabilities('otro')).toBe('');
		expect(sanitizeClientCapabilities(null)).toBe('');
	});

	it('lee y escribe la cookie de capacidades', () => {
		expect(capsFromCookie('otra=1; cinecat_caps=hevc10%2Cdesconocido')).toBe('hevc,hevc10');
		expect(capsFromCookie('cinecat_caps=hevc')).toBe('hevc');
		expect(capsFromCookie('')).toBe('');
		expect(capsFromCookie('cinecat_caps=%')).toBe('');
		const cookie = capsCookie('hevc10');
		expect(cookie).toContain('cinecat_caps=hevc,hevc10');
		expect(cookie).toContain('path=/');
		expect(cookie).toContain('max-age=31536000');
		expect(cookie).toContain('SameSite=Lax');
	});
});
