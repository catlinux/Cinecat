import type { Cookies } from '@sveltejs/kit';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
	SESSION_COOKIE,
	clearSessionCookie,
	getSessionToken,
	safeNextPath,
	setSessionCookie
} from './session';

function fakeCookies(getValue: string | undefined = undefined) {
	const get = vi.fn(() => getValue);
	const set = vi.fn();
	const remove = vi.fn();
	const cookies = { get, set, delete: remove } as unknown as Cookies;
	return { cookies, get, set, remove };
}

describe('sesión', () => {
	beforeEach(() => {
		vi.useRealTimers();
	});

	it('lee el token de la cookie', () => {
		const { cookies } = fakeCookies('token-123');
		expect(getSessionToken(cookies)).toBe('token-123');
	});

	it('devuelve null si no hay cookie', () => {
		const { cookies } = fakeCookies();
		expect(getSessionToken(cookies)).toBeNull();
	});

	it('fija una cookie httpOnly con SameSite=Lax y Secure según HTTPS', () => {
		const { cookies, set } = fakeCookies();

		setSessionCookie(cookies, 'token-123', new Date(Date.now() + 3600_000).toISOString(), true);

		expect(set).toHaveBeenCalledTimes(1);
		const [name, value, options] = set.mock.calls[0];
		expect(name).toBe(SESSION_COOKIE);
		expect(value).toBe('token-123');
		expect(options).toMatchObject({
			path: '/',
			httpOnly: true,
			sameSite: 'lax',
			secure: true
		});
		expect(options.maxAge).toBeGreaterThan(3000);
		expect(options.maxAge).toBeLessThanOrEqual(3600);
	});

	it('no marca Secure cuando la petición es HTTP', () => {
		const { cookies, set } = fakeCookies();

		setSessionCookie(cookies, 'token-123', '2026-10-26T12:00:00Z', false);

		expect(set.mock.calls[0][2].secure).toBe(false);
	});

	it('borra la cookie de sesión con la misma ruta', () => {
		const { cookies, remove } = fakeCookies('token-123');

		clearSessionCookie(cookies, true);

		expect(remove).toHaveBeenCalledWith(SESSION_COOKIE, { path: '/', secure: true });
	});
});

describe('safeNextPath', () => {
	const origin = 'https://cinecat.test';

	it('acepta rutas internas con query', () => {
		expect(safeNextPath('/vincular?codigo=AB', origin)).toBe('/vincular?codigo=AB');
		expect(safeNextPath('/%2F%2Fevil.com', origin)).toBe('/%2F%2Fevil.com');
	});

	it('rechaza otros orígenes, esquemas y barras peligrosas', () => {
		expect(safeNextPath('//evil.com', origin)).toBeNull();
		expect(safeNextPath('/\\evil.com', origin)).toBeNull();
		expect(safeNextPath('https://evil.com', origin)).toBeNull();
		expect(safeNextPath('javascript:alert(1)', origin)).toBeNull();
		expect(safeNextPath('', origin)).toBeNull();
	});

	it('rechaza caracteres de control', () => {
		expect(safeNextPath('/ruta\nfalsa', origin)).toBeNull();
		expect(safeNextPath('/ruta\u0000falsa', origin)).toBeNull();
	});
});
