import { describe, expect, it } from 'vitest';
import { withSecurityHeaders } from './security-headers';

function header(response: Response, name: string): string | null {
	return response.headers.get(name);
}

describe('cabeceras de seguridad', () => {
	it('añade todas las cabeceras por HTTP, sin HSTS', () => {
		const response = withSecurityHeaders(
			new Response('ok'),
			new URL('http://localhost/login')
		);

		expect(header(response, 'X-Content-Type-Options')).toBe('nosniff');
		expect(header(response, 'X-Frame-Options')).toBe('DENY');
		expect(header(response, 'Referrer-Policy')).toBe('same-origin');
		expect(header(response, 'Permissions-Policy')).toBe(
			'camera=(), microphone=(), geolocation=(), payment=()'
		);
		expect(header(response, 'Content-Security-Policy')).toBe(
			"frame-ancestors 'none'; object-src 'none'; base-uri 'self'; form-action 'self'"
		);
		expect(header(response, 'Strict-Transport-Security')).toBeNull();
	});

	it('añade HSTS solo cuando la petición es HTTPS', () => {
		const http = withSecurityHeaders(new Response('ok'), new URL('http://localhost/'));
		const https = withSecurityHeaders(new Response('ok'), new URL('https://localhost/'));

		expect(header(http, 'Strict-Transport-Security')).toBeNull();
		expect(header(https, 'Strict-Transport-Security')).toBe('max-age=31536000');
	});

	it('no pisa las cabeceras que ya puso una ruta', () => {
		const response = withSecurityHeaders(
			new Response('ok', {
				headers: {
					'X-Frame-Options': 'SAMEORIGIN',
					'Referrer-Policy': 'no-referrer'
				}
			}),
			new URL('https://localhost/')
		);

		expect(header(response, 'X-Frame-Options')).toBe('SAMEORIGIN');
		expect(header(response, 'Referrer-Policy')).toBe('no-referrer');
		expect(header(response, 'X-Content-Type-Options')).toBe('nosniff');
	});
});
