import { redirect } from '@sveltejs/kit';
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

// Devuelve el token de la sesión o manda a la página de login. Se usa en las
// cargas de las páginas protegidas.
export function requireSessionToken(locals: App.Locals): string {
	if (!locals.sessionToken) {
		redirect(303, '/login');
	}
	return locals.sessionToken;
}

// Acepta solo una ruta interna absoluta: evita redirecciones a otro origen,
// esquemas peligrosos y rutas con barras invertidas o caracteres de control.
export function safeNextPath(
	value: string | null | undefined,
	origin: string | URL
): string | null {
	if (typeof value !== 'string' || value === '') {
		return null;
	}
	if (!value.startsWith('/') || value.startsWith('//') || value.startsWith('/\\')) {
		return null;
	}
	if (value.includes('\\') || /[\u0000-\u001f\u007f-\u009f]/.test(value)) {
		return null;
	}
	try {
		const base = new URL(origin.toString());
		const target = new URL(value, base);
		if (target.origin !== base.origin) {
			return null;
		}
		return value;
	} catch {
		return null;
	}
}

function maxAgeFrom(expiresAt: string): number {
	const timestamp = Date.parse(expiresAt);
	if (Number.isNaN(timestamp)) {
		return DEFAULT_MAX_AGE;
	}
	return Math.max(0, Math.floor((timestamp - Date.now()) / 1000));
}
