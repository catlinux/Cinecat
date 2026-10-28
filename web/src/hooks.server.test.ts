import { describe, expect, it, vi } from 'vitest';
import { handle } from './hooks.server';

function event(path: string, protocol = 'https:') {
	return {
		url: new URL(`${protocol}//localhost${path}`),
		cookies: { get: () => null },
		locals: {}
	};
}

async function run(path: string, response: Response, protocol = 'https:') {
	const resolve = vi.fn(async () => response);
	const result = await handle({
		event: event(path, protocol),
		resolve
	} as never);
	return result;
}

describe('hooks de seguridad', () => {
	it('conserva las cabeceras en la página de login', async () => {
		const response = await run(
			'/login',
			new Response('<html></html>', {
				status: 200,
				headers: { 'Content-Type': 'text/html; charset=utf-8' }
			})
		);

		expect(response.status).toBe(200);
		expect(response.headers.get('X-Content-Type-Options')).toBe('nosniff');
		expect(response.headers.get('X-Frame-Options')).toBe('DENY');
		expect(response.headers.get('Strict-Transport-Security')).toBe('max-age=31536000');
	});

	it('conserva las cabeceras en una redirección', async () => {
		const response = await run(
			'/admin',
			new Response(null, { status: 303, headers: { Location: '/login' } })
		);

		expect(response.status).toBe(303);
		expect(response.headers.get('Location')).toBe('/login');
		expect(response.headers.get('X-Frame-Options')).toBe('DENY');
		expect(response.headers.get('Content-Security-Policy')).toContain("frame-ancestors 'none'");
		expect(response.headers.get('Strict-Transport-Security')).toBe('max-age=31536000');
	});

	it('no pone HSTS si la petición es HTTP', async () => {
		const response = await run('/login', new Response('ok'), 'http:');
		expect(response.headers.get('Strict-Transport-Security')).toBeNull();
		expect(response.headers.get('X-Content-Type-Options')).toBe('nosniff');
	});
});
