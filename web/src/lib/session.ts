import type { Cookies } from '@sveltejs/kit';

export const SESSION_COOKIE = 'cinecat_session';

const DEFAULT_MAX_AGE = 24 * 60 * 60;

// La cookie la escribe siempre el servidor: httpOnly, SameSite=Lax y Secure
// cuando la petición llega por HTTPS. El token nunca pasa por JavaScript de
// cliente ni por localStorage.
export function setSessionCookie(
	cookies: Cookies,
	token: string,
	expiresAt: string,
	secure: boolean
): void {
	cookies.set(SESSION_COOKIE, token, {
		path: '/',
		httpOnly: true,
		sameSite: 'lax',
		secure,
		maxAge: maxAgeFrom(expiresAt)
	});
}

export function getSessionToken(cookies: Cookies): string | null {
	return cookies.get(SESSION_COOKIE) ?? null;
}

export function clearSessionCookie(cookies: Cookies, secure: boolean): void {
	cookies.delete(SESSION_COOKIE, { path: '/', secure });
}

function maxAgeFrom(expiresAt: string): number {
	const timestamp = Date.parse(expiresAt);
	if (Number.isNaN(timestamp)) {
		return DEFAULT_MAX_AGE;
	}
	return Math.max(0, Math.floor((timestamp - Date.now()) / 1000));
}
