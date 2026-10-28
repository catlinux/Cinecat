const SECURITY_HEADERS: ReadonlyArray<readonly [string, string]> = [
	['X-Content-Type-Options', 'nosniff'],
	['X-Frame-Options', 'DENY'],
	['Referrer-Policy', 'same-origin'],
	['Permissions-Policy', 'camera=(), microphone=(), geolocation=(), payment=()'],
	[
		'Content-Security-Policy',
		"frame-ancestors 'none'; object-src 'none'; base-uri 'self'; form-action 'self'"
	]
];

// Añade las cabeceras de seguridad sin pisar las que ya haya puesto una ruta.
// La CSP es deliberadamente mínima: script-src, default-src e img-src quedan
// para una revisión posterior con navegadores reales.
export function withSecurityHeaders(response: Response, url: URL): Response {
	const headers = response.headers;
	for (const [name, value] of SECURITY_HEADERS) {
		if (!headers.has(name)) {
			headers.set(name, value);
		}
	}
	if (
		url.protocol === 'https:' &&
		!headers.has('Strict-Transport-Security')
	) {
		headers.set('Strict-Transport-Security', 'max-age=31536000');
	}
	return response;
}
